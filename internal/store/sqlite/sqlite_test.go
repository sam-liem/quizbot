package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/sam-liem/quizbot/internal/model"
	"github.com/sam-liem/quizbot/internal/store/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestDB opens an in-memory SQLite database and registers cleanup.
func newTestDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// samplePack returns a QuizPack with 2 topics and 2 questions.
func samplePack() model.QuizPack {
	return model.QuizPack{
		ID:          "pack-1",
		Name:        "Life in the UK",
		Description: "Test preparation pack",
		Version:     "1.0.0",
		TestFormat: model.TestFormat{
			QuestionCount: 24,
			PassMarkPct:   75.0,
			TimeLimitSec:  45 * 60,
		},
		Topics: []model.Topic{
			{ID: "topic-1", Name: "History", Description: "British history"},
			{ID: "topic-2", Name: "Culture", Description: "British culture"},
		},
		Questions: []model.Question{
			{
				ID:           "q-1",
				TopicID:      "topic-1",
				Text:         "When was the Battle of Hastings?",
				Choices:      []string{"1066", "1265", "1485", "1603"},
				CorrectIndex: 0,
				Explanation:  "The Battle of Hastings was in 1066.",
			},
			{
				ID:           "q-2",
				TopicID:      "topic-2",
				Text:         "What is the national flower of England?",
				Choices:      []string{"Thistle", "Daffodil", "Tudor Rose", "Shamrock"},
				CorrectIndex: 2,
				Explanation:  "The Tudor Rose is the national flower of England.",
			},
		},
	}
}

// ---- Quiz Pack tests ----

func TestSaveAndGetQuizPack(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	pack := samplePack()

	err := db.SaveQuizPack(ctx, pack)
	require.NoError(t, err)

	got, err := db.GetQuizPack(ctx, pack.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, pack.ID, got.ID)
	assert.Equal(t, pack.Name, got.Name)
	assert.Equal(t, pack.Description, got.Description)
	assert.Equal(t, pack.Version, got.Version)
	assert.Equal(t, pack.TestFormat, got.TestFormat)
	assert.Equal(t, pack.Topics, got.Topics)
	assert.Equal(t, pack.Questions, got.Questions)
}

func TestGetQuizPack_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	got, err := db.GetQuizPack(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestListQuizPacks(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	pack1 := samplePack()
	pack2 := samplePack()
	pack2.ID = "pack-2"
	pack2.Name = "Another Pack"

	require.NoError(t, db.SaveQuizPack(ctx, pack1))
	require.NoError(t, db.SaveQuizPack(ctx, pack2))

	packs, err := db.ListQuizPacks(ctx)
	require.NoError(t, err)
	assert.Len(t, packs, 2)

	ids := []string{packs[0].ID, packs[1].ID}
	assert.Contains(t, ids, "pack-1")
	assert.Contains(t, ids, "pack-2")
}

func TestSaveQuizPack_Upsert(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	pack := samplePack()
	require.NoError(t, db.SaveQuizPack(ctx, pack))

	pack.Name = "Updated Name"
	pack.Version = "2.0.0"
	require.NoError(t, db.SaveQuizPack(ctx, pack))

	got, err := db.GetQuizPack(ctx, pack.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "Updated Name", got.Name)
	assert.Equal(t, "2.0.0", got.Version)
}

// ---- Question State tests ----

func TestQuestionState_CRUD(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	state := model.QuestionState{
		UserID:          "user-1",
		PackID:          "pack-1",
		QuestionID:      "q-1",
		EaseFactor:      2.5,
		IntervalDays:    1.0,
		RepetitionCount: 1,
		NextReviewAt:    now.Add(24 * time.Hour),
		LastResult:      model.AnswerResultCorrect,
		LastReviewedAt:  now,
	}

	err := db.UpdateQuestionState(ctx, state)
	require.NoError(t, err)

	got, err := db.GetQuestionState(ctx, state.UserID, state.PackID, state.QuestionID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, state.UserID, got.UserID)
	assert.Equal(t, state.PackID, got.PackID)
	assert.Equal(t, state.QuestionID, got.QuestionID)
	assert.InDelta(t, state.EaseFactor, got.EaseFactor, 0.001)
	assert.InDelta(t, state.IntervalDays, got.IntervalDays, 0.001)
	assert.Equal(t, state.RepetitionCount, got.RepetitionCount)
	assert.Equal(t, state.LastResult, got.LastResult)
	assert.True(t, state.NextReviewAt.Equal(got.NextReviewAt), "NextReviewAt mismatch: want %v, got %v", state.NextReviewAt, got.NextReviewAt)
	assert.True(t, state.LastReviewedAt.Equal(got.LastReviewedAt), "LastReviewedAt mismatch: want %v, got %v", state.LastReviewedAt, got.LastReviewedAt)
}

func TestQuestionState_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	got, err := db.GetQuestionState(ctx, "user-1", "pack-1", "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

// ---- Topic Stats tests ----

func TestTopicStats_CRUD(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	stats := model.TopicStats{
		UserID:          "user-1",
		PackID:          "pack-1",
		TopicID:         "topic-1",
		TotalAttempts:   10,
		CorrectCount:    7,
		RollingAccuracy: 0.7,
		CurrentStreak:   3,
		BestStreak:      5,
	}

	err := db.UpdateTopicStats(ctx, stats)
	require.NoError(t, err)

	got, err := db.GetTopicStats(ctx, stats.UserID, stats.PackID, stats.TopicID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, stats, *got)
}

func TestTopicStats_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	got, err := db.GetTopicStats(ctx, "user-1", "pack-1", "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestListTopicStats(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	s1 := model.TopicStats{UserID: "user-1", PackID: "pack-1", TopicID: "topic-1", TotalAttempts: 5}
	s2 := model.TopicStats{UserID: "user-1", PackID: "pack-1", TopicID: "topic-2", TotalAttempts: 3}

	require.NoError(t, db.UpdateTopicStats(ctx, s1))
	require.NoError(t, db.UpdateTopicStats(ctx, s2))

	list, err := db.ListTopicStats(ctx, "user-1", "pack-1")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestListTopicStats_IsolatedByUser(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	s1 := model.TopicStats{UserID: "user-1", PackID: "pack-1", TopicID: "topic-1", TotalAttempts: 5}
	s2 := model.TopicStats{UserID: "user-2", PackID: "pack-1", TopicID: "topic-1", TotalAttempts: 8}

	require.NoError(t, db.UpdateTopicStats(ctx, s1))
	require.NoError(t, db.UpdateTopicStats(ctx, s2))

	list1, err := db.ListTopicStats(ctx, "user-1", "pack-1")
	require.NoError(t, err)
	assert.Len(t, list1, 1)
	assert.Equal(t, "user-1", list1[0].UserID)

	list2, err := db.ListTopicStats(ctx, "user-2", "pack-1")
	require.NoError(t, err)
	assert.Len(t, list2, 1)
	assert.Equal(t, "user-2", list2[0].UserID)
}

// ---- Study Session tests ----

func TestStudySession_CreateGetUpdate(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	session := model.StudySession{
		ID:            "session-1",
		UserID:        "user-1",
		PackID:        "pack-1",
		Mode:          model.SessionModePractice,
		StartedAt:     now,
		QuestionCount: 5,
		CorrectCount:  3,
		Attempts: []model.QuestionAttempt{
			{
				QuestionID:  "q-1",
				AnswerIndex: 0,
				Correct:     true,
				TimeTakenMs: 1200,
				AnsweredAt:  now,
			},
		},
	}

	err := db.CreateSession(ctx, session)
	require.NoError(t, err)

	got, err := db.GetSession(ctx, session.UserID, session.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, session.ID, got.ID)
	assert.Equal(t, session.UserID, got.UserID)
	assert.Equal(t, session.Mode, got.Mode)
	assert.Equal(t, session.QuestionCount, got.QuestionCount)
	assert.Equal(t, session.CorrectCount, got.CorrectCount)
	assert.Nil(t, got.EndedAt)
	require.Len(t, got.Attempts, 1)
	assert.Equal(t, "q-1", got.Attempts[0].QuestionID)

	// Update: mark as ended
	ended := now.Add(10 * time.Minute)
	session.EndedAt = &ended
	session.CorrectCount = 4

	err = db.UpdateSession(ctx, session)
	require.NoError(t, err)

	got2, err := db.GetSession(ctx, session.UserID, session.ID)
	require.NoError(t, err)
	require.NotNil(t, got2)
	assert.Equal(t, 4, got2.CorrectCount)
	require.NotNil(t, got2.EndedAt)
	assert.True(t, ended.Equal(*got2.EndedAt))
}

func TestStudySession_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	got, err := db.GetSession(ctx, "user-1", "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

// ---- Quiz Session tests ----

func TestQuizSession_SaveGetUpsert(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	qs := model.QuizSession{
		ID:           "quiz-session-1",
		UserID:       "user-1",
		PackID:       "pack-1",
		Mode:         model.SessionModeMock,
		QuestionIDs:  []string{"q-1", "q-2"},
		CurrentIndex: 0,
		Answers:      map[string]int{},
		StartedAt:    now,
		TimeLimitSec: 2700,
		Status:       model.QuizSessionStatusInProgress,
	}

	err := db.SaveQuizSession(ctx, qs)
	require.NoError(t, err)

	got, err := db.GetQuizSession(ctx, qs.UserID, qs.ID)
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, qs.ID, got.ID)
	assert.Equal(t, qs.UserID, got.UserID)
	assert.Equal(t, qs.Mode, got.Mode)
	assert.Equal(t, qs.QuestionIDs, got.QuestionIDs)
	assert.Equal(t, qs.Answers, got.Answers)
	assert.Equal(t, qs.Status, got.Status)

	// Upsert: advance question and record answer
	qs.CurrentIndex = 1
	qs.Answers = map[string]int{"q-1": 0}

	err = db.SaveQuizSession(ctx, qs)
	require.NoError(t, err)

	got2, err := db.GetQuizSession(ctx, qs.UserID, qs.ID)
	require.NoError(t, err)
	require.NotNil(t, got2)
	assert.Equal(t, 1, got2.CurrentIndex)
	assert.Equal(t, map[string]int{"q-1": 0}, got2.Answers)
}

func TestQuizSession_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	got, err := db.GetQuizSession(ctx, "user-1", "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, got)
}

// ---- Preferences tests ----

func TestPreferences_DefaultOnFirstGet(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	got, err := db.GetPreferences(ctx, "brand-new-user")
	require.NoError(t, err)
	require.NotNil(t, got)

	defaults := model.DefaultPreferences("brand-new-user")
	assert.Equal(t, defaults, *got)
}

func TestPreferences_Update(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	prefs := model.DefaultPreferences("user-1")
	prefs.DeliveryIntervalMin = 30
	prefs.MaxUnanswered = 5
	prefs.ActivePackIDs = []string{"pack-1", "pack-2"}
	prefs.NotifyInactivity = false
	prefs.QuietHoursStart = "23:00"

	err := db.UpdatePreferences(ctx, prefs)
	require.NoError(t, err)

	got, err := db.GetPreferences(ctx, "user-1")
	require.NoError(t, err)
	require.NotNil(t, got)

	assert.Equal(t, 30, got.DeliveryIntervalMin)
	assert.Equal(t, 5, got.MaxUnanswered)
	assert.Equal(t, []string{"pack-1", "pack-2"}, got.ActivePackIDs)
	assert.False(t, got.NotifyInactivity)
	assert.Equal(t, "23:00", got.QuietHoursStart)
}

// ---- Isolation test ----

func TestUserIsolation_QuestionState(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	s1 := model.QuestionState{
		UserID: "user-1", PackID: "pack-1", QuestionID: "q-1",
		EaseFactor: 2.5, IntervalDays: 1.0, RepetitionCount: 1,
		NextReviewAt: now.Add(24 * time.Hour), LastResult: model.AnswerResultCorrect,
		LastReviewedAt: now,
	}
	s2 := model.QuestionState{
		UserID: "user-2", PackID: "pack-1", QuestionID: "q-1",
		EaseFactor: 1.3, IntervalDays: 0.5, RepetitionCount: 2,
		NextReviewAt: now.Add(12 * time.Hour), LastResult: model.AnswerResultWrong,
		LastReviewedAt: now,
	}

	require.NoError(t, db.UpdateQuestionState(ctx, s1))
	require.NoError(t, db.UpdateQuestionState(ctx, s2))

	got1, err := db.GetQuestionState(ctx, "user-1", "pack-1", "q-1")
	require.NoError(t, err)
	require.NotNil(t, got1)
	assert.InDelta(t, 2.5, got1.EaseFactor, 0.001)
	assert.Equal(t, model.AnswerResultCorrect, got1.LastResult)

	got2, err := db.GetQuestionState(ctx, "user-2", "pack-1", "q-1")
	require.NoError(t, err)
	require.NotNil(t, got2)
	assert.InDelta(t, 1.3, got2.EaseFactor, 0.001)
	assert.Equal(t, model.AnswerResultWrong, got2.LastResult)
}

// ---- ListQuizSessions tests ----

func TestListQuizSessions(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	// Session 1: in_progress, started 2 hours ago
	qs1 := model.QuizSession{
		ID:           "qs-1",
		UserID:       "user-1",
		PackID:       "pack-1",
		Mode:         model.SessionModeMock,
		QuestionIDs:  []string{"q-1", "q-2"},
		CurrentIndex: 0,
		Answers:      map[string]int{},
		StartedAt:    now.Add(-2 * time.Hour),
		TimeLimitSec: 2700,
		Status:       model.QuizSessionStatusInProgress,
	}

	// Session 2: completed, started 1 hour ago
	qs2 := model.QuizSession{
		ID:           "qs-2",
		UserID:       "user-1",
		PackID:       "pack-1",
		Mode:         model.SessionModeMock,
		QuestionIDs:  []string{"q-1", "q-2"},
		CurrentIndex: 2,
		Answers:      map[string]int{"q-1": 0, "q-2": 1},
		StartedAt:    now.Add(-1 * time.Hour),
		TimeLimitSec: 2700,
		Status:       model.QuizSessionStatusCompleted,
	}

	// Session 3: in_progress, most recent (started 30 minutes ago)
	qs3 := model.QuizSession{
		ID:           "qs-3",
		UserID:       "user-1",
		PackID:       "pack-1",
		Mode:         model.SessionModePractice,
		QuestionIDs:  []string{"q-1"},
		CurrentIndex: 0,
		Answers:      map[string]int{},
		StartedAt:    now.Add(-30 * time.Minute),
		TimeLimitSec: 0,
		Status:       model.QuizSessionStatusInProgress,
	}

	require.NoError(t, db.SaveQuizSession(ctx, qs1))
	require.NoError(t, db.SaveQuizSession(ctx, qs2))
	require.NoError(t, db.SaveQuizSession(ctx, qs3))

	// List in_progress sessions — should return qs3 first (most recent), then qs1.
	inProgress, err := db.ListQuizSessions(ctx, "user-1", model.QuizSessionStatusInProgress)
	require.NoError(t, err)
	assert.Len(t, inProgress, 2, "expected 2 in_progress sessions")
	assert.Equal(t, "qs-3", inProgress[0].ID, "most recent in_progress session should be first")
	assert.Equal(t, "qs-1", inProgress[1].ID)
	for _, s := range inProgress {
		assert.Equal(t, model.QuizSessionStatusInProgress, s.Status)
	}

	// List completed sessions — should return only qs2.
	completed, err := db.ListQuizSessions(ctx, "user-1", model.QuizSessionStatusCompleted)
	require.NoError(t, err)
	assert.Len(t, completed, 1, "expected 1 completed session")
	assert.Equal(t, "qs-2", completed[0].ID)
	assert.Equal(t, model.QuizSessionStatusCompleted, completed[0].Status)
}

func TestListQuizSessions_EmptyResult(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	sessions, err := db.ListQuizSessions(ctx, "user-1", model.QuizSessionStatusInProgress)
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

func TestListQuizSessions_UserIsolation(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	qs1 := model.QuizSession{
		ID: "qs-u1", UserID: "user-1", PackID: "pack-1",
		Mode: model.SessionModeMock, QuestionIDs: []string{"q-1"},
		CurrentIndex: 0, Answers: map[string]int{},
		StartedAt: now, TimeLimitSec: 2700,
		Status: model.QuizSessionStatusInProgress,
	}
	qs2 := model.QuizSession{
		ID: "qs-u2", UserID: "user-2", PackID: "pack-1",
		Mode: model.SessionModeMock, QuestionIDs: []string{"q-1"},
		CurrentIndex: 0, Answers: map[string]int{},
		StartedAt: now, TimeLimitSec: 2700,
		Status: model.QuizSessionStatusInProgress,
	}

	require.NoError(t, db.SaveQuizSession(ctx, qs1))
	require.NoError(t, db.SaveQuizSession(ctx, qs2))

	list1, err := db.ListQuizSessions(ctx, "user-1", model.QuizSessionStatusInProgress)
	require.NoError(t, err)
	require.Len(t, list1, 1)
	assert.Equal(t, "user-1", list1[0].UserID)

	list2, err := db.ListQuizSessions(ctx, "user-2", model.QuizSessionStatusInProgress)
	require.NoError(t, err)
	require.Len(t, list2, 1)
	assert.Equal(t, "user-2", list2[0].UserID)
}
