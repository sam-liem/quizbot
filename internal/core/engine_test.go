package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/sam-liem/quizbot/internal/core"
	"github.com/sam-liem/quizbot/internal/model"
	"github.com/sam-liem/quizbot/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUserID = "user1"
	testPackID = "pack1"
)

// setupEngine creates a MockRepository, a QuizEngine, and saves a test pack
// with 5 questions across 2 topics.
func setupEngine(t *testing.T) (*core.QuizEngine, *store.MockRepository) {
	t.Helper()
	repo := store.NewMockRepository()
	engine := core.NewQuizEngine(repo)

	pack := model.QuizPack{
		ID:          testPackID,
		Name:        "Life in the UK",
		Description: "Test pack",
		Version:     "1.0",
		TestFormat: model.TestFormat{
			QuestionCount: 3,
			PassMarkPct:   75.0,
			TimeLimitSec:  45 * 60,
		},
		Topics: []model.Topic{
			{ID: "topic1", Name: "History"},
			{ID: "topic2", Name: "Government"},
		},
		Questions: []model.Question{
			{ID: "q001", TopicID: "topic1", Text: "Q1", Choices: []string{"A", "B", "C", "D"}, CorrectIndex: 0, Explanation: "exp1"},
			{ID: "q002", TopicID: "topic1", Text: "Q2", Choices: []string{"A", "B", "C", "D"}, CorrectIndex: 1, Explanation: "exp2"},
			{ID: "q003", TopicID: "topic1", Text: "Q3", Choices: []string{"A", "B", "C", "D"}, CorrectIndex: 2, Explanation: "exp3"},
			{ID: "q004", TopicID: "topic2", Text: "Q4", Choices: []string{"A", "B", "C", "D"}, CorrectIndex: 3, Explanation: "exp4"},
			{ID: "q005", TopicID: "topic2", Text: "Q5", Choices: []string{"A", "B", "C", "D"}, CorrectIndex: 0, Explanation: "exp5"},
		},
	}

	err := repo.SaveQuizPack(context.Background(), pack)
	require.NoError(t, err)

	return engine, repo
}

// --- StartQuiz ---

func TestStartQuiz_MockMode(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)
	require.NotNil(t, session)

	assert.Equal(t, testUserID, session.UserID)
	assert.Equal(t, testPackID, session.PackID)
	assert.Equal(t, model.SessionModeMock, session.Mode)
	assert.Equal(t, model.QuizSessionStatusInProgress, session.Status)
	// Mock mode uses pack test_format question_count (3)
	assert.Equal(t, 3, len(session.QuestionIDs))
	assert.Equal(t, 45*60, session.TimeLimitSec)
	assert.Equal(t, 0, session.CurrentIndex)
	assert.NotEmpty(t, session.ID)
}

func TestStartQuiz_PracticeMode(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModePractice, core.QuizOptions{})
	require.NoError(t, err)
	require.NotNil(t, session)

	assert.Equal(t, model.SessionModePractice, session.Mode)
	// Practice mode: untimed
	assert.Equal(t, 0, session.TimeLimitSec)
	// All 5 questions (no topic filter)
	assert.Equal(t, 5, len(session.QuestionIDs))
}

func TestStartQuiz_PracticeMode_TopicFilter(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModePractice, core.QuizOptions{TopicID: "topic1"})
	require.NoError(t, err)
	require.NotNil(t, session)

	// topic1 has q001, q002, q003
	assert.Equal(t, 3, len(session.QuestionIDs))
	for _, qID := range session.QuestionIDs {
		assert.Contains(t, []string{"q001", "q002", "q003"}, qID)
	}
}

func TestStartQuiz_PackNotFound(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, "nonexistent", model.SessionModeMock, core.QuizOptions{})
	assert.Error(t, err)
	assert.Nil(t, session)
}

// --- NextQuestion ---

func TestNextQuestion(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	q, err := engine.NextQuestion(ctx, testUserID, session.ID)
	require.NoError(t, err)
	require.NotNil(t, q)
	assert.NotEmpty(t, q.ID)
}

func TestNextQuestion_SessionCompleted(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	// Answer all questions to complete the session.
	pack, err := repo.GetQuizPack(ctx, testPackID)
	require.NoError(t, err)
	questionMap := make(map[string]model.Question)
	for _, q := range pack.Questions {
		questionMap[q.ID] = q
	}

	for _, qID := range session.QuestionIDs {
		q := questionMap[qID]
		_, err := engine.SubmitAnswer(ctx, testUserID, session.ID, qID, q.CorrectIndex)
		require.NoError(t, err)
	}

	q, err := engine.NextQuestion(ctx, testUserID, session.ID)
	require.NoError(t, err)
	assert.Nil(t, q, "NextQuestion should return nil when session is complete")
}

// --- SubmitAnswer ---

func TestSubmitAnswer_Correct(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	pack, err := repo.GetQuizPack(ctx, testPackID)
	require.NoError(t, err)

	firstQID := session.QuestionIDs[0]
	var correctIndex int
	for _, q := range pack.Questions {
		if q.ID == firstQID {
			correctIndex = q.CorrectIndex
			break
		}
	}

	result, err := engine.SubmitAnswer(ctx, testUserID, session.ID, firstQID, correctIndex)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Correct)
	assert.Equal(t, correctIndex, result.CorrectIndex)
}

func TestSubmitAnswer_Wrong(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	pack, err := repo.GetQuizPack(ctx, testPackID)
	require.NoError(t, err)

	firstQID := session.QuestionIDs[0]
	var correctIndex int
	for _, q := range pack.Questions {
		if q.ID == firstQID {
			correctIndex = q.CorrectIndex
			break
		}
	}

	wrongIndex := (correctIndex + 1) % 4
	result, err := engine.SubmitAnswer(ctx, testUserID, session.ID, firstQID, wrongIndex)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.Correct)
	assert.Equal(t, correctIndex, result.CorrectIndex)
}

func TestSubmitAnswer_UpdatesQuestionState(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	pack, err := repo.GetQuizPack(ctx, testPackID)
	require.NoError(t, err)

	firstQID := session.QuestionIDs[0]
	var correctIndex int
	for _, q := range pack.Questions {
		if q.ID == firstQID {
			correctIndex = q.CorrectIndex
			break
		}
	}

	_, err = engine.SubmitAnswer(ctx, testUserID, session.ID, firstQID, correctIndex)
	require.NoError(t, err)

	state, err := repo.GetQuestionState(ctx, testUserID, testPackID, firstQID)
	require.NoError(t, err)
	require.NotNil(t, state, "question state should have been created after answering")
	assert.Equal(t, model.AnswerResultCorrect, state.LastResult)
	assert.Equal(t, 1, state.RepetitionCount)
}

func TestSubmitAnswer_UpdatesTopicStats(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	pack, err := repo.GetQuizPack(ctx, testPackID)
	require.NoError(t, err)

	firstQID := session.QuestionIDs[0]
	var firstQ model.Question
	for _, q := range pack.Questions {
		if q.ID == firstQID {
			firstQ = q
			break
		}
	}

	_, err = engine.SubmitAnswer(ctx, testUserID, session.ID, firstQID, firstQ.CorrectIndex)
	require.NoError(t, err)

	stats, err := repo.GetTopicStats(ctx, testUserID, testPackID, firstQ.TopicID)
	require.NoError(t, err)
	require.NotNil(t, stats, "topic stats should be updated after answering")
	assert.Equal(t, 1, stats.TotalAttempts)
	assert.Equal(t, 1, stats.CorrectCount)
	assert.Equal(t, 1, stats.CurrentStreak)
	assert.Equal(t, 1, stats.BestStreak)
}

func TestSubmitAnswer_InvalidQuestionID(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	_, err = engine.SubmitAnswer(ctx, testUserID, session.ID, "not-in-session", 0)
	assert.Error(t, err)
}

// --- GetSessionStatus ---

func TestGetSessionStatus(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	status, err := engine.GetSessionStatus(ctx, testUserID, session.ID)
	require.NoError(t, err)
	require.NotNil(t, status)

	assert.Equal(t, 3, status.TotalQuestions)
	assert.Equal(t, 0, status.Answered)
	assert.Equal(t, 0, status.Correct)
	assert.Equal(t, model.QuizSessionStatusInProgress, status.Status)
	assert.Equal(t, 45*60, status.TimeLimitSec)
}

func TestGetSessionStatus_AfterAnswers(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	pack, err := repo.GetQuizPack(ctx, testPackID)
	require.NoError(t, err)
	questionMap := make(map[string]model.Question)
	for _, q := range pack.Questions {
		questionMap[q.ID] = q
	}

	// Answer first question correctly.
	firstQID := session.QuestionIDs[0]
	firstQ := questionMap[firstQID]
	_, err = engine.SubmitAnswer(ctx, testUserID, session.ID, firstQID, firstQ.CorrectIndex)
	require.NoError(t, err)

	// Answer second question wrong.
	secondQID := session.QuestionIDs[1]
	secondQ := questionMap[secondQID]
	wrongIndex := (secondQ.CorrectIndex + 1) % 4
	_, err = engine.SubmitAnswer(ctx, testUserID, session.ID, secondQID, wrongIndex)
	require.NoError(t, err)

	status, err := engine.GetSessionStatus(ctx, testUserID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, status.Answered)
	assert.Equal(t, 1, status.Correct)
}

// --- ResumeSession ---

func TestResumeSession(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	resumed, err := engine.ResumeSession(ctx, testUserID, session.ID)
	require.NoError(t, err)
	require.NotNil(t, resumed)
	assert.Equal(t, session.ID, resumed.ID)
	assert.Equal(t, model.QuizSessionStatusInProgress, resumed.Status)
}

func TestResumeSession_NotFound(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	_, err := engine.ResumeSession(ctx, testUserID, "nonexistent-session")
	assert.Error(t, err)
}

func TestResumeSession_CompletedSession(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	pack, err := repo.GetQuizPack(ctx, testPackID)
	require.NoError(t, err)
	questionMap := make(map[string]model.Question)
	for _, q := range pack.Questions {
		questionMap[q.ID] = q
	}

	// Answer all questions.
	for _, qID := range session.QuestionIDs {
		q := questionMap[qID]
		_, err := engine.SubmitAnswer(ctx, testUserID, session.ID, qID, q.CorrectIndex)
		require.NoError(t, err)
	}

	_, err = engine.ResumeSession(ctx, testUserID, session.ID)
	assert.Error(t, err, "should error when trying to resume a completed session")
}

// --- AbandonSession ---

func TestAbandonSession(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	err = engine.AbandonSession(ctx, testUserID, session.ID)
	require.NoError(t, err)

	// Verify status is abandoned.
	status, err := engine.GetSessionStatus(ctx, testUserID, session.ID)
	require.NoError(t, err)
	assert.Equal(t, model.QuizSessionStatusAbandoned, status.Status)
}

func TestAbandonSession_NotFound(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	err := engine.AbandonSession(ctx, testUserID, "nonexistent")
	assert.Error(t, err)
}

// --- StudySession logging ---

func TestSubmitAnswer_CreatesStudySession(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	pack, err := repo.GetQuizPack(ctx, testPackID)
	require.NoError(t, err)

	firstQID := session.QuestionIDs[0]
	var correctIndex int
	for _, q := range pack.Questions {
		if q.ID == firstQID {
			correctIndex = q.CorrectIndex
			break
		}
	}

	_, err = engine.SubmitAnswer(ctx, testUserID, session.ID, firstQID, correctIndex)
	require.NoError(t, err)

	studySession, err := repo.GetSession(ctx, testUserID, session.ID)
	require.NoError(t, err)
	require.NotNil(t, studySession, "a StudySession should exist after submitting an answer")

	assert.Equal(t, session.ID, studySession.ID)
	assert.Equal(t, testUserID, studySession.UserID)
	assert.Equal(t, testPackID, studySession.PackID)
	assert.Equal(t, session.Mode, studySession.Mode)
	assert.Len(t, studySession.Attempts, 1)
	assert.Equal(t, firstQID, studySession.Attempts[0].QuestionID)
	assert.Equal(t, correctIndex, studySession.Attempts[0].AnswerIndex)
	assert.True(t, studySession.Attempts[0].Correct)
	assert.Equal(t, 1, studySession.CorrectCount)
	assert.Nil(t, studySession.EndedAt, "EndedAt should be nil while session is still in_progress")
}

func TestAbandonSession_EndsStudySession(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	err = engine.AbandonSession(ctx, testUserID, session.ID)
	require.NoError(t, err)

	studySession, err := repo.GetSession(ctx, testUserID, session.ID)
	require.NoError(t, err)
	require.NotNil(t, studySession, "a StudySession should exist after abandoning")

	assert.NotNil(t, studySession.EndedAt, "EndedAt should be set after abandoning a session")
}

// --- Spaced Repetition Priority ---

func TestPracticeMode_SpacedRepetitionPriority(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	now := time.Now()

	// q003 is most overdue (furthest in the past).
	// q001 is slightly overdue.
	// q002 is not yet due.
	// q004 and q005 have no state (unseen).
	states := []model.QuestionState{
		{
			UserID:          testUserID,
			PackID:          testPackID,
			QuestionID:      "q001",
			EaseFactor:      2.5,
			IntervalDays:    1,
			RepetitionCount: 1,
			NextReviewAt:    now.Add(-1 * time.Hour),
			LastResult:      model.AnswerResultCorrect,
			LastReviewedAt:  now.Add(-25 * time.Hour),
		},
		{
			UserID:          testUserID,
			PackID:          testPackID,
			QuestionID:      "q002",
			EaseFactor:      2.5,
			IntervalDays:    6,
			RepetitionCount: 2,
			NextReviewAt:    now.Add(24 * time.Hour), // not yet due
			LastResult:      model.AnswerResultCorrect,
			LastReviewedAt:  now.Add(-5 * 24 * time.Hour),
		},
		{
			UserID:          testUserID,
			PackID:          testPackID,
			QuestionID:      "q003",
			EaseFactor:      2.5,
			IntervalDays:    1,
			RepetitionCount: 1,
			NextReviewAt:    now.Add(-48 * time.Hour), // most overdue
			LastResult:      model.AnswerResultCorrect,
			LastReviewedAt:  now.Add(-49 * time.Hour),
		},
	}

	for _, s := range states {
		err := repo.UpdateQuestionState(ctx, s)
		require.NoError(t, err)
	}

	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModePractice, core.QuizOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, session.QuestionIDs)

	// q003 must appear first (most overdue).
	assert.Equal(t, "q003", session.QuestionIDs[0], "most overdue question should appear first")
}

// --- FindResumableSession ---

func TestFindResumableSession(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	// Start two quiz sessions for the same user.
	session1, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	// Small sleep to ensure different timestamps.
	time.Sleep(2 * time.Millisecond)

	session2, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	// FindResumableSession should return the most recent in_progress session.
	found, err := engine.FindResumableSession(ctx, testUserID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, session2.ID, found.ID, "should return the most recently started session")
	assert.Equal(t, model.QuizSessionStatusInProgress, found.Status)

	_ = session1 // keep reference to suppress unused variable warning
}

func TestFindResumableSession_None(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	// No sessions exist for this user.
	found, err := engine.FindResumableSession(ctx, testUserID)
	require.NoError(t, err)
	assert.Nil(t, found, "should return nil when no in-progress sessions exist")
}

func TestFindResumableSession_CompletedSessionIgnored(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	// Start a session and complete it.
	session, err := engine.StartQuiz(ctx, testUserID, testPackID, model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	pack, err := repo.GetQuizPack(ctx, testPackID)
	require.NoError(t, err)
	questionMap := make(map[string]model.Question)
	for _, q := range pack.Questions {
		questionMap[q.ID] = q
	}

	for _, qID := range session.QuestionIDs {
		q := questionMap[qID]
		_, err = engine.SubmitAnswer(ctx, testUserID, session.ID, qID, q.CorrectIndex)
		require.NoError(t, err)
	}

	// Session is now completed; FindResumableSession should return nil.
	found, err := engine.FindResumableSession(ctx, testUserID)
	require.NoError(t, err)
	assert.Nil(t, found, "completed sessions should not be resumable")
}
