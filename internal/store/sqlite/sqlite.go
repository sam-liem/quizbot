package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sam-liem/quizbot/internal/model"
	"github.com/sam-liem/quizbot/internal/store"
	_ "modernc.org/sqlite" // register the sqlite driver
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Compile-time assertion that *DB implements store.Repository.
var _ store.Repository = (*DB)(nil)

// DB wraps a sql.DB and implements store.Repository.
type DB struct {
	db *sql.DB
}

// Open opens a SQLite database at dsn, configures WAL mode, and runs migrations.
func Open(dsn string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}

	if _, err = sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	if err = runMigrations(sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &DB{db: sqlDB}, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// runMigrations executes all embedded SQL migration files in order.
func runMigrations(db *sql.DB) error {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("reading migrations dir: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := migrationFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", entry.Name(), err)
		}
		if _, err = db.Exec(string(data)); err != nil {
			return fmt.Errorf("executing migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// ---- helpers ----

func marshalJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalJSON(s string, v any) error {
	return json.Unmarshal([]byte(s), v)
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing time %q: %w", s, err)
	}
	return t.UTC(), nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func intToBool(i int) bool {
	return i != 0
}

// ---- Quiz Packs ----

func (d *DB) SaveQuizPack(ctx context.Context, pack model.QuizPack) error {
	data, err := marshalJSON(pack)
	if err != nil {
		return fmt.Errorf("marshalling quiz pack: %w", err)
	}

	_, err = d.db.ExecContext(ctx, `
		INSERT INTO quiz_packs (id, name, description, version, data)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name        = excluded.name,
			description = excluded.description,
			version     = excluded.version,
			data        = excluded.data`,
		pack.ID, pack.Name, pack.Description, pack.Version, data)
	if err != nil {
		return fmt.Errorf("saving quiz pack: %w", err)
	}
	return nil
}

func (d *DB) GetQuizPack(ctx context.Context, packID string) (*model.QuizPack, error) {
	var data string
	err := d.db.QueryRowContext(ctx,
		`SELECT data FROM quiz_packs WHERE id = ?`, packID).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting quiz pack: %w", err)
	}

	var pack model.QuizPack
	if err = unmarshalJSON(data, &pack); err != nil {
		return nil, fmt.Errorf("unmarshalling quiz pack: %w", err)
	}
	return &pack, nil
}

func (d *DB) ListQuizPacks(ctx context.Context, _ string) ([]model.QuizPack, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT data FROM quiz_packs ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("listing quiz packs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var packs []model.QuizPack
	for rows.Next() {
		var data string
		if err = rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("scanning quiz pack row: %w", err)
		}
		var pack model.QuizPack
		if err = unmarshalJSON(data, &pack); err != nil {
			return nil, fmt.Errorf("unmarshalling quiz pack: %w", err)
		}
		packs = append(packs, pack)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating quiz pack rows: %w", err)
	}
	return packs, nil
}

// ---- Question State ----

func (d *DB) GetQuestionState(ctx context.Context, userID, packID, questionID string) (*model.QuestionState, error) {
	var (
		easeFactor      float64
		intervalDays    float64
		repetitionCount int
		nextReviewAt    string
		lastResult      string
		lastReviewedAt  string
	)

	err := d.db.QueryRowContext(ctx, `
		SELECT ease_factor, interval_days, repetition_count,
		       next_review_at, last_result, last_reviewed_at
		FROM question_states
		WHERE user_id = ? AND pack_id = ? AND question_id = ?`,
		userID, packID, questionID,
	).Scan(&easeFactor, &intervalDays, &repetitionCount, &nextReviewAt, &lastResult, &lastReviewedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting question state: %w", err)
	}

	nextReview, err := parseTime(nextReviewAt)
	if err != nil {
		return nil, fmt.Errorf("parsing next_review_at: %w", err)
	}
	lastReview, err := parseTime(lastReviewedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing last_reviewed_at: %w", err)
	}

	return &model.QuestionState{
		UserID:          userID,
		PackID:          packID,
		QuestionID:      questionID,
		EaseFactor:      easeFactor,
		IntervalDays:    intervalDays,
		RepetitionCount: repetitionCount,
		NextReviewAt:    nextReview,
		LastResult:      model.AnswerResult(lastResult),
		LastReviewedAt:  lastReview,
	}, nil
}

func (d *DB) UpdateQuestionState(ctx context.Context, state model.QuestionState) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO question_states
			(user_id, pack_id, question_id, ease_factor, interval_days,
			 repetition_count, next_review_at, last_result, last_reviewed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, pack_id, question_id) DO UPDATE SET
			ease_factor      = excluded.ease_factor,
			interval_days    = excluded.interval_days,
			repetition_count = excluded.repetition_count,
			next_review_at   = excluded.next_review_at,
			last_result      = excluded.last_result,
			last_reviewed_at = excluded.last_reviewed_at`,
		state.UserID, state.PackID, state.QuestionID,
		state.EaseFactor, state.IntervalDays, state.RepetitionCount,
		formatTime(state.NextReviewAt), string(state.LastResult),
		formatTime(state.LastReviewedAt),
	)
	if err != nil {
		return fmt.Errorf("updating question state: %w", err)
	}
	return nil
}

// ListQuestionStates returns question states for a user filtered and sorted
// according to filter. PackID is required. Result filtering and sorting are
// applied in SQL; TopicID filtering is not implemented here (v1 simplification:
// question_states has no topic_id column — callers must cross-reference pack
// data themselves).
func (d *DB) ListQuestionStates(ctx context.Context, userID string, filter model.QuestionHistoryFilter) ([]model.QuestionState, error) {
	query := `
		SELECT user_id, pack_id, question_id, ease_factor, interval_days,
		       repetition_count, next_review_at, last_result, last_reviewed_at
		FROM question_states
		WHERE user_id = ? AND pack_id = ?`
	args := []any{userID, filter.PackID}

	if filter.Result != "" {
		query += " AND last_result = ?"
		args = append(args, string(filter.Result))
	}

	// Determine ORDER BY column. Validated against an allow-list to prevent
	// SQL injection from caller-supplied SortBy strings.
	var orderCol string
	switch filter.SortBy {
	case "next_review":
		orderCol = "next_review_at"
	case "ease_factor":
		orderCol = "ease_factor"
	default:
		orderCol = "last_reviewed_at"
	}

	direction := "ASC"
	if filter.SortDesc {
		direction = "DESC"
	}
	query += " ORDER BY " + orderCol + " " + direction

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing question states: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []model.QuestionState
	for rows.Next() {
		var (
			uid             string
			packID          string
			questionID      string
			easeFactor      float64
			intervalDays    float64
			repetitionCount int
			nextReviewAt    string
			lastResult      string
			lastReviewedAt  string
		)
		if err = rows.Scan(&uid, &packID, &questionID,
			&easeFactor, &intervalDays, &repetitionCount,
			&nextReviewAt, &lastResult, &lastReviewedAt); err != nil {
			return nil, fmt.Errorf("scanning question state row: %w", err)
		}
		nextReview, err := parseTime(nextReviewAt)
		if err != nil {
			return nil, fmt.Errorf("parsing next_review_at: %w", err)
		}
		lastReview, err := parseTime(lastReviewedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing last_reviewed_at: %w", err)
		}
		result = append(result, model.QuestionState{
			UserID:          uid,
			PackID:          packID,
			QuestionID:      questionID,
			EaseFactor:      easeFactor,
			IntervalDays:    intervalDays,
			RepetitionCount: repetitionCount,
			NextReviewAt:    nextReview,
			LastResult:      model.AnswerResult(lastResult),
			LastReviewedAt:  lastReview,
		})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating question state rows: %w", err)
	}
	return result, nil
}

// ---- Topic Stats ----

func (d *DB) GetTopicStats(ctx context.Context, userID, packID, topicID string) (*model.TopicStats, error) {
	var stats model.TopicStats
	err := d.db.QueryRowContext(ctx, `
		SELECT user_id, pack_id, topic_id, total_attempts, correct_count,
		       rolling_accuracy, current_streak, best_streak
		FROM topic_stats
		WHERE user_id = ? AND pack_id = ? AND topic_id = ?`,
		userID, packID, topicID,
	).Scan(&stats.UserID, &stats.PackID, &stats.TopicID,
		&stats.TotalAttempts, &stats.CorrectCount,
		&stats.RollingAccuracy, &stats.CurrentStreak, &stats.BestStreak)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting topic stats: %w", err)
	}
	return &stats, nil
}

func (d *DB) UpdateTopicStats(ctx context.Context, stats model.TopicStats) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO topic_stats
			(user_id, pack_id, topic_id, total_attempts, correct_count,
			 rolling_accuracy, current_streak, best_streak)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, pack_id, topic_id) DO UPDATE SET
			total_attempts   = excluded.total_attempts,
			correct_count    = excluded.correct_count,
			rolling_accuracy = excluded.rolling_accuracy,
			current_streak   = excluded.current_streak,
			best_streak      = excluded.best_streak`,
		stats.UserID, stats.PackID, stats.TopicID,
		stats.TotalAttempts, stats.CorrectCount,
		stats.RollingAccuracy, stats.CurrentStreak, stats.BestStreak,
	)
	if err != nil {
		return fmt.Errorf("updating topic stats: %w", err)
	}
	return nil
}

func (d *DB) ListTopicStats(ctx context.Context, userID, packID string) ([]model.TopicStats, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT user_id, pack_id, topic_id, total_attempts, correct_count,
		       rolling_accuracy, current_streak, best_streak
		FROM topic_stats
		WHERE user_id = ? AND pack_id = ?
		ORDER BY topic_id`,
		userID, packID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing topic stats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []model.TopicStats
	for rows.Next() {
		var s model.TopicStats
		if err = rows.Scan(&s.UserID, &s.PackID, &s.TopicID,
			&s.TotalAttempts, &s.CorrectCount,
			&s.RollingAccuracy, &s.CurrentStreak, &s.BestStreak); err != nil {
			return nil, fmt.Errorf("scanning topic stats row: %w", err)
		}
		result = append(result, s)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating topic stats rows: %w", err)
	}
	return result, nil
}

// ---- Study Sessions ----

func (d *DB) CreateSession(ctx context.Context, session model.StudySession) error {
	attemptsJSON, err := marshalJSON(session.Attempts)
	if err != nil {
		return fmt.Errorf("marshalling attempts: %w", err)
	}

	var endedAt *string
	if session.EndedAt != nil {
		s := formatTime(*session.EndedAt)
		endedAt = &s
	}

	_, err = d.db.ExecContext(ctx, `
		INSERT INTO study_sessions
			(id, user_id, pack_id, mode, started_at, ended_at,
			 question_count, correct_count, attempts)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.PackID, string(session.Mode),
		formatTime(session.StartedAt), endedAt,
		session.QuestionCount, session.CorrectCount, attemptsJSON,
	)
	if err != nil {
		return fmt.Errorf("creating study session: %w", err)
	}
	return nil
}

func (d *DB) GetSession(ctx context.Context, userID, sessionID string) (*model.StudySession, error) {
	var (
		id            string
		uid           string
		packID        string
		mode          string
		startedAt     string
		endedAt       sql.NullString
		questionCount int
		correctCount  int
		attemptsJSON  string
	)

	err := d.db.QueryRowContext(ctx, `
		SELECT id, user_id, pack_id, mode, started_at, ended_at,
		       question_count, correct_count, attempts
		FROM study_sessions
		WHERE user_id = ? AND id = ?`,
		userID, sessionID,
	).Scan(&id, &uid, &packID, &mode, &startedAt, &endedAt,
		&questionCount, &correctCount, &attemptsJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting study session: %w", err)
	}

	started, err := parseTime(startedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing started_at: %w", err)
	}

	session := &model.StudySession{
		ID:            id,
		UserID:        uid,
		PackID:        packID,
		Mode:          model.SessionMode(mode),
		StartedAt:     started,
		QuestionCount: questionCount,
		CorrectCount:  correctCount,
	}

	if endedAt.Valid {
		t, err := parseTime(endedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parsing ended_at: %w", err)
		}
		session.EndedAt = &t
	}

	var attempts []model.QuestionAttempt
	if err = unmarshalJSON(attemptsJSON, &attempts); err != nil {
		return nil, fmt.Errorf("unmarshalling attempts: %w", err)
	}
	session.Attempts = attempts

	return session, nil
}

func (d *DB) UpdateSession(ctx context.Context, session model.StudySession) error {
	attemptsJSON, err := marshalJSON(session.Attempts)
	if err != nil {
		return fmt.Errorf("marshalling attempts: %w", err)
	}

	var endedAt *string
	if session.EndedAt != nil {
		s := formatTime(*session.EndedAt)
		endedAt = &s
	}

	_, err = d.db.ExecContext(ctx, `
		UPDATE study_sessions
		SET mode           = ?,
		    started_at     = ?,
		    ended_at       = ?,
		    question_count = ?,
		    correct_count  = ?,
		    attempts       = ?
		WHERE user_id = ? AND id = ?`,
		string(session.Mode), formatTime(session.StartedAt), endedAt,
		session.QuestionCount, session.CorrectCount, attemptsJSON,
		session.UserID, session.ID,
	)
	if err != nil {
		return fmt.Errorf("updating study session: %w", err)
	}
	return nil
}

// ---- Quiz Sessions ----

func (d *DB) SaveQuizSession(ctx context.Context, session model.QuizSession) error {
	questionIDsJSON, err := marshalJSON(session.QuestionIDs)
	if err != nil {
		return fmt.Errorf("marshalling question_ids: %w", err)
	}
	answersJSON, err := marshalJSON(session.Answers)
	if err != nil {
		return fmt.Errorf("marshalling answers: %w", err)
	}

	_, err = d.db.ExecContext(ctx, `
		INSERT INTO quiz_sessions
			(id, user_id, pack_id, mode, question_ids, current_index,
			 answers, started_at, time_limit_sec, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			mode          = excluded.mode,
			question_ids  = excluded.question_ids,
			current_index = excluded.current_index,
			answers       = excluded.answers,
			started_at    = excluded.started_at,
			time_limit_sec = excluded.time_limit_sec,
			status        = excluded.status`,
		session.ID, session.UserID, session.PackID, string(session.Mode),
		questionIDsJSON, session.CurrentIndex,
		answersJSON, formatTime(session.StartedAt),
		session.TimeLimitSec, string(session.Status),
	)
	if err != nil {
		return fmt.Errorf("saving quiz session: %w", err)
	}
	return nil
}

func (d *DB) GetQuizSession(ctx context.Context, userID, sessionID string) (*model.QuizSession, error) {
	var (
		id              string
		uid             string
		packID          string
		mode            string
		questionIDsJSON string
		currentIndex    int
		answersJSON     string
		startedAt       string
		timeLimitSec    int
		status          string
	)

	err := d.db.QueryRowContext(ctx, `
		SELECT id, user_id, pack_id, mode, question_ids, current_index,
		       answers, started_at, time_limit_sec, status
		FROM quiz_sessions
		WHERE user_id = ? AND id = ?`,
		userID, sessionID,
	).Scan(&id, &uid, &packID, &mode, &questionIDsJSON, &currentIndex,
		&answersJSON, &startedAt, &timeLimitSec, &status)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting quiz session: %w", err)
	}

	started, err := parseTime(startedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing started_at: %w", err)
	}

	var questionIDs []string
	if err = unmarshalJSON(questionIDsJSON, &questionIDs); err != nil {
		return nil, fmt.Errorf("unmarshalling question_ids: %w", err)
	}

	var answers map[string]int
	if err = unmarshalJSON(answersJSON, &answers); err != nil {
		return nil, fmt.Errorf("unmarshalling answers: %w", err)
	}
	if answers == nil {
		answers = map[string]int{}
	}

	return &model.QuizSession{
		ID:           id,
		UserID:       uid,
		PackID:       packID,
		Mode:         model.SessionMode(mode),
		QuestionIDs:  questionIDs,
		CurrentIndex: currentIndex,
		Answers:      answers,
		StartedAt:    started,
		TimeLimitSec: timeLimitSec,
		Status:       model.QuizSessionStatus(status),
	}, nil
}

// ListQuizSessions returns all quiz sessions for the given user with the given status,
// ordered by started_at DESC (most recent first).
func (d *DB) ListQuizSessions(ctx context.Context, userID string, status model.QuizSessionStatus) ([]model.QuizSession, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, user_id, pack_id, mode, question_ids, current_index,
		       answers, started_at, time_limit_sec, status
		FROM quiz_sessions
		WHERE user_id = ? AND status = ?
		ORDER BY started_at DESC`,
		userID, string(status),
	)
	if err != nil {
		return nil, fmt.Errorf("listing quiz sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var sessions []model.QuizSession
	for rows.Next() {
		var (
			id              string
			uid             string
			packID          string
			mode            string
			questionIDsJSON string
			currentIndex    int
			answersJSON     string
			startedAt       string
			timeLimitSec    int
			st              string
		)
		if err = rows.Scan(&id, &uid, &packID, &mode, &questionIDsJSON, &currentIndex,
			&answersJSON, &startedAt, &timeLimitSec, &st); err != nil {
			return nil, fmt.Errorf("scanning quiz session row: %w", err)
		}

		started, err := parseTime(startedAt)
		if err != nil {
			return nil, fmt.Errorf("parsing started_at: %w", err)
		}

		var questionIDs []string
		if err = unmarshalJSON(questionIDsJSON, &questionIDs); err != nil {
			return nil, fmt.Errorf("unmarshalling question_ids: %w", err)
		}

		var answers map[string]int
		if err = unmarshalJSON(answersJSON, &answers); err != nil {
			return nil, fmt.Errorf("unmarshalling answers: %w", err)
		}
		if answers == nil {
			answers = map[string]int{}
		}

		sessions = append(sessions, model.QuizSession{
			ID:           id,
			UserID:       uid,
			PackID:       packID,
			Mode:         model.SessionMode(mode),
			QuestionIDs:  questionIDs,
			CurrentIndex: currentIndex,
			Answers:      answers,
			StartedAt:    started,
			TimeLimitSec: timeLimitSec,
			Status:       model.QuizSessionStatus(st),
		})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating quiz session rows: %w", err)
	}
	return sessions, nil
}

// ---- User Preferences ----

func (d *DB) GetPreferences(ctx context.Context, userID string) (*model.UserPreferences, error) {
	var (
		deliveryIntervalMin  int
		maxUnanswered        int
		activePackIDsJSON    string
		focusMode            string
		notifyInactivity     int
		notifyInactivityDays int
		notifyWeakTopic      int
		notifyWeakTopicPct   float64
		notifyMilestones     int
		notifyReadiness      int
		notifyStreak         int
		quietHoursStart      string
		quietHoursEnd        string
	)

	err := d.db.QueryRowContext(ctx, `
		SELECT delivery_interval_min, max_unanswered, active_pack_ids, focus_mode,
		       notify_inactivity, notify_inactivity_days,
		       notify_weak_topic, notify_weak_topic_pct,
		       notify_milestones, notify_readiness, notify_streak,
		       quiet_hours_start, quiet_hours_end
		FROM user_preferences
		WHERE user_id = ?`, userID,
	).Scan(
		&deliveryIntervalMin, &maxUnanswered, &activePackIDsJSON, &focusMode,
		&notifyInactivity, &notifyInactivityDays,
		&notifyWeakTopic, &notifyWeakTopicPct,
		&notifyMilestones, &notifyReadiness, &notifyStreak,
		&quietHoursStart, &quietHoursEnd,
	)

	if err == sql.ErrNoRows {
		// Return defaults for unknown users — not an error.
		defaults := model.DefaultPreferences(userID)
		return &defaults, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting preferences: %w", err)
	}

	var activePackIDs []string
	if err = unmarshalJSON(activePackIDsJSON, &activePackIDs); err != nil {
		return nil, fmt.Errorf("unmarshalling active_pack_ids: %w", err)
	}
	if activePackIDs == nil {
		activePackIDs = []string{}
	}

	return &model.UserPreferences{
		UserID:               userID,
		DeliveryIntervalMin:  deliveryIntervalMin,
		MaxUnanswered:        maxUnanswered,
		ActivePackIDs:        activePackIDs,
		FocusMode:            model.FocusMode(focusMode),
		NotifyInactivity:     intToBool(notifyInactivity),
		NotifyInactivityDays: notifyInactivityDays,
		NotifyWeakTopic:      intToBool(notifyWeakTopic),
		NotifyWeakTopicPct:   notifyWeakTopicPct,
		NotifyMilestones:     intToBool(notifyMilestones),
		NotifyReadiness:      intToBool(notifyReadiness),
		NotifyStreak:         intToBool(notifyStreak),
		QuietHoursStart:      quietHoursStart,
		QuietHoursEnd:        quietHoursEnd,
	}, nil
}

func (d *DB) UpdatePreferences(ctx context.Context, prefs model.UserPreferences) error {
	activePackIDsJSON, err := marshalJSON(prefs.ActivePackIDs)
	if err != nil {
		return fmt.Errorf("marshalling active_pack_ids: %w", err)
	}

	_, err = d.db.ExecContext(ctx, `
		INSERT INTO user_preferences
			(user_id, delivery_interval_min, max_unanswered, active_pack_ids,
			 focus_mode, notify_inactivity, notify_inactivity_days,
			 notify_weak_topic, notify_weak_topic_pct,
			 notify_milestones, notify_readiness, notify_streak,
			 quiet_hours_start, quiet_hours_end)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			delivery_interval_min  = excluded.delivery_interval_min,
			max_unanswered         = excluded.max_unanswered,
			active_pack_ids        = excluded.active_pack_ids,
			focus_mode             = excluded.focus_mode,
			notify_inactivity      = excluded.notify_inactivity,
			notify_inactivity_days = excluded.notify_inactivity_days,
			notify_weak_topic      = excluded.notify_weak_topic,
			notify_weak_topic_pct  = excluded.notify_weak_topic_pct,
			notify_milestones      = excluded.notify_milestones,
			notify_readiness       = excluded.notify_readiness,
			notify_streak          = excluded.notify_streak,
			quiet_hours_start      = excluded.quiet_hours_start,
			quiet_hours_end        = excluded.quiet_hours_end`,
		prefs.UserID, prefs.DeliveryIntervalMin, prefs.MaxUnanswered, activePackIDsJSON,
		string(prefs.FocusMode),
		boolToInt(prefs.NotifyInactivity), prefs.NotifyInactivityDays,
		boolToInt(prefs.NotifyWeakTopic), prefs.NotifyWeakTopicPct,
		boolToInt(prefs.NotifyMilestones), boolToInt(prefs.NotifyReadiness),
		boolToInt(prefs.NotifyStreak),
		prefs.QuietHoursStart, prefs.QuietHoursEnd,
	)
	if err != nil {
		return fmt.Errorf("updating preferences: %w", err)
	}
	return nil
}
