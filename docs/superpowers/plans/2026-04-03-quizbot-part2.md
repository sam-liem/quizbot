# QuizBot Implementation Plan — Part 2 (Tasks 6-10)

Continues from Part 1. All types, interfaces, and implementations from Tasks 1-5 are assumed complete.

---

## Task 6: Quiz Engine

**Goal:** Implement the core quiz engine that orchestrates quiz sessions — starting quizzes, delivering questions, scoring answers, and managing session lifecycle. Uses a mock repository for isolated testing.

### Steps

- [ ] **6.1 — Create `internal/store/mock_repository.go` — mock repository for testing**

```go
// internal/store/mock_repository.go
package store

import (
	"context"
	"fmt"
	"sync"

	"github.com/sam-liem/quizbot/internal/model"
)

// MockRepository is an in-memory implementation of Repository for testing.
type MockRepository struct {
	mu sync.Mutex

	QuizPacks      map[string]model.QuizPack
	QuestionStates map[string]model.QuestionState // key: "userID|packID|questionID"
	TopicStats     map[string]model.TopicStats    // key: "userID|packID|topicID"
	StudySessions  map[string]model.StudySession   // key: "userID|sessionID"
	QuizSessions   map[string]model.QuizSession    // key: "userID|sessionID"
	Preferences    map[string]model.UserPreferences // key: userID
}

// NewMockRepository creates an empty MockRepository.
func NewMockRepository() *MockRepository {
	return &MockRepository{
		QuizPacks:      make(map[string]model.QuizPack),
		QuestionStates: make(map[string]model.QuestionState),
		TopicStats:     make(map[string]model.TopicStats),
		StudySessions:  make(map[string]model.StudySession),
		QuizSessions:   make(map[string]model.QuizSession),
		Preferences:    make(map[string]model.UserPreferences),
	}
}

func questionStateKey(userID, packID, questionID string) string {
	return userID + "|" + packID + "|" + questionID
}

func topicStatsKey(userID, packID, topicID string) string {
	return userID + "|" + packID + "|" + topicID
}

func sessionKey(userID, sessionID string) string {
	return userID + "|" + sessionID
}

func (m *MockRepository) SaveQuizPack(_ context.Context, pack model.QuizPack) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QuizPacks[pack.ID] = pack
	return nil
}

func (m *MockRepository) GetQuizPack(_ context.Context, packID string) (*model.QuizPack, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pack, ok := m.QuizPacks[packID]
	if !ok {
		return nil, nil
	}
	return &pack, nil
}

func (m *MockRepository) ListQuizPacks(_ context.Context) ([]model.QuizPack, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	packs := make([]model.QuizPack, 0, len(m.QuizPacks))
	for _, p := range m.QuizPacks {
		packs = append(packs, p)
	}
	return packs, nil
}

func (m *MockRepository) GetQuestionState(_ context.Context, userID, packID, questionID string) (*model.QuestionState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.QuestionStates[questionStateKey(userID, packID, questionID)]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

func (m *MockRepository) UpdateQuestionState(_ context.Context, state model.QuestionState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QuestionStates[questionStateKey(state.UserID, state.PackID, state.QuestionID)] = state
	return nil
}

func (m *MockRepository) GetTopicStats(_ context.Context, userID, packID, topicID string) (*model.TopicStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stats, ok := m.TopicStats[topicStatsKey(userID, packID, topicID)]
	if !ok {
		return nil, nil
	}
	return &stats, nil
}

func (m *MockRepository) UpdateTopicStats(_ context.Context, stats model.TopicStats) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TopicStats[topicStatsKey(stats.UserID, stats.PackID, stats.TopicID)] = stats
	return nil
}

func (m *MockRepository) ListTopicStats(_ context.Context, userID, packID string) ([]model.TopicStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []model.TopicStats
	prefix := userID + "|" + packID + "|"
	for key, stats := range m.TopicStats {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			result = append(result, stats)
		}
	}
	return result, nil
}

func (m *MockRepository) CreateSession(_ context.Context, session model.StudySession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := sessionKey(session.UserID, session.ID)
	if _, exists := m.StudySessions[key]; exists {
		return fmt.Errorf("session %q already exists", session.ID)
	}
	m.StudySessions[key] = session
	return nil
}

func (m *MockRepository) GetSession(_ context.Context, userID, sessionID string) (*model.StudySession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.StudySessions[sessionKey(userID, sessionID)]
	if !ok {
		return nil, nil
	}
	return &session, nil
}

func (m *MockRepository) UpdateSession(_ context.Context, session model.StudySession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StudySessions[sessionKey(session.UserID, session.ID)] = session
	return nil
}

func (m *MockRepository) SaveQuizSession(_ context.Context, session model.QuizSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.QuizSessions[sessionKey(session.UserID, session.ID)] = session
	return nil
}

func (m *MockRepository) GetQuizSession(_ context.Context, userID, sessionID string) (*model.QuizSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.QuizSessions[sessionKey(userID, sessionID)]
	if !ok {
		return nil, nil
	}
	return &session, nil
}

func (m *MockRepository) GetPreferences(_ context.Context, userID string) (*model.UserPreferences, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefs, ok := m.Preferences[userID]
	if !ok {
		defaults := model.DefaultPreferences(userID)
		return &defaults, nil
	}
	return &prefs, nil
}

func (m *MockRepository) UpdatePreferences(_ context.Context, prefs model.UserPreferences) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Preferences[prefs.UserID] = prefs
	return nil
}
```

- [ ] **6.2 — Verify mock repository compiles and satisfies the interface**

```bash
cd /Users/samliem/LIFE_IN_UK
go build ./internal/store/...
```

Expected: clean build. `MockRepository` satisfies `Repository`.

- [ ] **6.3 — Write failing tests FIRST: `internal/core/engine_test.go`**

```go
// internal/core/engine_test.go
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

func setupEngine(t *testing.T) (*core.QuizEngine, *store.MockRepository) {
	t.Helper()
	repo := store.NewMockRepository()
	engine := core.NewQuizEngine(repo)

	pack := model.QuizPack{
		ID:          "test-pack",
		Name:        "Test Pack",
		Description: "A test quiz pack",
		Version:     "1.0.0",
		TestFormat: model.TestFormat{
			QuestionCount: 3,
			PassMarkPct:   75.0,
			TimeLimitSec:  300,
		},
		Topics: []model.Topic{
			{ID: "t1", Name: "Topic 1"},
			{ID: "t2", Name: "Topic 2"},
		},
		Questions: []model.Question{
			{
				ID: "q001", TopicID: "t1",
				Text:         "Question 1?",
				Choices:      []string{"A", "B", "C", "D"},
				CorrectIndex: 0,
				Explanation:  "A is correct.",
			},
			{
				ID: "q002", TopicID: "t1",
				Text:         "Question 2?",
				Choices:      []string{"A", "B", "C", "D"},
				CorrectIndex: 1,
				Explanation:  "B is correct.",
			},
			{
				ID: "q003", TopicID: "t2",
				Text:         "Question 3?",
				Choices:      []string{"A", "B", "C", "D"},
				CorrectIndex: 2,
				Explanation:  "C is correct.",
			},
			{
				ID: "q004", TopicID: "t2",
				Text:         "Question 4?",
				Choices:      []string{"A", "B", "C", "D"},
				CorrectIndex: 3,
				Explanation:  "D is correct.",
			},
			{
				ID: "q005", TopicID: "t1",
				Text:         "Question 5?",
				Choices:      []string{"A", "B", "C", "D"},
				CorrectIndex: 0,
				Explanation:  "A is correct.",
			},
		},
	}

	ctx := context.Background()
	require.NoError(t, repo.SaveQuizPack(ctx, pack))

	return engine, repo
}

func TestStartQuiz_MockMode(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	assert.NotEmpty(t, session.ID)
	assert.Equal(t, "user1", session.UserID)
	assert.Equal(t, "test-pack", session.PackID)
	assert.Equal(t, model.SessionModeMock, session.Mode)
	assert.Equal(t, model.QuizSessionStatusInProgress, session.Status)
	assert.Equal(t, 3, len(session.QuestionIDs)) // uses pack's test_format.question_count
	assert.Equal(t, 300, session.TimeLimitSec)    // uses pack's test_format.time_limit_sec
	assert.Equal(t, 0, session.CurrentIndex)
}

func TestStartQuiz_PracticeMode(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	opts := core.QuizOptions{
		QuestionCount: 2,
	}
	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModePractice, opts)
	require.NoError(t, err)

	assert.Equal(t, model.SessionModePractice, session.Mode)
	assert.Equal(t, 2, len(session.QuestionIDs))
	assert.Equal(t, 0, session.TimeLimitSec) // practice mode is untimed
}

func TestStartQuiz_PracticeMode_TopicFilter(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	opts := core.QuizOptions{
		QuestionCount: 10, // more than available, should cap
		TopicID:       "t2",
	}
	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModePractice, opts)
	require.NoError(t, err)

	// Only 2 questions in topic t2 (q003, q004).
	assert.Equal(t, 2, len(session.QuestionIDs))
	for _, qid := range session.QuestionIDs {
		assert.Contains(t, []string{"q003", "q004"}, qid)
	}
}

func TestStartQuiz_PackNotFound(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	_, err := engine.StartQuiz(ctx, "user1", "nonexistent", model.SessionModeMock, core.QuizOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestNextQuestion(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	q, err := engine.NextQuestion(ctx, "user1", session.ID)
	require.NoError(t, err)
	require.NotNil(t, q)
	assert.Equal(t, session.QuestionIDs[0], q.ID)
}

func TestNextQuestion_SessionCompleted(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	// Answer all questions.
	for i := range session.QuestionIDs {
		_, err := engine.NextQuestion(ctx, "user1", session.ID)
		require.NoError(t, err)

		_, err = engine.SubmitAnswer(ctx, "user1", session.ID, session.QuestionIDs[i], 0)
		require.NoError(t, err)
	}

	// No more questions.
	q, err := engine.NextQuestion(ctx, "user1", session.ID)
	require.NoError(t, err)
	assert.Nil(t, q)
}

func TestSubmitAnswer_Correct(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	_, err = engine.NextQuestion(ctx, "user1", session.ID)
	require.NoError(t, err)

	qid := session.QuestionIDs[0]

	// Find the correct index for this question.
	pack, _ := engine.Repo().GetQuizPack(ctx, "test-pack")
	var correctIndex int
	var explanation string
	for _, q := range pack.Questions {
		if q.ID == qid {
			correctIndex = q.CorrectIndex
			explanation = q.Explanation
			break
		}
	}

	result, err := engine.SubmitAnswer(ctx, "user1", session.ID, qid, correctIndex)
	require.NoError(t, err)

	assert.True(t, result.Correct)
	assert.Equal(t, correctIndex, result.CorrectIndex)
	assert.Equal(t, explanation, result.Explanation)
}

func TestSubmitAnswer_Wrong(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	_, err = engine.NextQuestion(ctx, "user1", session.ID)
	require.NoError(t, err)

	qid := session.QuestionIDs[0]

	// Find the correct index and pick a wrong one.
	pack, _ := engine.Repo().GetQuizPack(ctx, "test-pack")
	var correctIndex int
	for _, q := range pack.Questions {
		if q.ID == qid {
			correctIndex = q.CorrectIndex
			break
		}
	}
	wrongIndex := (correctIndex + 1) % 4

	result, err := engine.SubmitAnswer(ctx, "user1", session.ID, qid, wrongIndex)
	require.NoError(t, err)

	assert.False(t, result.Correct)
	assert.Equal(t, correctIndex, result.CorrectIndex)
}

func TestSubmitAnswer_UpdatesQuestionState(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModePractice, core.QuizOptions{QuestionCount: 1})
	require.NoError(t, err)

	_, err = engine.NextQuestion(ctx, "user1", session.ID)
	require.NoError(t, err)

	qid := session.QuestionIDs[0]

	// Submit correct answer.
	pack, _ := repo.GetQuizPack(ctx, "test-pack")
	var correctIndex int
	for _, q := range pack.Questions {
		if q.ID == qid {
			correctIndex = q.CorrectIndex
			break
		}
	}

	_, err = engine.SubmitAnswer(ctx, "user1", session.ID, qid, correctIndex)
	require.NoError(t, err)

	// Verify question state was created/updated.
	state, err := repo.GetQuestionState(ctx, "user1", "test-pack", qid)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, model.AnswerResultCorrect, state.LastResult)
	assert.Equal(t, 1, state.RepetitionCount)
}

func TestSubmitAnswer_UpdatesTopicStats(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModePractice, core.QuizOptions{QuestionCount: 1})
	require.NoError(t, err)

	_, err = engine.NextQuestion(ctx, "user1", session.ID)
	require.NoError(t, err)

	qid := session.QuestionIDs[0]

	// Find the question's topic and correct index.
	pack, _ := repo.GetQuizPack(ctx, "test-pack")
	var correctIndex int
	var topicID string
	for _, q := range pack.Questions {
		if q.ID == qid {
			correctIndex = q.CorrectIndex
			topicID = q.TopicID
			break
		}
	}

	_, err = engine.SubmitAnswer(ctx, "user1", session.ID, qid, correctIndex)
	require.NoError(t, err)

	stats, err := repo.GetTopicStats(ctx, "user1", "test-pack", topicID)
	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 1, stats.TotalAttempts)
	assert.Equal(t, 1, stats.CorrectCount)
}

func TestSubmitAnswer_InvalidQuestionID(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	_, err = engine.NextQuestion(ctx, "user1", session.ID)
	require.NoError(t, err)

	_, err = engine.SubmitAnswer(ctx, "user1", session.ID, "nonexistent", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not part of session")
}

func TestGetSessionStatus(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	status, err := engine.GetSessionStatus(ctx, "user1", session.ID)
	require.NoError(t, err)

	assert.Equal(t, 3, status.TotalQuestions)
	assert.Equal(t, 0, status.Answered)
	assert.Equal(t, 0, status.Correct)
	assert.Equal(t, model.QuizSessionStatusInProgress, status.Status)
	assert.Equal(t, 300, status.TimeLimitSec)
}

func TestGetSessionStatus_AfterAnswers(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	// Answer first question correctly.
	_, err = engine.NextQuestion(ctx, "user1", session.ID)
	require.NoError(t, err)

	qid := session.QuestionIDs[0]
	pack, _ := engine.Repo().GetQuizPack(ctx, "test-pack")
	var correctIndex int
	for _, q := range pack.Questions {
		if q.ID == qid {
			correctIndex = q.CorrectIndex
			break
		}
	}
	_, err = engine.SubmitAnswer(ctx, "user1", session.ID, qid, correctIndex)
	require.NoError(t, err)

	status, err := engine.GetSessionStatus(ctx, "user1", session.ID)
	require.NoError(t, err)

	assert.Equal(t, 3, status.TotalQuestions)
	assert.Equal(t, 1, status.Answered)
	assert.Equal(t, 1, status.Correct)
}

func TestResumeSession(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	// Answer first question.
	_, err = engine.NextQuestion(ctx, "user1", session.ID)
	require.NoError(t, err)
	_, err = engine.SubmitAnswer(ctx, "user1", session.ID, session.QuestionIDs[0], 0)
	require.NoError(t, err)

	// Resume the session.
	resumed, err := engine.ResumeSession(ctx, "user1", session.ID)
	require.NoError(t, err)

	assert.Equal(t, session.ID, resumed.ID)
	assert.Equal(t, model.QuizSessionStatusInProgress, resumed.Status)
	assert.Equal(t, 1, resumed.CurrentIndex) // should be at second question
}

func TestResumeSession_NotFound(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	_, err := engine.ResumeSession(ctx, "user1", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestResumeSession_CompletedSession(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	// Answer all questions.
	for _, qid := range session.QuestionIDs {
		_, err = engine.NextQuestion(ctx, "user1", session.ID)
		require.NoError(t, err)
		_, err = engine.SubmitAnswer(ctx, "user1", session.ID, qid, 0)
		require.NoError(t, err)
	}

	_, err = engine.ResumeSession(ctx, "user1", session.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in progress")
}

func TestAbandonSession(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()

	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModeMock, core.QuizOptions{})
	require.NoError(t, err)

	err = engine.AbandonSession(ctx, "user1", session.ID)
	require.NoError(t, err)

	qs, err := repo.GetQuizSession(ctx, "user1", session.ID)
	require.NoError(t, err)
	assert.Equal(t, model.QuizSessionStatusAbandoned, qs.Status)
}

func TestAbandonSession_NotFound(t *testing.T) {
	engine, _ := setupEngine(t)
	ctx := context.Background()

	err := engine.AbandonSession(ctx, "user1", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPracticeMode_SpacedRepetitionPriority(t *testing.T) {
	engine, repo := setupEngine(t)
	ctx := context.Background()
	now := time.Now().UTC()

	// Pre-seed question states so q003 is most overdue.
	states := []model.QuestionState{
		{
			UserID: "user1", PackID: "test-pack", QuestionID: "q001",
			EaseFactor: 2.5, IntervalDays: 1, RepetitionCount: 1,
			NextReviewAt: now.Add(24 * time.Hour), // not due
			LastResult: model.AnswerResultCorrect, LastReviewedAt: now,
		},
		{
			UserID: "user1", PackID: "test-pack", QuestionID: "q002",
			EaseFactor: 2.5, IntervalDays: 1, RepetitionCount: 1,
			NextReviewAt: now.Add(-1 * time.Hour), // 1 hour overdue
			LastResult: model.AnswerResultCorrect, LastReviewedAt: now.Add(-25 * time.Hour),
		},
		{
			UserID: "user1", PackID: "test-pack", QuestionID: "q003",
			EaseFactor: 1.5, IntervalDays: 1, RepetitionCount: 0,
			NextReviewAt: now.Add(-48 * time.Hour), // most overdue
			LastResult: model.AnswerResultWrong, LastReviewedAt: now.Add(-72 * time.Hour),
		},
	}
	for _, s := range states {
		require.NoError(t, repo.UpdateQuestionState(ctx, s))
	}

	// Start practice session with 3 questions.
	session, err := engine.StartQuiz(ctx, "user1", "test-pack", model.SessionModePractice, core.QuizOptions{QuestionCount: 3})
	require.NoError(t, err)

	// The first question should be the most overdue (q003).
	assert.Equal(t, "q003", session.QuestionIDs[0])
}
```

- [ ] **6.4 — Verify tests fail (TDD red phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./internal/core/... 2>&1 | head -20
```

Expected: compilation errors because `QuizEngine`, `NewQuizEngine`, `QuizOptions`, `SubmitAnswerResult`, and `SessionStatus` do not exist.

- [ ] **6.5 — Implement `internal/core/engine.go`**

```go
// internal/core/engine.go
package core

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/sam-liem/quizbot/internal/model"
	"github.com/sam-liem/quizbot/internal/store"
)

// QuizOptions holds optional parameters for starting a quiz.
type QuizOptions struct {
	QuestionCount int    // custom question count (practice mode)
	TopicID       string // optional topic filter (practice mode)
}

// SubmitAnswerResult is the result returned after submitting an answer.
type SubmitAnswerResult struct {
	Correct      bool   `json:"correct"`
	CorrectIndex int    `json:"correct_index"`
	Explanation  string `json:"explanation"`
}

// SessionStatus describes the current state of a quiz session.
type SessionStatus struct {
	TotalQuestions int                    `json:"total_questions"`
	Answered       int                    `json:"answered"`
	Correct        int                    `json:"correct"`
	Status         model.QuizSessionStatus `json:"status"`
	TimeLimitSec   int                    `json:"time_limit_sec"`
	StartedAt      time.Time              `json:"started_at"`
}

// QuizEngine orchestrates quiz sessions, question delivery, answer
// scoring, and session lifecycle management.
type QuizEngine struct {
	repo store.Repository
}

// NewQuizEngine creates a QuizEngine with injected dependencies.
func NewQuizEngine(repo store.Repository) *QuizEngine {
	return &QuizEngine{repo: repo}
}

// Repo returns the underlying repository. Useful for test assertions.
func (e *QuizEngine) Repo() store.Repository {
	return e.repo
}

// StartQuiz creates a new quiz session. Mock mode uses the pack's
// test_format (question_count, time_limit_sec) with random selection.
// Practice mode uses the provided options and respects spaced
// repetition priority.
func (e *QuizEngine) StartQuiz(ctx context.Context, userID, packID string, mode model.SessionMode, opts QuizOptions) (*model.QuizSession, error) {
	pack, err := e.repo.GetQuizPack(ctx, packID)
	if err != nil {
		return nil, fmt.Errorf("getting quiz pack: %w", err)
	}
	if pack == nil {
		return nil, fmt.Errorf("quiz pack %q not found", packID)
	}

	var questionIDs []string
	var timeLimitSec int

	switch mode {
	case model.SessionModeMock:
		questionIDs = e.selectMockQuestions(pack)
		timeLimitSec = pack.TestFormat.TimeLimitSec
	case model.SessionModePractice:
		var err error
		questionIDs, err = e.selectPracticeQuestions(ctx, userID, pack, opts)
		if err != nil {
			return nil, fmt.Errorf("selecting practice questions: %w", err)
		}
		timeLimitSec = 0 // practice mode is untimed
	default:
		return nil, fmt.Errorf("unsupported session mode %q", mode)
	}

	if len(questionIDs) == 0 {
		return nil, fmt.Errorf("no questions available for the given criteria")
	}

	session := model.QuizSession{
		ID:           generateID(),
		UserID:       userID,
		PackID:       packID,
		Mode:         mode,
		QuestionIDs:  questionIDs,
		CurrentIndex: 0,
		Answers:      make(map[string]int),
		StartedAt:    time.Now().UTC(),
		TimeLimitSec: timeLimitSec,
		Status:       model.QuizSessionStatusInProgress,
	}

	if err := e.repo.SaveQuizSession(ctx, session); err != nil {
		return nil, fmt.Errorf("saving quiz session: %w", err)
	}

	return &session, nil
}

// selectMockQuestions randomly selects question_count questions from the
// pack for a mock test.
func (e *QuizEngine) selectMockQuestions(pack *model.QuizPack) []string {
	count := pack.TestFormat.QuestionCount
	if count > len(pack.Questions) {
		count = len(pack.Questions)
	}

	// Shuffle and take the first `count`.
	indices := rand.Perm(len(pack.Questions))
	ids := make([]string, count)
	for i := 0; i < count; i++ {
		ids[i] = pack.Questions[indices[i]].ID
	}
	return ids
}

// selectPracticeQuestions selects questions for practice mode, respecting
// spaced repetition priority and optional topic filter.
func (e *QuizEngine) selectPracticeQuestions(ctx context.Context, userID string, pack *model.QuizPack, opts QuizOptions) ([]string, error) {
	// Filter questions by topic if specified.
	var candidates []model.Question
	for _, q := range pack.Questions {
		if opts.TopicID != "" && q.TopicID != opts.TopicID {
			continue
		}
		candidates = append(candidates, q)
	}

	count := opts.QuestionCount
	if count <= 0 {
		count = len(candidates)
	}
	if count > len(candidates) {
		count = len(candidates)
	}

	now := time.Now().UTC()

	// Gather question states for all candidates.
	type questionWithOverdue struct {
		id      string
		overdue time.Duration
		hasSate bool
	}
	var withStates []questionWithOverdue
	var unseen []string

	for _, q := range candidates {
		state, err := e.repo.GetQuestionState(ctx, userID, pack.ID, q.ID)
		if err != nil {
			return nil, fmt.Errorf("getting question state for %q: %w", q.ID, err)
		}
		if state == nil {
			unseen = append(unseen, q.ID)
		} else {
			overdue := now.Sub(state.NextReviewAt)
			withStates = append(withStates, questionWithOverdue{
				id:      q.ID,
				overdue: overdue,
				hasSate: true,
			})
		}
	}

	// Sort by most overdue first.
	for i := 0; i < len(withStates); i++ {
		for j := i + 1; j < len(withStates); j++ {
			if withStates[j].overdue > withStates[i].overdue {
				withStates[i], withStates[j] = withStates[j], withStates[i]
			}
		}
	}

	// Build the result: overdue questions first, then unseen questions.
	var result []string
	for _, ws := range withStates {
		if len(result) >= count {
			break
		}
		if ws.overdue > 0 {
			result = append(result, ws.id)
		}
	}

	// Mix in unseen questions.
	for _, id := range unseen {
		if len(result) >= count {
			break
		}
		result = append(result, id)
	}

	// If still under count, add non-overdue seen questions.
	for _, ws := range withStates {
		if len(result) >= count {
			break
		}
		if ws.overdue <= 0 {
			result = append(result, ws.id)
		}
	}

	return result, nil
}

// NextQuestion returns the next question in the session, or nil if the
// session is completed.
func (e *QuizEngine) NextQuestion(ctx context.Context, userID, sessionID string) (*model.Question, error) {
	session, err := e.repo.GetQuizSession(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("getting quiz session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("quiz session %q not found", sessionID)
	}

	if session.Status != model.QuizSessionStatusInProgress {
		return nil, nil
	}

	if session.CurrentIndex >= len(session.QuestionIDs) {
		return nil, nil
	}

	questionID := session.QuestionIDs[session.CurrentIndex]

	pack, err := e.repo.GetQuizPack(ctx, session.PackID)
	if err != nil {
		return nil, fmt.Errorf("getting quiz pack: %w", err)
	}
	if pack == nil {
		return nil, fmt.Errorf("quiz pack %q not found", session.PackID)
	}

	for _, q := range pack.Questions {
		if q.ID == questionID {
			return &q, nil
		}
	}

	return nil, fmt.Errorf("question %q not found in pack %q", questionID, session.PackID)
}

// SubmitAnswer scores an answer, updates spaced repetition state and
// topic stats, advances the session, and returns the result.
func (e *QuizEngine) SubmitAnswer(ctx context.Context, userID, sessionID, questionID string, answerIndex int) (*SubmitAnswerResult, error) {
	session, err := e.repo.GetQuizSession(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("getting quiz session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("quiz session %q not found", sessionID)
	}

	// Validate questionID is part of this session.
	found := false
	for _, qid := range session.QuestionIDs {
		if qid == questionID {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("question %q not part of session %q", questionID, sessionID)
	}

	// Look up the question in the pack.
	pack, err := e.repo.GetQuizPack(ctx, session.PackID)
	if err != nil {
		return nil, fmt.Errorf("getting quiz pack: %w", err)
	}
	if pack == nil {
		return nil, fmt.Errorf("quiz pack %q not found", session.PackID)
	}

	var question *model.Question
	for _, q := range pack.Questions {
		if q.ID == questionID {
			question = &q
			break
		}
	}
	if question == nil {
		return nil, fmt.Errorf("question %q not found in pack %q", questionID, session.PackID)
	}

	correct := answerIndex == question.CorrectIndex
	now := time.Now().UTC()

	// Update spaced repetition state.
	existingState, err := e.repo.GetQuestionState(ctx, userID, session.PackID, questionID)
	if err != nil {
		return nil, fmt.Errorf("getting question state: %w", err)
	}

	var state model.QuestionState
	if existingState != nil {
		state = *existingState
	} else {
		state = model.QuestionState{
			UserID:     userID,
			PackID:     session.PackID,
			QuestionID: questionID,
			EaseFactor: 2.5,
		}
	}

	state = CalculateNextReview(state, correct, now)
	if err := e.repo.UpdateQuestionState(ctx, state); err != nil {
		return nil, fmt.Errorf("updating question state: %w", err)
	}

	// Update topic stats.
	if err := e.updateTopicStats(ctx, userID, session.PackID, question.TopicID, correct); err != nil {
		return nil, fmt.Errorf("updating topic stats: %w", err)
	}

	// Update session state.
	session.Answers[questionID] = answerIndex
	session.CurrentIndex++

	if session.CurrentIndex >= len(session.QuestionIDs) {
		session.Status = model.QuizSessionStatusCompleted
	}

	if err := e.repo.SaveQuizSession(ctx, *session); err != nil {
		return nil, fmt.Errorf("saving quiz session: %w", err)
	}

	return &SubmitAnswerResult{
		Correct:      correct,
		CorrectIndex: question.CorrectIndex,
		Explanation:  question.Explanation,
	}, nil
}

// updateTopicStats updates the aggregate topic stats after an answer.
func (e *QuizEngine) updateTopicStats(ctx context.Context, userID, packID, topicID string, correct bool) error {
	stats, err := e.repo.GetTopicStats(ctx, userID, packID, topicID)
	if err != nil {
		return fmt.Errorf("getting topic stats: %w", err)
	}

	if stats == nil {
		stats = &model.TopicStats{
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

	if err := e.repo.UpdateTopicStats(ctx, *stats); err != nil {
		return fmt.Errorf("updating topic stats: %w", err)
	}

	return nil
}

// GetSessionStatus returns the current progress of a quiz session.
func (e *QuizEngine) GetSessionStatus(ctx context.Context, userID, sessionID string) (*SessionStatus, error) {
	session, err := e.repo.GetQuizSession(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("getting quiz session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("quiz session %q not found", sessionID)
	}

	pack, err := e.repo.GetQuizPack(ctx, session.PackID)
	if err != nil {
		return nil, fmt.Errorf("getting quiz pack: %w", err)
	}
	if pack == nil {
		return nil, fmt.Errorf("quiz pack %q not found", session.PackID)
	}

	// Count correct answers.
	correctCount := 0
	for qid, answerIdx := range session.Answers {
		for _, q := range pack.Questions {
			if q.ID == qid && answerIdx == q.CorrectIndex {
				correctCount++
				break
			}
		}
	}

	return &SessionStatus{
		TotalQuestions: len(session.QuestionIDs),
		Answered:       len(session.Answers),
		Correct:        correctCount,
		Status:         session.Status,
		TimeLimitSec:   session.TimeLimitSec,
		StartedAt:      session.StartedAt,
	}, nil
}

// ResumeSession resumes an in-progress quiz session.
func (e *QuizEngine) ResumeSession(ctx context.Context, userID, sessionID string) (*model.QuizSession, error) {
	session, err := e.repo.GetQuizSession(ctx, userID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("getting quiz session: %w", err)
	}
	if session == nil {
		return nil, fmt.Errorf("quiz session %q not found", sessionID)
	}
	if session.Status != model.QuizSessionStatusInProgress {
		return nil, fmt.Errorf("quiz session %q not in progress (status: %s)", sessionID, session.Status)
	}

	return session, nil
}

// AbandonSession marks a quiz session as abandoned.
func (e *QuizEngine) AbandonSession(ctx context.Context, userID, sessionID string) error {
	session, err := e.repo.GetQuizSession(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("getting quiz session: %w", err)
	}
	if session == nil {
		return fmt.Errorf("quiz session %q not found", sessionID)
	}

	session.Status = model.QuizSessionStatusAbandoned
	if err := e.repo.SaveQuizSession(ctx, *session); err != nil {
		return fmt.Errorf("saving quiz session: %w", err)
	}

	return nil
}

// generateID produces a unique session identifier using timestamp and
// random component.
func generateID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())
}
```

- [ ] **6.6 — Run tests (TDD green phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test -v ./internal/core/...
```

Expected: all tests pass (engine, repetition, and stats tests).

- [ ] **6.7 — Run full test suite to verify nothing is broken**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./...
```

Expected: all tests pass across all packages.

- [ ] **6.8 — Commit: "Add quiz engine with mock and practice modes"**

```bash
cd /Users/samliem/LIFE_IN_UK
git add internal/core/engine.go internal/core/engine_test.go internal/store/mock_repository.go
git commit -m "Add quiz engine with mock and practice modes

Implement QuizEngine with StartQuiz, NextQuestion, SubmitAnswer,
GetSessionStatus, ResumeSession, and AbandonSession. Mock mode uses
random selection with pack test_format. Practice mode respects spaced
repetition priority with optional topic filter. SubmitAnswer updates
question state and topic stats. Includes MockRepository for testing."
```

---

## Task 7: Explainer

**Goal:** Define the Explainer interface and implement StaticExplainer that returns pre-authored explanation text.

### Steps

- [ ] **7.1 — Write failing tests FIRST: `internal/explainer/static_test.go`**

```go
// internal/explainer/static_test.go
package explainer_test

import (
	"testing"

	"github.com/sam-liem/quizbot/internal/explainer"
	"github.com/sam-liem/quizbot/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestStaticExplainer_WithExplanation(t *testing.T) {
	e := explainer.NewStaticExplainer()

	question := model.Question{
		ID:           "q001",
		Text:         "When was the Magna Carta sealed?",
		Choices:      []string{"1205", "1210", "1215", "1220"},
		CorrectIndex: 2,
		Explanation:  "The Magna Carta was sealed by King John in 1215.",
	}

	result := e.Explain(question, 0, 2)
	assert.Equal(t, "The Magna Carta was sealed by King John in 1215.", result)
}

func TestStaticExplainer_EmptyExplanation(t *testing.T) {
	e := explainer.NewStaticExplainer()

	question := model.Question{
		ID:           "q002",
		Text:         "What is 2+2?",
		Choices:      []string{"3", "4", "5", "6"},
		CorrectIndex: 1,
		Explanation:  "",
	}

	result := e.Explain(question, 0, 1)
	assert.Equal(t, "The correct answer is: 4", result)
}

func TestStaticExplainer_EmptyExplanation_FirstChoice(t *testing.T) {
	e := explainer.NewStaticExplainer()

	question := model.Question{
		ID:           "q003",
		Text:         "What is the capital of France?",
		Choices:      []string{"Paris", "London", "Berlin"},
		CorrectIndex: 0,
		Explanation:  "",
	}

	result := e.Explain(question, 1, 0)
	assert.Equal(t, "The correct answer is: Paris", result)
}

func TestStaticExplainer_ImplementsInterface(t *testing.T) {
	var e explainer.Explainer = explainer.NewStaticExplainer()
	assert.NotNil(t, e)
}

func TestStaticExplainer_CorrectAnswerGenericMessage(t *testing.T) {
	e := explainer.NewStaticExplainer()

	question := model.Question{
		ID:           "q004",
		Text:         "Pick one",
		Choices:      []string{"A", "B"},
		CorrectIndex: 0,
		Explanation:  "",
	}

	// User answered correctly, no authored explanation.
	result := e.Explain(question, 0, 0)
	assert.Equal(t, "The correct answer is: A", result)
}

func TestStaticExplainer_WhitespaceOnlyExplanation(t *testing.T) {
	e := explainer.NewStaticExplainer()

	question := model.Question{
		ID:           "q005",
		Text:         "Pick one",
		Choices:      []string{"X", "Y"},
		CorrectIndex: 1,
		Explanation:  "   ",
	}

	// Whitespace-only explanation should be treated as empty.
	result := e.Explain(question, 0, 1)
	assert.Equal(t, "The correct answer is: Y", result)
}
```

- [ ] **7.2 — Verify tests fail (TDD red phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./internal/explainer/... 2>&1 | head -10
```

Expected: compilation errors because `explainer.Explainer`, `explainer.StaticExplainer`, and `explainer.NewStaticExplainer` do not exist.

- [ ] **7.3 — Create `internal/explainer/explainer.go` — Explainer interface**

```go
// internal/explainer/explainer.go
package explainer

import "github.com/sam-liem/quizbot/internal/model"

// Explainer generates an explanation for a quiz answer. Implementations
// range from simple static lookups to LLM-powered explanations.
//
// All output from an Explainer should be treated as potentially unsafe
// (especially LLM implementations) and passed through the sanitizer
// before reaching any output channel.
type Explainer interface {
	// Explain returns an explanation for why the correct answer is
	// correct. userAnswer is the index the user chose; correctAnswer
	// is the index of the correct choice.
	Explain(question model.Question, userAnswer, correctAnswer int) string
}
```

- [ ] **7.4 — Create `internal/explainer/static.go` — StaticExplainer**

```go
// internal/explainer/static.go
package explainer

import (
	"fmt"
	"strings"

	"github.com/sam-liem/quizbot/internal/model"
)

// StaticExplainer returns the question's pre-authored explanation field.
// If the explanation is empty or whitespace-only, it returns a generic
// message indicating the correct answer.
type StaticExplainer struct{}

// NewStaticExplainer creates a new StaticExplainer.
func NewStaticExplainer() *StaticExplainer {
	return &StaticExplainer{}
}

// Explain returns the pre-authored explanation for the question. If no
// explanation is available, returns "The correct answer is: <choice>".
func (e *StaticExplainer) Explain(question model.Question, userAnswer, correctAnswer int) string {
	trimmed := strings.TrimSpace(question.Explanation)
	if trimmed != "" {
		return trimmed
	}

	if correctAnswer >= 0 && correctAnswer < len(question.Choices) {
		return fmt.Sprintf("The correct answer is: %s", question.Choices[correctAnswer])
	}

	return "No explanation available."
}
```

- [ ] **7.5 — Run tests (TDD green phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test -v ./internal/explainer/...
```

Expected: all tests pass.

- [ ] **7.6 — Run full test suite**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./...
```

Expected: all tests pass across all packages.

- [ ] **7.7 — Commit: "Add Explainer interface and StaticExplainer"**

```bash
cd /Users/samliem/LIFE_IN_UK
git add internal/explainer/
git commit -m "Add Explainer interface and StaticExplainer

Define the Explainer interface with Explain(question, userAnswer,
correctAnswer) signature. StaticExplainer returns the question's
pre-authored explanation or a generic fallback message. Whitespace-only
explanations are treated as empty."
```

---

## Task 8: Security

**Goal:** Implement AES-256-GCM encryption for API key storage and a sanitizer for stripping bot commands, escaping HTML, and truncating output.

### Steps

- [ ] **8.1 — Write failing tests FIRST: `internal/security/crypto_test.go`**

```go
// internal/security/crypto_test.go
package security_test

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/sam-liem/quizbot/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTestKey(t *testing.T) string {
	t.Helper()
	key := make([]byte, 32) // AES-256
	_, err := rand.Read(key)
	require.NoError(t, err)
	return hex.EncodeToString(key)
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	key := generateTestKey(t)
	plaintext := "sk-my-secret-api-key-12345"

	ciphertext, err := security.Encrypt(plaintext, key)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)
	assert.NotEmpty(t, ciphertext)

	decrypted, err := security.Decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	key := generateTestKey(t)

	ciphertext, err := security.Encrypt("", key)
	require.NoError(t, err)

	decrypted, err := security.Decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Equal(t, "", decrypted)
}

func TestEncryptDecrypt_LongText(t *testing.T) {
	key := generateTestKey(t)
	plaintext := "This is a longer API key with special characters: !@#$%^&*()_+-=[]{}|;':\",./<>?"

	ciphertext, err := security.Encrypt(plaintext, key)
	require.NoError(t, err)

	decrypted, err := security.Decrypt(ciphertext, key)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := generateTestKey(t)
	key2 := generateTestKey(t)

	ciphertext, err := security.Encrypt("secret", key1)
	require.NoError(t, err)

	_, err = security.Decrypt(ciphertext, key2)
	assert.Error(t, err)
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := generateTestKey(t)

	ciphertext, err := security.Encrypt("secret", key)
	require.NoError(t, err)

	// Decode, tamper, re-encode.
	raw, err := hex.DecodeString(ciphertext)
	require.NoError(t, err)
	require.True(t, len(raw) > 5)

	// Flip a byte in the middle of the ciphertext.
	raw[len(raw)/2] ^= 0xFF
	tampered := hex.EncodeToString(raw)

	_, err = security.Decrypt(tampered, key)
	assert.Error(t, err)
}

func TestEncrypt_InvalidKey(t *testing.T) {
	_, err := security.Encrypt("secret", "too-short")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key")
}

func TestDecrypt_InvalidCiphertext(t *testing.T) {
	key := generateTestKey(t)

	_, err := security.Decrypt("not-hex", key)
	assert.Error(t, err)
}

func TestEncrypt_DifferentCiphertextEachTime(t *testing.T) {
	key := generateTestKey(t)
	plaintext := "same-plaintext"

	c1, err := security.Encrypt(plaintext, key)
	require.NoError(t, err)

	c2, err := security.Encrypt(plaintext, key)
	require.NoError(t, err)

	// GCM uses a random nonce, so ciphertexts should differ.
	assert.NotEqual(t, c1, c2)
}
```

- [ ] **8.2 — Write failing tests: `internal/security/sanitizer_test.go`**

```go
// internal/security/sanitizer_test.go
package security_test

import (
	"strings"
	"testing"

	"github.com/sam-liem/quizbot/internal/security"
	"github.com/stretchr/testify/assert"
)

func TestSanitize_BotCommandsStripped(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "start command",
			input: "Hello /start world",
			want:  "Hello  world",
		},
		{
			name:  "quiz command",
			input: "Try /quiz --mock now",
			want:  "Try  --mock now",
		},
		{
			name:  "resume command",
			input: "Please /resume your session",
			want:  "Please  your session",
		},
		{
			name:  "stats command",
			input: "Check /stats here",
			want:  "Check  here",
		},
		{
			name:  "packs command",
			input: "Run /packs list",
			want:  "Run  list",
		},
		{
			name:  "config command",
			input: "Set /config value",
			want:  "Set  value",
		},
		{
			name:  "help command",
			input: "Type /help for info",
			want:  "Type  for info",
		},
		{
			name:  "multiple commands",
			input: "/start and /quiz and /help",
			want:  " and  and ",
		},
		{
			name:  "no commands",
			input: "This is a normal explanation.",
			want:  "This is a normal explanation.",
		},
		{
			name:  "command at start of line",
			input: "/start immediately",
			want:  " immediately",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := security.Sanitize(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSanitize_HTMLEscaped(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "angle brackets",
			input: "<script>alert('xss')</script>",
			want:  "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name:  "ampersand",
			input: "Tom & Jerry",
			want:  "Tom &amp; Jerry",
		},
		{
			name:  "double quotes",
			input: `She said "hello"`,
			want:  "She said &#34;hello&#34;",
		},
		{
			name:  "mixed",
			input: `<b>bold & "quoted"</b>`,
			want:  "&lt;b&gt;bold &amp; &#34;quoted&#34;&lt;/b&gt;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := security.Sanitize(tt.input)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestSanitize_Truncation(t *testing.T) {
	// MaxSanitizedLength is 4096 characters.
	longInput := strings.Repeat("a", 5000)
	result := security.Sanitize(longInput)
	assert.Equal(t, 4096, len(result))
}

func TestSanitize_ShortStringNotTruncated(t *testing.T) {
	input := "Short explanation."
	result := security.Sanitize(input)
	assert.Equal(t, "Short explanation.", result)
}

func TestSanitize_EmptyString(t *testing.T) {
	result := security.Sanitize("")
	assert.Equal(t, "", result)
}

func TestSanitize_CommandAndHTMLCombined(t *testing.T) {
	input := "/start <script>alert('xss')</script>"
	result := security.Sanitize(input)
	// Command stripped, then HTML escaped.
	assert.Equal(t, " &lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;", result)
}
```

- [ ] **8.3 — Verify tests fail (TDD red phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./internal/security/... 2>&1 | head -10
```

Expected: compilation errors because `security.Encrypt`, `security.Decrypt`, and `security.Sanitize` do not exist.

- [ ] **8.4 — Create `internal/security/crypto.go`**

```go
// internal/security/crypto.go
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// Encrypt encrypts plaintext using AES-256-GCM. The key must be a
// hex-encoded 32-byte (64 hex character) string. Returns hex-encoded
// ciphertext with the nonce prepended.
func Encrypt(plaintext, hexKey string) (string, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return "", fmt.Errorf("decoding encryption key: %w", err)
	}
	if len(key) != 32 {
		return "", fmt.Errorf("encryption key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generating nonce: %w", err)
	}

	// Seal appends the ciphertext to nonce, so the result is nonce+ciphertext.
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	return hex.EncodeToString(sealed), nil
}

// Decrypt decrypts hex-encoded ciphertext that was encrypted with
// Encrypt. The key must be the same hex-encoded 32-byte string used
// for encryption.
func Decrypt(hexCiphertext, hexKey string) (string, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return "", fmt.Errorf("decoding encryption key: %w", err)
	}
	if len(key) != 32 {
		return "", fmt.Errorf("encryption key must be 32 bytes (64 hex chars), got %d bytes", len(key))
	}

	ciphertext, err := hex.DecodeString(hexCiphertext)
	if err != nil {
		return "", fmt.Errorf("decoding ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("creating AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("creating GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short: expected at least %d bytes, got %d", nonceSize, len(ciphertext))
	}

	nonce := ciphertext[:nonceSize]
	encrypted := ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("decrypting: %w", err)
	}

	return string(plaintext), nil
}
```

- [ ] **8.5 — Create `internal/security/sanitizer.go`**

```go
// internal/security/sanitizer.go
package security

import (
	"html"
	"strings"
)

// MaxSanitizedLength is the maximum length of sanitized output. Strings
// longer than this are truncated.
const MaxSanitizedLength = 4096

// botCommands is the set of Telegram bot commands that must be stripped
// from any output to prevent command injection.
var botCommands = []string{
	"/start",
	"/quiz",
	"/resume",
	"/stats",
	"/packs",
	"/config",
	"/help",
}

// Sanitize processes untrusted text (e.g., LLM output) to make it safe
// for display in Telegram or CLI. It:
//  1. Strips bot command patterns to prevent command injection.
//  2. Escapes HTML special characters.
//  3. Truncates to MaxSanitizedLength.
func Sanitize(input string) string {
	if input == "" {
		return ""
	}

	// Step 1: Strip bot commands.
	result := input
	for _, cmd := range botCommands {
		result = strings.ReplaceAll(result, cmd, "")
	}

	// Step 2: Escape HTML special characters.
	result = html.EscapeString(result)

	// Step 3: Truncate to max length.
	if len(result) > MaxSanitizedLength {
		result = result[:MaxSanitizedLength]
	}

	return result
}
```

- [ ] **8.6 — Run tests (TDD green phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test -v ./internal/security/...
```

Expected: all tests pass.

- [ ] **8.7 — Run full test suite**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./...
```

Expected: all tests pass across all packages.

- [ ] **8.8 — Commit: "Add AES-256-GCM encryption and output sanitizer"**

```bash
cd /Users/samliem/LIFE_IN_UK
git add internal/security/
git commit -m "Add AES-256-GCM encryption and output sanitizer

Implement Encrypt/Decrypt using AES-256-GCM with random nonces and
hex encoding. Implement Sanitize to strip bot commands, escape HTML
special chars, and truncate to 4096 chars. Both modules have
comprehensive table-driven tests."
```

---

## Task 9: Config Loading

**Goal:** Implement configuration loading from YAML file with validation, and create the example config template.

### Steps

- [ ] **9.1 — Write failing tests FIRST: `config/config_test.go`**

```go
// config/config_test.go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sam-liem/quizbot/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
	return path
}

func TestLoadConfig_ValidFull(t *testing.T) {
	yaml := `
telegram_bot_token: "123456:ABC-DEF"
sqlite_path: "/tmp/quizbot.db"
llm_api_key: "sk-abc123"
listen_address: ":8080"
log_level: "info"
encryption_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
`
	path := writeTestConfig(t, yaml)

	cfg, err := config.LoadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, "123456:ABC-DEF", cfg.TelegramBotToken)
	assert.Equal(t, "/tmp/quizbot.db", cfg.SQLitePath)
	assert.Equal(t, "sk-abc123", cfg.LLMApiKey)
	assert.Equal(t, ":8080", cfg.ListenAddress)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", cfg.EncryptionKey)
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := config.LoadConfig("/nonexistent/path/config.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "opening")
}

func TestLoadConfig_MissingTelegramToken(t *testing.T) {
	yaml := `
sqlite_path: "/tmp/quizbot.db"
encryption_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
`
	path := writeTestConfig(t, yaml)

	_, err := config.LoadConfig(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "telegram_bot_token")
}

func TestLoadConfig_MissingSQLitePath(t *testing.T) {
	yaml := `
telegram_bot_token: "123456:ABC-DEF"
encryption_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
`
	path := writeTestConfig(t, yaml)

	_, err := config.LoadConfig(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite_path")
}

func TestLoadConfig_MissingEncryptionKey(t *testing.T) {
	yaml := `
telegram_bot_token: "123456:ABC-DEF"
sqlite_path: "/tmp/quizbot.db"
`
	path := writeTestConfig(t, yaml)

	_, err := config.LoadConfig(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "encryption_key")
}

func TestLoadConfig_DefaultLogLevel(t *testing.T) {
	yaml := `
telegram_bot_token: "123456:ABC-DEF"
sqlite_path: "/tmp/quizbot.db"
encryption_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
`
	path := writeTestConfig(t, yaml)

	cfg, err := config.LoadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, "info", cfg.LogLevel)
}

func TestLoadConfig_DefaultListenAddress(t *testing.T) {
	yaml := `
telegram_bot_token: "123456:ABC-DEF"
sqlite_path: "/tmp/quizbot.db"
encryption_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
`
	path := writeTestConfig(t, yaml)

	cfg, err := config.LoadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, ":8080", cfg.ListenAddress)
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	yaml := `
this is not: valid: yaml: [
`
	path := writeTestConfig(t, yaml)

	_, err := config.LoadConfig(path)
	assert.Error(t, err)
}

func TestLoadConfig_OptionalLLMApiKey(t *testing.T) {
	yaml := `
telegram_bot_token: "123456:ABC-DEF"
sqlite_path: "/tmp/quizbot.db"
encryption_key: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
`
	path := writeTestConfig(t, yaml)

	cfg, err := config.LoadConfig(path)
	require.NoError(t, err)

	assert.Equal(t, "", cfg.LLMApiKey) // optional, empty is OK
}
```

- [ ] **9.2 — Verify tests fail (TDD red phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./config/... 2>&1 | head -10
```

Expected: compilation errors because `config.Config` and `config.LoadConfig` do not exist.

- [ ] **9.3 — Create `config/config.go`**

```go
// config/config.go
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds infrastructure configuration loaded from a YAML file.
// This is distinct from user preferences which are stored in the
// database and modifiable at runtime.
type Config struct {
	TelegramBotToken string `yaml:"telegram_bot_token"`
	SQLitePath       string `yaml:"sqlite_path"`
	LLMApiKey        string `yaml:"llm_api_key"`
	ListenAddress    string `yaml:"listen_address"`
	LogLevel         string `yaml:"log_level"`
	EncryptionKey    string `yaml:"encryption_key"`
}

// LoadConfig reads and validates a YAML configuration file from the
// given path. Returns an error if the file cannot be read, parsed, or
// is missing required fields.
func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening config file: %w", err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config YAML: %w", err)
	}

	// Apply defaults.
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = ":8080"
	}

	// Validate required fields.
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate checks that all required configuration fields are present.
func (c *Config) validate() error {
	var missing []string

	if c.TelegramBotToken == "" {
		missing = append(missing, "telegram_bot_token")
	}
	if c.SQLitePath == "" {
		missing = append(missing, "sqlite_path")
	}
	if c.EncryptionKey == "" {
		missing = append(missing, "encryption_key")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required config fields: %s", strings.Join(missing, ", "))
	}

	return nil
}
```

- [ ] **9.4 — Run tests (TDD green phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test -v ./config/...
```

Expected: all tests pass.

- [ ] **9.5 — Create `config.example.yaml` at project root**

```yaml
# config.example.yaml
# QuizBot infrastructure configuration.
# Copy this file to config.yaml and fill in your values.
# config.yaml is gitignored — never commit secrets.

# Telegram Bot API token (from @BotFather).
telegram_bot_token: ""

# Path to the SQLite database file.
sqlite_path: "./quizbot.db"

# LLM API key for AI-powered explanations (optional, v2 feature).
# Leave empty to use static explanations only.
llm_api_key: ""

# Address for HTTP endpoints (default: ":8080").
listen_address: ":8080"

# Logging verbosity: debug, info, warn, error (default: "info").
log_level: "info"

# Hex-encoded 32-byte key for encrypting secrets in the database.
# Generate with: openssl rand -hex 32
encryption_key: ""
```

- [ ] **9.6 — Run full test suite**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./...
```

Expected: all tests pass across all packages.

- [ ] **9.7 — Commit: "Add config loading from YAML with validation"**

```bash
cd /Users/samliem/LIFE_IN_UK
git add config/ config.example.yaml
git commit -m "Add config loading from YAML with validation

Implement LoadConfig that reads a YAML config file, applies defaults
for log_level and listen_address, and validates required fields
(telegram_bot_token, sqlite_path, encryption_key). Create
config.example.yaml template with documentation."
```

---

## Task 10: Notifier

**Goal:** Implement the notification system that checks user activity and stats against configurable thresholds, respecting user preferences, frequency caps, and quiet hours.

### Steps

- [ ] **10.1 — Create `internal/model/notification.go` — Notification types**

```go
// internal/model/notification.go
package model

import "time"

// NotificationType classifies the kind of notification.
type NotificationType string

const (
	NotificationInactivityReminder NotificationType = "inactivity_reminder"
	NotificationWeakTopicAlert     NotificationType = "weak_topic_alert"
	NotificationMilestoneCelebration NotificationType = "milestone_celebration"
	NotificationReadinessUpdate    NotificationType = "readiness_update"
	NotificationStreakWarning      NotificationType = "streak_warning"
)

// Notification is a message to be delivered to a user via the Messenger.
type Notification struct {
	Type    NotificationType `json:"type"`
	UserID  string           `json:"user_id"`
	Title   string           `json:"title"`
	Message string           `json:"message"`
	// TopicID is set for WeakTopicAlert notifications.
	TopicID string `json:"topic_id,omitempty"`
	// CreatedAt is when the notification was generated.
	CreatedAt time.Time `json:"created_at"`
}
```

- [ ] **10.2 — Verify model compiles**

```bash
cd /Users/samliem/LIFE_IN_UK
go build ./internal/model/...
```

Expected: clean build.

- [ ] **10.3 — Write failing tests FIRST: `internal/core/notifier_test.go`**

```go
// internal/core/notifier_test.go
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

func setupNotifier(t *testing.T) (*core.Notifier, *store.MockRepository) {
	t.Helper()
	repo := store.NewMockRepository()
	notifier := core.NewNotifier(repo)
	return notifier, repo
}

func seedPrefsAndPack(t *testing.T, repo *store.MockRepository) {
	t.Helper()
	ctx := context.Background()

	pack := model.QuizPack{
		ID:   "test-pack",
		Name: "Test Pack",
		TestFormat: model.TestFormat{
			QuestionCount: 24,
			PassMarkPct:   75.0,
		},
		Topics: []model.Topic{
			{ID: "t1", Name: "Topic 1"},
			{ID: "t2", Name: "Topic 2"},
		},
		Questions: []model.Question{
			{ID: "q1", TopicID: "t1"},
			{ID: "q2", TopicID: "t2"},
		},
	}
	require.NoError(t, repo.SaveQuizPack(ctx, pack))

	prefs := model.DefaultPreferences("user1")
	prefs.ActivePackIDs = []string{"test-pack"}
	require.NoError(t, repo.UpdatePreferences(ctx, prefs))
}

func TestInactivityReminder_Triggers(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	// Create a study session from 3 days ago (beyond default 2-day threshold).
	threeDaysAgo := time.Now().UTC().Add(-72 * time.Hour)
	session := model.StudySession{
		ID:        "s1",
		UserID:    "user1",
		PackID:    "test-pack",
		Mode:      model.SessionModePractice,
		StartedAt: threeDaysAgo,
		Attempts:  []model.QuestionAttempt{},
	}
	require.NoError(t, repo.CreateSession(ctx, session))

	notifications, err := notifier.CheckNotifications(ctx, "user1", threeDaysAgo)
	require.NoError(t, err)

	found := false
	for _, n := range notifications {
		if n.Type == model.NotificationInactivityReminder {
			found = true
			assert.Contains(t, n.Message, "studied")
			break
		}
	}
	assert.True(t, found, "expected InactivityReminder notification")
}

func TestInactivityReminder_NoTriggerWhenRecent(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	// Create a study session from 1 hour ago (within threshold).
	recentTime := time.Now().UTC().Add(-1 * time.Hour)
	session := model.StudySession{
		ID:        "s1",
		UserID:    "user1",
		PackID:    "test-pack",
		Mode:      model.SessionModePractice,
		StartedAt: recentTime,
		Attempts:  []model.QuestionAttempt{},
	}
	require.NoError(t, repo.CreateSession(ctx, session))

	notifications, err := notifier.CheckNotifications(ctx, "user1", recentTime)
	require.NoError(t, err)

	for _, n := range notifications {
		assert.NotEqual(t, model.NotificationInactivityReminder, n.Type)
	}
}

func TestInactivityReminder_DisabledByPref(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	prefs := model.DefaultPreferences("user1")
	prefs.ActivePackIDs = []string{"test-pack"}
	prefs.NotifyInactivity = false
	require.NoError(t, repo.UpdatePreferences(ctx, prefs))

	threeDaysAgo := time.Now().UTC().Add(-72 * time.Hour)
	session := model.StudySession{
		ID: "s1", UserID: "user1", PackID: "test-pack",
		Mode: model.SessionModePractice, StartedAt: threeDaysAgo,
		Attempts: []model.QuestionAttempt{},
	}
	require.NoError(t, repo.CreateSession(ctx, session))

	notifications, err := notifier.CheckNotifications(ctx, "user1", threeDaysAgo)
	require.NoError(t, err)

	for _, n := range notifications {
		assert.NotEqual(t, model.NotificationInactivityReminder, n.Type)
	}
}

func TestWeakTopicAlert_Triggers(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	// Set topic t1 accuracy below the default 50% threshold.
	stats := model.TopicStats{
		UserID: "user1", PackID: "test-pack", TopicID: "t1",
		TotalAttempts: 10, CorrectCount: 3, RollingAccuracy: 0.3,
	}
	require.NoError(t, repo.UpdateTopicStats(ctx, stats))

	now := time.Now().UTC()
	notifications, err := notifier.CheckNotifications(ctx, "user1", now)
	require.NoError(t, err)

	found := false
	for _, n := range notifications {
		if n.Type == model.NotificationWeakTopicAlert && n.TopicID == "t1" {
			found = true
			assert.Contains(t, n.Message, "Topic 1")
			break
		}
	}
	assert.True(t, found, "expected WeakTopicAlert notification for t1")
}

func TestWeakTopicAlert_NoTriggerAboveThreshold(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	stats := model.TopicStats{
		UserID: "user1", PackID: "test-pack", TopicID: "t1",
		TotalAttempts: 10, CorrectCount: 8, RollingAccuracy: 0.8,
	}
	require.NoError(t, repo.UpdateTopicStats(ctx, stats))

	now := time.Now().UTC()
	notifications, err := notifier.CheckNotifications(ctx, "user1", now)
	require.NoError(t, err)

	for _, n := range notifications {
		if n.Type == model.NotificationWeakTopicAlert {
			assert.NotEqual(t, "t1", n.TopicID)
		}
	}
}

func TestWeakTopicAlert_DisabledByPref(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	prefs := model.DefaultPreferences("user1")
	prefs.ActivePackIDs = []string{"test-pack"}
	prefs.NotifyWeakTopic = false
	require.NoError(t, repo.UpdatePreferences(ctx, prefs))

	stats := model.TopicStats{
		UserID: "user1", PackID: "test-pack", TopicID: "t1",
		TotalAttempts: 10, CorrectCount: 2, RollingAccuracy: 0.2,
	}
	require.NoError(t, repo.UpdateTopicStats(ctx, stats))

	now := time.Now().UTC()
	notifications, err := notifier.CheckNotifications(ctx, "user1", now)
	require.NoError(t, err)

	for _, n := range notifications {
		assert.NotEqual(t, model.NotificationWeakTopicAlert, n.Type)
	}
}

func TestMilestoneCelebration_TotalAttempts(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	// Set topic stats to exactly 100 total attempts (milestone).
	stats := model.TopicStats{
		UserID: "user1", PackID: "test-pack", TopicID: "t1",
		TotalAttempts: 100, CorrectCount: 80, RollingAccuracy: 0.8,
	}
	require.NoError(t, repo.UpdateTopicStats(ctx, stats))

	now := time.Now().UTC()
	notifications, err := notifier.CheckNotifications(ctx, "user1", now)
	require.NoError(t, err)

	found := false
	for _, n := range notifications {
		if n.Type == model.NotificationMilestoneCelebration {
			found = true
			assert.Contains(t, n.Message, "100")
			break
		}
	}
	assert.True(t, found, "expected MilestoneCelebration for 100 attempts")
}

func TestMilestoneCelebration_DisabledByPref(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	prefs := model.DefaultPreferences("user1")
	prefs.ActivePackIDs = []string{"test-pack"}
	prefs.NotifyMilestones = false
	require.NoError(t, repo.UpdatePreferences(ctx, prefs))

	stats := model.TopicStats{
		UserID: "user1", PackID: "test-pack", TopicID: "t1",
		TotalAttempts: 100, CorrectCount: 80, RollingAccuracy: 0.8,
	}
	require.NoError(t, repo.UpdateTopicStats(ctx, stats))

	now := time.Now().UTC()
	notifications, err := notifier.CheckNotifications(ctx, "user1", now)
	require.NoError(t, err)

	for _, n := range notifications {
		assert.NotEqual(t, model.NotificationMilestoneCelebration, n.Type)
	}
}

func TestStreakWarning_Triggers(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	// Create a session from yesterday to establish a streak, but none today.
	yesterday := time.Now().UTC().Add(-30 * time.Hour)
	session := model.StudySession{
		ID: "s1", UserID: "user1", PackID: "test-pack",
		Mode: model.SessionModePractice, StartedAt: yesterday,
		Attempts: []model.QuestionAttempt{},
	}
	require.NoError(t, repo.CreateSession(ctx, session))

	// Set topic stats with an active streak.
	stats := model.TopicStats{
		UserID: "user1", PackID: "test-pack", TopicID: "t1",
		TotalAttempts: 30, CorrectCount: 25, RollingAccuracy: 0.83,
		CurrentStreak: 5, BestStreak: 10,
	}
	require.NoError(t, repo.UpdateTopicStats(ctx, stats))

	now := time.Now().UTC()
	notifications, err := notifier.CheckNotifications(ctx, "user1", yesterday)
	require.NoError(t, err)

	found := false
	for _, n := range notifications {
		if n.Type == model.NotificationStreakWarning {
			found = true
			assert.Contains(t, n.Message, "streak")
			break
		}
	}
	assert.True(t, found, "expected StreakWarning notification")
}

func TestStreakWarning_DisabledByPref(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	prefs := model.DefaultPreferences("user1")
	prefs.ActivePackIDs = []string{"test-pack"}
	prefs.NotifyStreak = false
	require.NoError(t, repo.UpdatePreferences(ctx, prefs))

	yesterday := time.Now().UTC().Add(-30 * time.Hour)
	session := model.StudySession{
		ID: "s1", UserID: "user1", PackID: "test-pack",
		Mode: model.SessionModePractice, StartedAt: yesterday,
		Attempts: []model.QuestionAttempt{},
	}
	require.NoError(t, repo.CreateSession(ctx, session))

	stats := model.TopicStats{
		UserID: "user1", PackID: "test-pack", TopicID: "t1",
		TotalAttempts: 30, CorrectCount: 25, CurrentStreak: 5,
	}
	require.NoError(t, repo.UpdateTopicStats(ctx, stats))

	notifications, err := notifier.CheckNotifications(ctx, "user1", yesterday)
	require.NoError(t, err)

	for _, n := range notifications {
		assert.NotEqual(t, model.NotificationStreakWarning, n.Type)
	}
}

func TestQuietHours_SuppressesNotifications(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	// Set quiet hours to 22:00-08:00.
	prefs := model.DefaultPreferences("user1")
	prefs.ActivePackIDs = []string{"test-pack"}
	prefs.QuietHoursStart = "22:00"
	prefs.QuietHoursEnd = "08:00"
	require.NoError(t, repo.UpdatePreferences(ctx, prefs))

	// Simulate conditions that would trigger inactivity.
	threeDaysAgo := time.Now().UTC().Add(-72 * time.Hour)
	session := model.StudySession{
		ID: "s1", UserID: "user1", PackID: "test-pack",
		Mode: model.SessionModePractice, StartedAt: threeDaysAgo,
		Attempts: []model.QuestionAttempt{},
	}
	require.NoError(t, repo.CreateSession(ctx, session))

	// Create a "now" time at 23:00 (during quiet hours).
	now := time.Date(2026, 4, 3, 23, 0, 0, 0, time.UTC)
	notifications, err := notifier.CheckNotifications(ctx, "user1", threeDaysAgo, core.WithCurrentTime(now))
	require.NoError(t, err)

	assert.Empty(t, notifications, "no notifications should be sent during quiet hours")
}

func TestQuietHours_AllowsOutsideQuietHours(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	prefs := model.DefaultPreferences("user1")
	prefs.ActivePackIDs = []string{"test-pack"}
	prefs.QuietHoursStart = "22:00"
	prefs.QuietHoursEnd = "08:00"
	require.NoError(t, repo.UpdatePreferences(ctx, prefs))

	threeDaysAgo := time.Now().UTC().Add(-72 * time.Hour)
	session := model.StudySession{
		ID: "s1", UserID: "user1", PackID: "test-pack",
		Mode: model.SessionModePractice, StartedAt: threeDaysAgo,
		Attempts: []model.QuestionAttempt{},
	}
	require.NoError(t, repo.CreateSession(ctx, session))

	// Create a "now" time at 10:00 (outside quiet hours).
	now := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)
	notifications, err := notifier.CheckNotifications(ctx, "user1", threeDaysAgo, core.WithCurrentTime(now))
	require.NoError(t, err)

	assert.NotEmpty(t, notifications, "notifications should be sent outside quiet hours")
}

func TestReadinessUpdate_Triggers(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	stats := model.TopicStats{
		UserID: "user1", PackID: "test-pack", TopicID: "t1",
		TotalAttempts: 50, CorrectCount: 40, RollingAccuracy: 0.8,
	}
	require.NoError(t, repo.UpdateTopicStats(ctx, stats))

	// Use a time on Monday (weekly trigger day).
	monday := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC) // Monday
	notifications, err := notifier.CheckNotifications(ctx, "user1", time.Now().UTC(), core.WithCurrentTime(monday))
	require.NoError(t, err)

	found := false
	for _, n := range notifications {
		if n.Type == model.NotificationReadinessUpdate {
			found = true
			assert.Contains(t, n.Message, "readiness")
			break
		}
	}
	assert.True(t, found, "expected ReadinessUpdate notification on Monday")
}

func TestReadinessUpdate_DisabledByPref(t *testing.T) {
	notifier, repo := setupNotifier(t)
	ctx := context.Background()

	seedPrefsAndPack(t, repo)

	prefs := model.DefaultPreferences("user1")
	prefs.ActivePackIDs = []string{"test-pack"}
	prefs.NotifyReadiness = false
	require.NoError(t, repo.UpdatePreferences(ctx, prefs))

	monday := time.Date(2026, 4, 6, 10, 0, 0, 0, time.UTC)
	notifications, err := notifier.CheckNotifications(ctx, "user1", time.Now().UTC(), core.WithCurrentTime(monday))
	require.NoError(t, err)

	for _, n := range notifications {
		assert.NotEqual(t, model.NotificationReadinessUpdate, n.Type)
	}
}
```

- [ ] **10.4 — Verify tests fail (TDD red phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./internal/core/... 2>&1 | head -20
```

Expected: compilation errors because `Notifier`, `NewNotifier`, `WithCurrentTime`, and related types do not exist.

- [ ] **10.5 — Implement `internal/core/notifier.go`**

```go
// internal/core/notifier.go
package core

import (
	"context"
	"fmt"
	"time"

	"github.com/sam-liem/quizbot/internal/model"
	"github.com/sam-liem/quizbot/internal/store"
)

// Notifier checks various conditions against user preferences and stats
// to determine which notifications should be sent. It does not deliver
// them — the caller (scheduler or CLI) handles delivery via Messenger.
type Notifier struct {
	repo store.Repository
}

// NewNotifier creates a Notifier with injected dependencies.
func NewNotifier(repo store.Repository) *Notifier {
	return &Notifier{repo: repo}
}

// NotifierOption configures a CheckNotifications call.
type NotifierOption func(*notifierConfig)

type notifierConfig struct {
	now time.Time
}

// WithCurrentTime overrides the current time for testing.
func WithCurrentTime(t time.Time) NotifierOption {
	return func(c *notifierConfig) {
		c.now = t
	}
}

// CheckNotifications evaluates all notification conditions for a user
// and returns the notifications that should be delivered. The
// lastActivity parameter is the timestamp of the user's most recent
// study activity.
func (n *Notifier) CheckNotifications(ctx context.Context, userID string, lastActivity time.Time, opts ...NotifierOption) ([]model.Notification, error) {
	cfg := &notifierConfig{
		now: time.Now().UTC(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	prefs, err := n.repo.GetPreferences(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("getting preferences: %w", err)
	}
	if prefs == nil {
		return nil, nil
	}

	// Check quiet hours — if we're in quiet hours, suppress all.
	if isQuietHours(cfg.now, prefs.QuietHoursStart, prefs.QuietHoursEnd) {
		return nil, nil
	}

	var notifications []model.Notification

	// Check each notification type.
	if prefs.NotifyInactivity {
		if notif := n.checkInactivity(userID, lastActivity, prefs, cfg.now); notif != nil {
			notifications = append(notifications, *notif)
		}
	}

	if prefs.NotifyWeakTopic {
		weakNotifs, err := n.checkWeakTopics(ctx, userID, prefs, cfg.now)
		if err != nil {
			return nil, fmt.Errorf("checking weak topics: %w", err)
		}
		notifications = append(notifications, weakNotifs...)
	}

	if prefs.NotifyMilestones {
		milestoneNotifs, err := n.checkMilestones(ctx, userID, prefs, cfg.now)
		if err != nil {
			return nil, fmt.Errorf("checking milestones: %w", err)
		}
		notifications = append(notifications, milestoneNotifs...)
	}

	if prefs.NotifyReadiness {
		readinessNotifs, err := n.checkReadiness(ctx, userID, prefs, cfg.now)
		if err != nil {
			return nil, fmt.Errorf("checking readiness: %w", err)
		}
		notifications = append(notifications, readinessNotifs...)
	}

	if prefs.NotifyStreak {
		if notif := n.checkStreak(ctx, userID, lastActivity, prefs, cfg.now); notif != nil {
			notifications = append(notifications, *notif)
		}
	}

	return notifications, nil
}

// checkInactivity returns an InactivityReminder if the user hasn't
// studied within the configured threshold.
func (n *Notifier) checkInactivity(userID string, lastActivity time.Time, prefs *model.UserPreferences, now time.Time) *model.Notification {
	threshold := time.Duration(prefs.NotifyInactivityDays) * 24 * time.Hour
	if now.Sub(lastActivity) < threshold {
		return nil
	}

	daysSince := int(now.Sub(lastActivity).Hours() / 24)

	return &model.Notification{
		Type:      model.NotificationInactivityReminder,
		UserID:    userID,
		Title:     "Time to study!",
		Message:   fmt.Sprintf("You haven't studied in %d days. A quick session keeps your knowledge fresh!", daysSince),
		CreatedAt: now,
	}
}

// checkWeakTopics returns WeakTopicAlert notifications for topics with
// rolling accuracy below the configured threshold.
func (n *Notifier) checkWeakTopics(ctx context.Context, userID string, prefs *model.UserPreferences, now time.Time) ([]model.Notification, error) {
	var notifications []model.Notification

	for _, packID := range prefs.ActivePackIDs {
		pack, err := n.repo.GetQuizPack(ctx, packID)
		if err != nil {
			return nil, fmt.Errorf("getting quiz pack %q: %w", packID, err)
		}
		if pack == nil {
			continue
		}

		topicStats, err := n.repo.ListTopicStats(ctx, userID, packID)
		if err != nil {
			return nil, fmt.Errorf("listing topic stats: %w", err)
		}

		// Build topic name lookup.
		topicNames := make(map[string]string)
		for _, t := range pack.Topics {
			topicNames[t.ID] = t.Name
		}

		thresholdPct := prefs.NotifyWeakTopicPct / 100.0

		for _, stats := range topicStats {
			if stats.TotalAttempts < 5 {
				continue // not enough data to judge
			}
			if stats.RollingAccuracy < thresholdPct {
				topicName := topicNames[stats.TopicID]
				if topicName == "" {
					topicName = stats.TopicID
				}
				notifications = append(notifications, model.Notification{
					Type:      model.NotificationWeakTopicAlert,
					UserID:    userID,
					Title:     "Weak topic detected",
					Message:   fmt.Sprintf("Your accuracy on \"%s\" is %.0f%%. Focus on this topic to improve!", topicName, stats.RollingAccuracy*100),
					TopicID:   stats.TopicID,
					CreatedAt: now,
				})
			}
		}
	}

	return notifications, nil
}

// milestoneThresholds defines the total attempt counts that trigger
// milestone celebrations.
var milestoneThresholds = []int{50, 100, 250, 500, 1000, 2500, 5000}

// checkMilestones returns MilestoneCelebration notifications for
// round question counts.
func (n *Notifier) checkMilestones(ctx context.Context, userID string, prefs *model.UserPreferences, now time.Time) ([]model.Notification, error) {
	var notifications []model.Notification

	for _, packID := range prefs.ActivePackIDs {
		topicStats, err := n.repo.ListTopicStats(ctx, userID, packID)
		if err != nil {
			return nil, fmt.Errorf("listing topic stats: %w", err)
		}

		// Sum total attempts across all topics in this pack.
		totalAttempts := 0
		for _, stats := range topicStats {
			totalAttempts += stats.TotalAttempts
		}

		for _, threshold := range milestoneThresholds {
			if totalAttempts == threshold {
				notifications = append(notifications, model.Notification{
					Type:      model.NotificationMilestoneCelebration,
					UserID:    userID,
					Title:     "Milestone reached!",
					Message:   fmt.Sprintf("You've answered %d questions! Keep up the great work!", threshold),
					CreatedAt: now,
				})
				break
			}
		}
	}

	return notifications, nil
}

// checkReadiness returns a ReadinessUpdate notification on Mondays
// (weekly summary).
func (n *Notifier) checkReadiness(ctx context.Context, userID string, prefs *model.UserPreferences, now time.Time) ([]model.Notification, error) {
	// Only trigger on Mondays.
	if now.Weekday() != time.Monday {
		return nil, nil
	}

	var notifications []model.Notification

	for _, packID := range prefs.ActivePackIDs {
		pack, err := n.repo.GetQuizPack(ctx, packID)
		if err != nil {
			return nil, fmt.Errorf("getting quiz pack %q: %w", packID, err)
		}
		if pack == nil {
			continue
		}

		topicStats, err := n.repo.ListTopicStats(ctx, userID, packID)
		if err != nil {
			return nil, fmt.Errorf("listing topic stats: %w", err)
		}

		readiness := CalculateReadiness(topicStats, *pack)

		notifications = append(notifications, model.Notification{
			Type:      model.NotificationReadinessUpdate,
			UserID:    userID,
			Title:     "Weekly readiness update",
			Message:   fmt.Sprintf("Your estimated readiness for \"%s\" is %.0f%%. Keep studying to improve!", pack.Name, readiness*100),
			CreatedAt: now,
		})
	}

	return notifications, nil
}

// checkStreak returns a StreakWarning if the user has an active streak
// but hasn't studied today.
func (n *Notifier) checkStreak(ctx context.Context, userID string, lastActivity time.Time, prefs *model.UserPreferences, now time.Time) *model.Notification {
	// Check if last activity was before today.
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if !lastActivity.Before(todayStart) {
		return nil // studied today already
	}

	// Check if any active pack has an active streak.
	hasStreak := false
	for _, packID := range prefs.ActivePackIDs {
		topicStats, err := n.repo.ListTopicStats(ctx, userID, packID)
		if err != nil {
			continue
		}
		for _, stats := range topicStats {
			if stats.CurrentStreak > 0 {
				hasStreak = true
				break
			}
		}
		if hasStreak {
			break
		}
	}

	if !hasStreak {
		return nil
	}

	return &model.Notification{
		Type:      model.NotificationStreakWarning,
		UserID:    userID,
		Title:     "Don't break your streak!",
		Message:   "You haven't studied today and your streak is at risk. A quick session will keep it going!",
		CreatedAt: now,
	}
}

// isQuietHours checks if the given time falls within the user's quiet
// hours window. Handles the overnight case (e.g., 22:00 to 08:00).
func isQuietHours(now time.Time, startStr, endStr string) bool {
	if startStr == "" || endStr == "" {
		return false
	}

	start, err := time.Parse("15:04", startStr)
	if err != nil {
		return false
	}
	end, err := time.Parse("15:04", endStr)
	if err != nil {
		return false
	}

	currentMinutes := now.Hour()*60 + now.Minute()
	startMinutes := start.Hour()*60 + start.Minute()
	endMinutes := end.Hour()*60 + end.Minute()

	if startMinutes <= endMinutes {
		// Same-day window (e.g., 08:00 to 17:00).
		return currentMinutes >= startMinutes && currentMinutes < endMinutes
	}

	// Overnight window (e.g., 22:00 to 08:00).
	return currentMinutes >= startMinutes || currentMinutes < endMinutes
}
```

- [ ] **10.6 — Run tests (TDD green phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test -v ./internal/core/...
```

Expected: all tests pass (notifier, engine, repetition, and stats tests).

- [ ] **10.7 — Run full test suite**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./...
```

Expected: all tests pass across all packages.

- [ ] **10.8 — Commit: "Add notification system with all notification types"**

```bash
cd /Users/samliem/LIFE_IN_UK
git add internal/model/notification.go internal/core/notifier.go internal/core/notifier_test.go
git commit -m "Add notification system with all notification types

Implement Notifier with CheckNotifications that evaluates five
notification types: InactivityReminder, WeakTopicAlert,
MilestoneCelebration, ReadinessUpdate, and StreakWarning. Each type
respects user preference toggles. Quiet hours suppression and weekly
readiness schedule implemented. Tests cover all trigger conditions,
preference overrides, and quiet hours."
```
