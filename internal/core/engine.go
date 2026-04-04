package core

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/sam-liem/quizbot/internal/model"
	"github.com/sam-liem/quizbot/internal/store"
)

// QuizOptions holds optional parameters for starting a quiz session.
type QuizOptions struct {
	QuestionCount int
	TopicID       string
}

// SubmitAnswerResult holds the outcome of a submitted answer.
type SubmitAnswerResult struct {
	Correct      bool
	CorrectIndex int
	Explanation  string
}

// SessionStatus holds a summary of the current quiz session state.
type SessionStatus struct {
	TotalQuestions int
	Answered       int
	Correct        int
	Status         model.QuizSessionStatus
	TimeLimitSec   int
	StartedAt      time.Time
}

// QuizEngine orchestrates quiz sessions.
type QuizEngine struct {
	repo store.Repository
}

// NewQuizEngine creates a new QuizEngine with the given repository.
func NewQuizEngine(repo store.Repository) *QuizEngine {
	return &QuizEngine{repo: repo}
}

// Repo returns the underlying repository (useful for test assertions).
func (e *QuizEngine) Repo() store.Repository {
	return e.repo
}

// generateID returns a unique string ID based on the current nanosecond clock
// plus a small random suffix to avoid collisions within the same nanosecond.
func generateID() string {
	return fmt.Sprintf("%d%d", time.Now().UnixNano(), rand.Intn(1000))
}

// StartQuiz creates and persists a new QuizSession for the given user/pack/mode.
//
// Mock mode: selects pack.TestFormat.QuestionCount random questions, sets time
// limit from pack.TestFormat.TimeLimitSec.
//
// Practice mode: orders questions by spaced-repetition priority (overdue first,
// then unseen, then not-yet-due). Supports optional topic filter. Untimed.
func (e *QuizEngine) StartQuiz(
	ctx context.Context,
	userID, packID string,
	mode model.SessionMode,
	opts QuizOptions,
) (*model.QuizSession, error) {
	pack, err := e.repo.GetQuizPack(ctx, packID)
	if err != nil {
		return nil, fmt.Errorf("StartQuiz: getting pack: %w", err)
	}

	// Filter questions by topic if requested (practice mode).
	questions := pack.Questions
	if opts.TopicID != "" {
		var filtered []model.Question
		for _, q := range questions {
			if q.TopicID == opts.TopicID {
				filtered = append(filtered, q)
			}
		}
		questions = filtered
	}

	if len(questions) == 0 {
		return nil, fmt.Errorf("StartQuiz: no questions available for pack %q (topic filter: %q)", packID, opts.TopicID)
	}

	var questionIDs []string
	var timeLimitSec int

	switch mode {
	case model.SessionModeMock:
		count := pack.TestFormat.QuestionCount
		if opts.QuestionCount > 0 {
			count = opts.QuestionCount
		}
		if count > len(questions) {
			count = len(questions)
		}
		questionIDs = randomSubset(questions, count)
		timeLimitSec = pack.TestFormat.TimeLimitSec

	case model.SessionModePractice:
		questionIDs, err = e.buildPracticeOrder(ctx, userID, packID, questions)
		if err != nil {
			return nil, fmt.Errorf("StartQuiz: building practice order: %w", err)
		}
		// Practice is untimed.
		timeLimitSec = 0

	default:
		return nil, fmt.Errorf("StartQuiz: unsupported mode %q", mode)
	}

	session := model.QuizSession{
		ID:           generateID(),
		UserID:       userID,
		PackID:       packID,
		Mode:         mode,
		QuestionIDs:  questionIDs,
		CurrentIndex: 0,
		Answers:      make(map[string]int),
		StartedAt:    time.Now(),
		TimeLimitSec: timeLimitSec,
		Status:       model.QuizSessionStatusInProgress,
	}

	if err := e.repo.SaveQuizSession(ctx, session); err != nil {
		return nil, fmt.Errorf("StartQuiz: saving session: %w", err)
	}

	return &session, nil
}

// buildPracticeOrder returns question IDs sorted by spaced-repetition priority:
//  1. Overdue questions (NextReviewAt <= now), most overdue first.
//  2. Unseen questions (no state).
//  3. Not-yet-due questions, earliest due date first.
func (e *QuizEngine) buildPracticeOrder(
	ctx context.Context,
	userID, packID string,
	questions []model.Question,
) ([]string, error) {
	now := time.Now()

	type entry struct {
		id       string
		category int // 0=overdue, 1=unseen, 2=not-due
		dueAt    time.Time
	}

	entries := make([]entry, 0, len(questions))
	for _, q := range questions {
		state, err := e.repo.GetQuestionState(ctx, userID, packID, q.ID)
		if err != nil {
			return nil, fmt.Errorf("getting question state for %q: %w", q.ID, err)
		}

		var en entry
		en.id = q.ID

		if state == nil {
			// Unseen.
			en.category = 1
			en.dueAt = time.Time{}
		} else if !state.NextReviewAt.After(now) {
			// Overdue or exactly due.
			en.category = 0
			en.dueAt = state.NextReviewAt
		} else {
			// Not yet due.
			en.category = 2
			en.dueAt = state.NextReviewAt
		}

		entries = append(entries, en)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].category != entries[j].category {
			return entries[i].category < entries[j].category
		}
		// Within category 0 (overdue): earliest NextReviewAt = most overdue = first.
		// Within category 2 (not-due): earliest NextReviewAt = soonest due = first.
		// Category 1 (unseen): order is stable/arbitrary.
		if entries[i].category == 0 || entries[i].category == 2 {
			return entries[i].dueAt.Before(entries[j].dueAt)
		}
		return false
	})

	ids := make([]string, len(entries))
	for i, en := range entries {
		ids[i] = en.id
	}
	return ids, nil
}

// randomSubset returns count random question IDs from questions without replacement.
func randomSubset(questions []model.Question, count int) []string {
	indices := rand.Perm(len(questions))
	ids := make([]string, count)
	for i := 0; i < count; i++ {
		ids[i] = questions[indices[i]].ID
	}
	return ids
}

// NextQuestion returns the current unanswered question in the session, or nil if complete.
func (e *QuizEngine) NextQuestion(ctx context.Context, userID, sessionID string) (*model.Question, error) {
	session, err := e.repo.GetQuizSession(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("NextQuestion: getting session: %w", err)
	}

	if session.Status != model.QuizSessionStatusInProgress {
		return nil, nil
	}

	if session.CurrentIndex >= len(session.QuestionIDs) {
		return nil, nil
	}

	// Find the next unanswered question starting at CurrentIndex.
	for i := session.CurrentIndex; i < len(session.QuestionIDs); i++ {
		qID := session.QuestionIDs[i]
		if _, answered := session.Answers[qID]; !answered {
			pack, err := e.repo.GetQuizPack(ctx, session.PackID)
			if err != nil {
				return nil, fmt.Errorf("NextQuestion: getting pack: %w", err)
			}
			for _, q := range pack.Questions {
				if q.ID == qID {
					return &q, nil
				}
			}
			return nil, fmt.Errorf("NextQuestion: question %q not found in pack", qID)
		}
	}

	return nil, nil
}

// SubmitAnswer records an answer for the given question, updates spaced-repetition
// state and topic stats, advances the session index, and marks the session complete
// when all questions are answered.
func (e *QuizEngine) SubmitAnswer(
	ctx context.Context,
	userID, sessionID, questionID string,
	answerIndex int,
) (*SubmitAnswerResult, error) {
	session, err := e.repo.GetQuizSession(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("SubmitAnswer: getting session: %w", err)
	}

	// Validate that questionID belongs to this session.
	inSession := false
	for _, qID := range session.QuestionIDs {
		if qID == questionID {
			inSession = true
			break
		}
	}
	if !inSession {
		return nil, fmt.Errorf("SubmitAnswer: question %q is not part of session %q", questionID, sessionID)
	}

	// Look up the question in the pack.
	pack, err := e.repo.GetQuizPack(ctx, session.PackID)
	if err != nil {
		return nil, fmt.Errorf("SubmitAnswer: getting pack: %w", err)
	}

	var question *model.Question
	for i := range pack.Questions {
		if pack.Questions[i].ID == questionID {
			question = &pack.Questions[i]
			break
		}
	}
	if question == nil {
		return nil, fmt.Errorf("SubmitAnswer: question %q not found in pack", questionID)
	}

	correct := answerIndex == question.CorrectIndex

	// Record answer in session.
	session.Answers[questionID] = answerIndex

	// Advance CurrentIndex past all answered questions.
	for session.CurrentIndex < len(session.QuestionIDs) {
		if _, answered := session.Answers[session.QuestionIDs[session.CurrentIndex]]; answered {
			session.CurrentIndex++
		} else {
			break
		}
	}

	// Mark completed if all questions answered.
	if len(session.Answers) >= len(session.QuestionIDs) {
		session.Status = model.QuizSessionStatusCompleted
	}

	if err := e.repo.SaveQuizSession(ctx, *session); err != nil {
		return nil, fmt.Errorf("SubmitAnswer: saving session: %w", err)
	}

	// Update spaced repetition state.
	if err := e.updateQuestionState(ctx, userID, session.PackID, questionID, correct); err != nil {
		return nil, fmt.Errorf("SubmitAnswer: updating question state: %w", err)
	}

	// Update topic stats.
	if err := e.updateTopicStats(ctx, userID, session.PackID, question.TopicID, correct); err != nil {
		return nil, fmt.Errorf("SubmitAnswer: updating topic stats: %w", err)
	}

	return &SubmitAnswerResult{
		Correct:      correct,
		CorrectIndex: question.CorrectIndex,
		Explanation:  question.Explanation,
	}, nil
}

// updateQuestionState loads existing state (or creates a fresh one), applies
// CalculateNextReview, and persists the updated state.
func (e *QuizEngine) updateQuestionState(
	ctx context.Context,
	userID, packID, questionID string,
	correct bool,
) error {
	state, err := e.repo.GetQuestionState(ctx, userID, packID, questionID)
	if err != nil {
		return fmt.Errorf("getting state: %w", err)
	}

	var current model.QuestionState
	if state != nil {
		current = *state
	} else {
		current = model.QuestionState{
			UserID:       userID,
			PackID:       packID,
			QuestionID:   questionID,
			EaseFactor:   2.5,
			IntervalDays: 0,
		}
	}

	updated := CalculateNextReview(current, correct, time.Now())
	return e.repo.UpdateQuestionState(ctx, updated)
}

// updateTopicStats loads existing topic stats (or creates fresh ones), updates
// attempt/streak counters, recalculates rolling accuracy, and persists.
func (e *QuizEngine) updateTopicStats(
	ctx context.Context,
	userID, packID, topicID string,
	correct bool,
) error {
	ts, err := e.repo.GetTopicStats(ctx, userID, packID, topicID)
	if err != nil {
		return fmt.Errorf("getting topic stats: %w", err)
	}

	var stats model.TopicStats
	if ts != nil {
		stats = *ts
	} else {
		stats = model.TopicStats{
			UserID:  userID,
			PackID:  packID,
			TopicID: topicID,
		}
	}

	stats.TotalAttempts++
	if correct {
		stats.CorrectCount++
		stats.CurrentStreak++
		if stats.CurrentStreak > stats.BestStreak {
			stats.BestStreak = stats.CurrentStreak
		}
	} else {
		stats.CurrentStreak = 0
	}

	if stats.TotalAttempts > 0 {
		stats.RollingAccuracy = float64(stats.CorrectCount) / float64(stats.TotalAttempts)
	}

	return e.repo.UpdateTopicStats(ctx, stats)
}

// GetSessionStatus returns a summary of the current session state.
func (e *QuizEngine) GetSessionStatus(ctx context.Context, userID, sessionID string) (*SessionStatus, error) {
	session, err := e.repo.GetQuizSession(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("GetSessionStatus: getting session: %w", err)
	}

	pack, err := e.repo.GetQuizPack(ctx, session.PackID)
	if err != nil {
		return nil, fmt.Errorf("GetSessionStatus: getting pack: %w", err)
	}

	// Build a lookup of correct indices.
	correctIndex := make(map[string]int, len(pack.Questions))
	for _, q := range pack.Questions {
		correctIndex[q.ID] = q.CorrectIndex
	}

	answered := len(session.Answers)
	correct := 0
	for qID, idx := range session.Answers {
		if ci, ok := correctIndex[qID]; ok && idx == ci {
			correct++
		}
	}

	return &SessionStatus{
		TotalQuestions: len(session.QuestionIDs),
		Answered:       answered,
		Correct:        correct,
		Status:         session.Status,
		TimeLimitSec:   session.TimeLimitSec,
		StartedAt:      session.StartedAt,
	}, nil
}

// ResumeSession returns the session if it is still in_progress.
// Returns an error if the session is not found or not in_progress.
func (e *QuizEngine) ResumeSession(ctx context.Context, userID, sessionID string) (*model.QuizSession, error) {
	session, err := e.repo.GetQuizSession(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("ResumeSession: %w", err)
	}

	if session.Status != model.QuizSessionStatusInProgress {
		return nil, fmt.Errorf("ResumeSession: session %q is not in_progress (status: %s)", sessionID, session.Status)
	}

	return session, nil
}

// FindResumableSession finds the most recent in-progress quiz session for the
// given user, or returns nil if no such session exists.
func (e *QuizEngine) FindResumableSession(ctx context.Context, userID string) (*model.QuizSession, error) {
	sessions, err := e.repo.ListQuizSessions(ctx, userID, model.QuizSessionStatusInProgress)
	if err != nil {
		return nil, fmt.Errorf("FindResumableSession: listing sessions: %w", err)
	}
	if len(sessions) == 0 {
		return nil, nil
	}
	// Sessions are ordered by started_at DESC, so the first is the most recent.
	return &sessions[0], nil
}

// AbandonSession marks a session as abandoned.
func (e *QuizEngine) AbandonSession(ctx context.Context, userID, sessionID string) error {
	session, err := e.repo.GetQuizSession(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("AbandonSession: %w", err)
	}

	session.Status = model.QuizSessionStatusAbandoned
	if err := e.repo.SaveQuizSession(ctx, *session); err != nil {
		return fmt.Errorf("AbandonSession: saving session: %w", err)
	}

	return nil
}
