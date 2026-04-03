# QuizBot Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an adaptive quiz preparation tool with spaced repetition, Telegram delivery, and CLI — targeting the Life in the UK test as first quiz pack.

**Architecture:** Single Go binary with clean interface boundaries. All subsystems (storage, messaging, explanation) behind interfaces for testability and future SaaS expansion. Repository layer scoped by user ID. CLI and Telegram bot are thin adapters over a shared core engine.

**Tech Stack:** Go 1.22+, modernc.org/sqlite, cobra, telegram-bot-api/v5, yaml.v3, slog, testify

**Principles:** TDD (red-green-refactor), frequent commits per module/feature, no placeholders.

---

## Task 1: Project Bootstrap

**Goal:** Initialize Go module and create all domain types from the spec.

### Steps

- [ ] **1.1 — Initialize Go module**

```bash
cd /Users/samliem/LIFE_IN_UK
go mod init github.com/sam-liem/quizbot
```

Expected output: `go.mod` file created with module path `github.com/sam-liem/quizbot`.

- [ ] **1.2 — Create `internal/model/enums.go`**

All enum types used across the domain model.

```go
// internal/model/enums.go
package model

// SessionMode represents how a quiz session was initiated.
type SessionMode string

const (
	SessionModeMock      SessionMode = "mock"
	SessionModePractice  SessionMode = "practice"
	SessionModeScheduled SessionMode = "scheduled"
)

// QuizSessionStatus represents the lifecycle state of a quiz session.
type QuizSessionStatus string

const (
	QuizSessionStatusInProgress QuizSessionStatus = "in_progress"
	QuizSessionStatusCompleted  QuizSessionStatus = "completed"
	QuizSessionStatusAbandoned  QuizSessionStatus = "abandoned"
)

// AnswerResult represents the outcome of answering a question.
type AnswerResult string

const (
	AnswerResultCorrect AnswerResult = "correct"
	AnswerResultWrong   AnswerResult = "wrong"
	AnswerResultSkipped AnswerResult = "skipped"
)

// FocusMode represents how multiple active packs are handled.
type FocusMode string

const (
	FocusModeSingle      FocusMode = "single"
	FocusModeInterleaved FocusMode = "interleaved"
)
```

- [ ] **1.3 — Create `internal/model/types.go`**

Core content types: QuizPack, Question, Topic, TestFormat.

```go
// internal/model/types.go
package model

// TestFormat defines the parameters for a mock test.
type TestFormat struct {
	QuestionCount int     `json:"question_count" yaml:"question_count"`
	PassMarkPct   float64 `json:"pass_mark_pct" yaml:"pass_mark_pct"`
	TimeLimitSec  int     `json:"time_limit_sec" yaml:"time_limit_sec"`
}

// Topic represents a category of questions within a quiz pack.
type Topic struct {
	ID          string `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Question represents a single multiple-choice question.
type Question struct {
	ID           string   `json:"id" yaml:"id"`
	TopicID      string   `json:"topic_id" yaml:"topic_id"`
	Text         string   `json:"text" yaml:"text"`
	Choices      []string `json:"choices" yaml:"choices"`
	CorrectIndex int      `json:"correct_index" yaml:"correct_index"`
	Explanation  string   `json:"explanation,omitempty" yaml:"explanation,omitempty"`
}

// QuizPack is the self-contained unit of quiz content.
type QuizPack struct {
	ID          string     `json:"id" yaml:"id"`
	Name        string     `json:"name" yaml:"name"`
	Description string     `json:"description" yaml:"description"`
	Version     string     `json:"version" yaml:"version"`
	TestFormat  TestFormat `json:"test_format" yaml:"test_format"`
	Topics      []Topic    `json:"topics" yaml:"topics"`
	Questions   []Question `json:"questions" yaml:"questions"`
}
```

- [ ] **1.4 — Create `internal/model/state.go`**

Spaced repetition state per question per user.

```go
// internal/model/state.go
package model

import "time"

// QuestionState holds SM-2 spaced repetition metadata for a single
// question, scoped to one user and one quiz pack.
type QuestionState struct {
	UserID          string       `json:"user_id"`
	QuestionID      string       `json:"question_id"`
	PackID          string       `json:"pack_id"`
	EaseFactor      float64      `json:"ease_factor"`
	IntervalDays    float64      `json:"interval_days"`
	RepetitionCount int          `json:"repetition_count"`
	NextReviewAt    time.Time    `json:"next_review_at"`
	LastResult      AnswerResult `json:"last_result"`
	LastReviewedAt  time.Time    `json:"last_reviewed_at"`
}

// TopicStats holds aggregate statistics for a single topic within a
// quiz pack, scoped to one user.
type TopicStats struct {
	UserID          string  `json:"user_id"`
	PackID          string  `json:"pack_id"`
	TopicID         string  `json:"topic_id"`
	TotalAttempts   int     `json:"total_attempts"`
	CorrectCount    int     `json:"correct_count"`
	RollingAccuracy float64 `json:"rolling_accuracy"`
	CurrentStreak   int     `json:"current_streak"`
	BestStreak      int     `json:"best_streak"`
}
```

- [ ] **1.5 — Create `internal/model/session.go`**

Study session and quiz session types.

```go
// internal/model/session.go
package model

import "time"

// QuestionAttempt records a single answer to a single question within
// a study session.
type QuestionAttempt struct {
	QuestionID  string    `json:"question_id"`
	AnswerIndex int       `json:"answer_index"`
	Correct     bool      `json:"correct"`
	TimeTakenMs int       `json:"time_taken_ms"`
	AnsweredAt  time.Time `json:"answered_at"`
}

// StudySession records a complete study sitting — a sequence of
// question attempts in a single mode.
type StudySession struct {
	ID            string            `json:"id"`
	UserID        string            `json:"user_id"`
	PackID        string            `json:"pack_id"`
	Mode          SessionMode       `json:"mode"`
	StartedAt     time.Time         `json:"started_at"`
	EndedAt       *time.Time        `json:"ended_at,omitempty"`
	QuestionCount int               `json:"question_count"`
	CorrectCount  int               `json:"correct_count"`
	Attempts      []QuestionAttempt `json:"attempts"`
}

// QuizSession tracks an in-progress quiz for resume support.
type QuizSession struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id"`
	PackID       string            `json:"pack_id"`
	Mode         SessionMode       `json:"mode"`
	QuestionIDs  []string          `json:"question_ids"`
	CurrentIndex int               `json:"current_index"`
	Answers      map[string]int    `json:"answers"`
	StartedAt    time.Time         `json:"started_at"`
	TimeLimitSec int               `json:"time_limit_sec"`
	Status       QuizSessionStatus `json:"status"`
}
```

- [ ] **1.6 — Create `internal/model/preferences.go`**

User preferences stored in DB.

```go
// internal/model/preferences.go
package model

// UserPreferences holds per-user configuration that is stored in the
// database and modifiable at runtime via CLI or Telegram.
type UserPreferences struct {
	UserID               string   `json:"user_id"`
	DeliveryIntervalMin  int      `json:"delivery_interval_min"`
	MaxUnanswered        int      `json:"max_unanswered"`
	ActivePackIDs        []string `json:"active_pack_ids"`
	FocusMode            FocusMode `json:"focus_mode"`
	NotifyInactivity     bool     `json:"notify_inactivity"`
	NotifyInactivityDays int      `json:"notify_inactivity_days"`
	NotifyWeakTopic      bool     `json:"notify_weak_topic"`
	NotifyWeakTopicPct   float64  `json:"notify_weak_topic_pct"`
	NotifyMilestones     bool     `json:"notify_milestones"`
	NotifyReadiness      bool     `json:"notify_readiness"`
	NotifyStreak         bool     `json:"notify_streak"`
	QuietHoursStart      string   `json:"quiet_hours_start"`
	QuietHoursEnd        string   `json:"quiet_hours_end"`
}

// DefaultPreferences returns a UserPreferences with the spec defaults applied.
func DefaultPreferences(userID string) UserPreferences {
	return UserPreferences{
		UserID:               userID,
		DeliveryIntervalMin:  60,
		MaxUnanswered:        3,
		ActivePackIDs:        []string{},
		FocusMode:            FocusModeSingle,
		NotifyInactivity:     true,
		NotifyInactivityDays: 2,
		NotifyWeakTopic:      true,
		NotifyWeakTopicPct:   50.0,
		NotifyMilestones:     true,
		NotifyReadiness:      true,
		NotifyStreak:         true,
		QuietHoursStart:      "22:00",
		QuietHoursEnd:        "08:00",
	}
}
```

- [ ] **1.7 — Verify the model package compiles**

```bash
cd /Users/samliem/LIFE_IN_UK
go build ./internal/model/...
```

Expected: no errors, clean build.

- [ ] **1.8 — Commit: "Add domain model types"**

```bash
cd /Users/samliem/LIFE_IN_UK
git add go.mod internal/model/
git commit -m "Add domain model types

Define all core types from the spec: QuizPack, Question, Topic,
TestFormat, QuestionState, TopicStats, StudySession, QuizSession,
QuestionAttempt, UserPreferences, and all enums."
```

---

## Task 2: Repository Interface + SQLite Implementation

**Goal:** Define the Repository interface and implement it with SQLite, validated by integration tests against in-memory SQLite.

### Steps

- [ ] **2.1 — Create `internal/store/repository.go` — Repository interface**

```go
// internal/store/repository.go
package store

import (
	"context"

	"github.com/sam-liem/quizbot/internal/model"
)

// Repository defines the data access layer for QuizBot. Every method
// is scoped by user ID for SaaS readiness, even in v1 single-user mode.
type Repository interface {
	// Quiz Packs
	SaveQuizPack(ctx context.Context, pack model.QuizPack) error
	GetQuizPack(ctx context.Context, packID string) (*model.QuizPack, error)
	ListQuizPacks(ctx context.Context) ([]model.QuizPack, error)

	// Spaced Repetition State
	GetQuestionState(ctx context.Context, userID, packID, questionID string) (*model.QuestionState, error)
	UpdateQuestionState(ctx context.Context, state model.QuestionState) error

	// Topic Statistics
	GetTopicStats(ctx context.Context, userID, packID, topicID string) (*model.TopicStats, error)
	UpdateTopicStats(ctx context.Context, stats model.TopicStats) error
	ListTopicStats(ctx context.Context, userID, packID string) ([]model.TopicStats, error)

	// Study Sessions
	CreateSession(ctx context.Context, session model.StudySession) error
	GetSession(ctx context.Context, userID, sessionID string) (*model.StudySession, error)
	UpdateSession(ctx context.Context, session model.StudySession) error

	// Quiz Sessions (resumable)
	SaveQuizSession(ctx context.Context, session model.QuizSession) error
	GetQuizSession(ctx context.Context, userID, sessionID string) (*model.QuizSession, error)

	// User Preferences
	GetPreferences(ctx context.Context, userID string) (*model.UserPreferences, error)
	UpdatePreferences(ctx context.Context, prefs model.UserPreferences) error
}
```

- [ ] **2.2 — Verify repository interface compiles**

```bash
cd /Users/samliem/LIFE_IN_UK
go build ./internal/store/...
```

Expected: clean build.

- [ ] **2.3 — Install SQLite dependency**

```bash
cd /Users/samliem/LIFE_IN_UK
go get modernc.org/sqlite
```

- [ ] **2.4 — Create `internal/store/sqlite/migrations/001_initial.sql`**

Complete schema covering all tables.

```sql
-- internal/store/sqlite/migrations/001_initial.sql

CREATE TABLE IF NOT EXISTS quiz_packs (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version     TEXT NOT NULL DEFAULT '1.0.0',
    data        TEXT NOT NULL  -- JSON blob of the full QuizPack
);

CREATE TABLE IF NOT EXISTS question_states (
    user_id          TEXT    NOT NULL,
    pack_id          TEXT    NOT NULL,
    question_id      TEXT    NOT NULL,
    ease_factor      REAL    NOT NULL DEFAULT 2.5,
    interval_days    REAL    NOT NULL DEFAULT 0,
    repetition_count INTEGER NOT NULL DEFAULT 0,
    next_review_at   TEXT    NOT NULL,
    last_result      TEXT    NOT NULL DEFAULT '',
    last_reviewed_at TEXT    NOT NULL DEFAULT '',
    PRIMARY KEY (user_id, pack_id, question_id)
);

CREATE INDEX IF NOT EXISTS idx_question_states_review
    ON question_states (user_id, pack_id, next_review_at);

CREATE TABLE IF NOT EXISTS topic_stats (
    user_id          TEXT    NOT NULL,
    pack_id          TEXT    NOT NULL,
    topic_id         TEXT    NOT NULL,
    total_attempts   INTEGER NOT NULL DEFAULT 0,
    correct_count    INTEGER NOT NULL DEFAULT 0,
    rolling_accuracy REAL    NOT NULL DEFAULT 0,
    current_streak   INTEGER NOT NULL DEFAULT 0,
    best_streak      INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, pack_id, topic_id)
);

CREATE TABLE IF NOT EXISTS study_sessions (
    id             TEXT PRIMARY KEY,
    user_id        TEXT    NOT NULL,
    pack_id        TEXT    NOT NULL,
    mode           TEXT    NOT NULL,
    started_at     TEXT    NOT NULL,
    ended_at       TEXT,
    question_count INTEGER NOT NULL DEFAULT 0,
    correct_count  INTEGER NOT NULL DEFAULT 0,
    attempts       TEXT    NOT NULL DEFAULT '[]'  -- JSON array of QuestionAttempt
);

CREATE INDEX IF NOT EXISTS idx_study_sessions_user
    ON study_sessions (user_id, started_at DESC);

CREATE TABLE IF NOT EXISTS quiz_sessions (
    id             TEXT PRIMARY KEY,
    user_id        TEXT    NOT NULL,
    pack_id        TEXT    NOT NULL,
    mode           TEXT    NOT NULL,
    question_ids   TEXT    NOT NULL DEFAULT '[]',  -- JSON array of strings
    current_index  INTEGER NOT NULL DEFAULT 0,
    answers        TEXT    NOT NULL DEFAULT '{}',   -- JSON object string->int
    started_at     TEXT    NOT NULL,
    time_limit_sec INTEGER NOT NULL DEFAULT 0,
    status         TEXT    NOT NULL DEFAULT 'in_progress'
);

CREATE INDEX IF NOT EXISTS idx_quiz_sessions_user
    ON quiz_sessions (user_id, status);

CREATE TABLE IF NOT EXISTS user_preferences (
    user_id                TEXT PRIMARY KEY,
    delivery_interval_min  INTEGER NOT NULL DEFAULT 60,
    max_unanswered         INTEGER NOT NULL DEFAULT 3,
    active_pack_ids        TEXT    NOT NULL DEFAULT '[]',  -- JSON array of strings
    focus_mode             TEXT    NOT NULL DEFAULT 'single',
    notify_inactivity      INTEGER NOT NULL DEFAULT 1,
    notify_inactivity_days INTEGER NOT NULL DEFAULT 2,
    notify_weak_topic      INTEGER NOT NULL DEFAULT 1,
    notify_weak_topic_pct  REAL    NOT NULL DEFAULT 50.0,
    notify_milestones      INTEGER NOT NULL DEFAULT 1,
    notify_readiness       INTEGER NOT NULL DEFAULT 1,
    notify_streak          INTEGER NOT NULL DEFAULT 1,
    quiet_hours_start      TEXT    NOT NULL DEFAULT '22:00',
    quiet_hours_end        TEXT    NOT NULL DEFAULT '08:00'
);
```

- [ ] **2.5 — Write integration tests FIRST: `internal/store/sqlite/sqlite_test.go`**

TDD: write all tests before the implementation. Tests use in-memory SQLite.

```go
// internal/store/sqlite/sqlite_test.go
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

func newTestDB(t *testing.T) *sqlite.DB {
	t.Helper()
	db, err := sqlite.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}

func samplePack() model.QuizPack {
	return model.QuizPack{
		ID:          "life-in-uk",
		Name:        "Life in the UK",
		Description: "Official Life in the UK Test preparation",
		Version:     "1.0.0",
		TestFormat: model.TestFormat{
			QuestionCount: 24,
			PassMarkPct:   75.0,
			TimeLimitSec:  2700,
		},
		Topics: []model.Topic{
			{ID: "history", Name: "History of the UK"},
			{ID: "values", Name: "British Values"},
		},
		Questions: []model.Question{
			{
				ID:           "q001",
				TopicID:      "history",
				Text:         "When was the Magna Carta sealed?",
				Choices:      []string{"1205", "1210", "1215", "1220"},
				CorrectIndex: 2,
				Explanation:  "The Magna Carta was sealed by King John in 1215.",
			},
			{
				ID:           "q002",
				TopicID:      "values",
				Text:         "What is the currency of the UK?",
				Choices:      []string{"Euro", "Dollar", "Pound sterling", "Franc"},
				CorrectIndex: 2,
				Explanation:  "The UK uses the pound sterling.",
			},
		},
	}
}

func TestSaveAndGetQuizPack(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	pack := samplePack()

	err := db.SaveQuizPack(ctx, pack)
	require.NoError(t, err)

	got, err := db.GetQuizPack(ctx, "life-in-uk")
	require.NoError(t, err)
	assert.Equal(t, pack.ID, got.ID)
	assert.Equal(t, pack.Name, got.Name)
	assert.Equal(t, pack.Description, got.Description)
	assert.Equal(t, pack.Version, got.Version)
	assert.Equal(t, pack.TestFormat.QuestionCount, got.TestFormat.QuestionCount)
	assert.Len(t, got.Topics, 2)
	assert.Len(t, got.Questions, 2)
	assert.Equal(t, "q001", got.Questions[0].ID)
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

	packs, err := db.ListQuizPacks(ctx)
	require.NoError(t, err)
	assert.Empty(t, packs)

	err = db.SaveQuizPack(ctx, samplePack())
	require.NoError(t, err)

	packs, err = db.ListQuizPacks(ctx)
	require.NoError(t, err)
	assert.Len(t, packs, 1)
	assert.Equal(t, "life-in-uk", packs[0].ID)
}

func TestSaveQuizPack_Upsert(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	pack := samplePack()

	err := db.SaveQuizPack(ctx, pack)
	require.NoError(t, err)

	pack.Version = "2.0.0"
	err = db.SaveQuizPack(ctx, pack)
	require.NoError(t, err)

	got, err := db.GetQuizPack(ctx, pack.ID)
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", got.Version)
}

func TestQuestionState(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	state := model.QuestionState{
		UserID:          "user1",
		PackID:          "pack1",
		QuestionID:      "q001",
		EaseFactor:      2.5,
		IntervalDays:    1.0,
		RepetitionCount: 1,
		NextReviewAt:    now.Add(24 * time.Hour),
		LastResult:      model.AnswerResultCorrect,
		LastReviewedAt:  now,
	}

	err := db.UpdateQuestionState(ctx, state)
	require.NoError(t, err)

	got, err := db.GetQuestionState(ctx, "user1", "pack1", "q001")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 2.5, got.EaseFactor)
	assert.Equal(t, 1.0, got.IntervalDays)
	assert.Equal(t, 1, got.RepetitionCount)
	assert.Equal(t, model.AnswerResultCorrect, got.LastResult)
}

func TestQuestionState_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	got, err := db.GetQuestionState(ctx, "user1", "pack1", "q999")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestTopicStats(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	stats := model.TopicStats{
		UserID:          "user1",
		PackID:          "pack1",
		TopicID:         "history",
		TotalAttempts:   10,
		CorrectCount:    7,
		RollingAccuracy: 0.7,
		CurrentStreak:   3,
		BestStreak:      5,
	}

	err := db.UpdateTopicStats(ctx, stats)
	require.NoError(t, err)

	got, err := db.GetTopicStats(ctx, "user1", "pack1", "history")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 10, got.TotalAttempts)
	assert.Equal(t, 7, got.CorrectCount)
	assert.InDelta(t, 0.7, got.RollingAccuracy, 0.001)
	assert.Equal(t, 3, got.CurrentStreak)
	assert.Equal(t, 5, got.BestStreak)
}

func TestTopicStats_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	got, err := db.GetTopicStats(ctx, "user1", "pack1", "missing")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestListTopicStats(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	stats1 := model.TopicStats{
		UserID: "user1", PackID: "pack1", TopicID: "history",
		TotalAttempts: 5, CorrectCount: 3, RollingAccuracy: 0.6,
	}
	stats2 := model.TopicStats{
		UserID: "user1", PackID: "pack1", TopicID: "values",
		TotalAttempts: 8, CorrectCount: 6, RollingAccuracy: 0.75,
	}

	require.NoError(t, db.UpdateTopicStats(ctx, stats1))
	require.NoError(t, db.UpdateTopicStats(ctx, stats2))

	list, err := db.ListTopicStats(ctx, "user1", "pack1")
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestListTopicStats_IsolatedByUser(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	stats1 := model.TopicStats{
		UserID: "user1", PackID: "pack1", TopicID: "history",
		TotalAttempts: 5, CorrectCount: 3,
	}
	stats2 := model.TopicStats{
		UserID: "user2", PackID: "pack1", TopicID: "history",
		TotalAttempts: 10, CorrectCount: 9,
	}

	require.NoError(t, db.UpdateTopicStats(ctx, stats1))
	require.NoError(t, db.UpdateTopicStats(ctx, stats2))

	list, err := db.ListTopicStats(ctx, "user1", "pack1")
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, 5, list[0].TotalAttempts)
}

func TestStudySession(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	session := model.StudySession{
		ID:            "sess1",
		UserID:        "user1",
		PackID:        "pack1",
		Mode:          model.SessionModePractice,
		StartedAt:     now,
		QuestionCount: 10,
		CorrectCount:  7,
		Attempts: []model.QuestionAttempt{
			{
				QuestionID:  "q001",
				AnswerIndex: 2,
				Correct:     true,
				TimeTakenMs: 5000,
				AnsweredAt:  now,
			},
		},
	}

	err := db.CreateSession(ctx, session)
	require.NoError(t, err)

	got, err := db.GetSession(ctx, "user1", "sess1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "sess1", got.ID)
	assert.Equal(t, model.SessionModePractice, got.Mode)
	assert.Equal(t, 10, got.QuestionCount)
	assert.Len(t, got.Attempts, 1)
	assert.Equal(t, "q001", got.Attempts[0].QuestionID)
}

func TestStudySession_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	got, err := db.GetSession(ctx, "user1", "missing")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestStudySession_Update(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	session := model.StudySession{
		ID:        "sess1",
		UserID:    "user1",
		PackID:    "pack1",
		Mode:      model.SessionModePractice,
		StartedAt: now,
		Attempts:  []model.QuestionAttempt{},
	}

	require.NoError(t, db.CreateSession(ctx, session))

	endedAt := now.Add(30 * time.Minute)
	session.EndedAt = &endedAt
	session.QuestionCount = 15
	session.CorrectCount = 12

	require.NoError(t, db.UpdateSession(ctx, session))

	got, err := db.GetSession(ctx, "user1", "sess1")
	require.NoError(t, err)
	require.NotNil(t, got.EndedAt)
	assert.Equal(t, 15, got.QuestionCount)
	assert.Equal(t, 12, got.CorrectCount)
}

func TestQuizSession(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	qs := model.QuizSession{
		ID:           "qs1",
		UserID:       "user1",
		PackID:       "pack1",
		Mode:         model.SessionModeMock,
		QuestionIDs:  []string{"q001", "q002", "q003"},
		CurrentIndex: 1,
		Answers:      map[string]int{"q001": 2},
		StartedAt:    now,
		TimeLimitSec: 2700,
		Status:       model.QuizSessionStatusInProgress,
	}

	err := db.SaveQuizSession(ctx, qs)
	require.NoError(t, err)

	got, err := db.GetQuizSession(ctx, "user1", "qs1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "qs1", got.ID)
	assert.Equal(t, model.SessionModeMock, got.Mode)
	assert.Equal(t, []string{"q001", "q002", "q003"}, got.QuestionIDs)
	assert.Equal(t, 1, got.CurrentIndex)
	assert.Equal(t, map[string]int{"q001": 2}, got.Answers)
	assert.Equal(t, 2700, got.TimeLimitSec)
	assert.Equal(t, model.QuizSessionStatusInProgress, got.Status)
}

func TestQuizSession_NotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	got, err := db.GetQuizSession(ctx, "user1", "missing")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestQuizSession_Upsert(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	qs := model.QuizSession{
		ID:           "qs1",
		UserID:       "user1",
		PackID:       "pack1",
		Mode:         model.SessionModeMock,
		QuestionIDs:  []string{"q001", "q002"},
		CurrentIndex: 0,
		Answers:      map[string]int{},
		StartedAt:    now,
		Status:       model.QuizSessionStatusInProgress,
	}

	require.NoError(t, db.SaveQuizSession(ctx, qs))

	qs.CurrentIndex = 2
	qs.Answers = map[string]int{"q001": 1, "q002": 3}
	qs.Status = model.QuizSessionStatusCompleted

	require.NoError(t, db.SaveQuizSession(ctx, qs))

	got, err := db.GetQuizSession(ctx, "user1", "qs1")
	require.NoError(t, err)
	assert.Equal(t, 2, got.CurrentIndex)
	assert.Equal(t, model.QuizSessionStatusCompleted, got.Status)
	assert.Len(t, got.Answers, 2)
}

func TestPreferences_DefaultOnFirstGet(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	got, err := db.GetPreferences(ctx, "user1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "user1", got.UserID)
	assert.Equal(t, 60, got.DeliveryIntervalMin)
	assert.Equal(t, 3, got.MaxUnanswered)
	assert.Equal(t, model.FocusModeSingle, got.FocusMode)
	assert.True(t, got.NotifyInactivity)
	assert.Equal(t, "22:00", got.QuietHoursStart)
}

func TestPreferences_Update(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	prefs := model.DefaultPreferences("user1")
	prefs.DeliveryIntervalMin = 30
	prefs.MaxUnanswered = 5
	prefs.ActivePackIDs = []string{"pack1", "pack2"}
	prefs.FocusMode = model.FocusModeInterleaved
	prefs.NotifyInactivity = false
	prefs.QuietHoursStart = "23:00"

	err := db.UpdatePreferences(ctx, prefs)
	require.NoError(t, err)

	got, err := db.GetPreferences(ctx, "user1")
	require.NoError(t, err)
	assert.Equal(t, 30, got.DeliveryIntervalMin)
	assert.Equal(t, 5, got.MaxUnanswered)
	assert.Equal(t, []string{"pack1", "pack2"}, got.ActivePackIDs)
	assert.Equal(t, model.FocusModeInterleaved, got.FocusMode)
	assert.False(t, got.NotifyInactivity)
	assert.Equal(t, "23:00", got.QuietHoursStart)
}

func TestUserIsolation_QuestionState(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	state1 := model.QuestionState{
		UserID: "user1", PackID: "pack1", QuestionID: "q001",
		EaseFactor: 2.5, IntervalDays: 1, NextReviewAt: now,
		LastResult: model.AnswerResultCorrect, LastReviewedAt: now,
	}
	state2 := model.QuestionState{
		UserID: "user2", PackID: "pack1", QuestionID: "q001",
		EaseFactor: 1.5, IntervalDays: 3, NextReviewAt: now,
		LastResult: model.AnswerResultWrong, LastReviewedAt: now,
	}

	require.NoError(t, db.UpdateQuestionState(ctx, state1))
	require.NoError(t, db.UpdateQuestionState(ctx, state2))

	got1, err := db.GetQuestionState(ctx, "user1", "pack1", "q001")
	require.NoError(t, err)
	assert.Equal(t, 2.5, got1.EaseFactor)

	got2, err := db.GetQuestionState(ctx, "user2", "pack1", "q001")
	require.NoError(t, err)
	assert.Equal(t, 1.5, got2.EaseFactor)
}
```

- [ ] **2.6 — Verify tests fail (TDD red phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./internal/store/sqlite/... 2>&1 | head -20
```

Expected: compilation errors because `sqlite.DB` and `sqlite.Open` do not exist yet.

- [ ] **2.7 — Install testify dependency**

```bash
cd /Users/samliem/LIFE_IN_UK
go get github.com/stretchr/testify
```

- [ ] **2.8 — Create `internal/store/sqlite/sqlite.go` — full SQLite implementation**

```go
// internal/store/sqlite/sqlite.go
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/sam-liem/quizbot/internal/model"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const timeFormat = time.RFC3339

// DB is the SQLite-backed implementation of store.Repository.
type DB struct {
	db *sql.DB
}

// Open creates a new DB connection and runs migrations.
func Open(dsn string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite: %w", err)
	}

	// Enable WAL mode for better concurrent read performance.
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}

	d := &DB{db: sqlDB}
	if err := d.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return d, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) migrate() error {
	data, err := migrationsFS.ReadFile("migrations/001_initial.sql")
	if err != nil {
		return fmt.Errorf("reading migration file: %w", err)
	}
	if _, err := d.db.Exec(string(data)); err != nil {
		return fmt.Errorf("executing migration: %w", err)
	}
	return nil
}

// --- Quiz Packs ---

func (d *DB) SaveQuizPack(ctx context.Context, pack model.QuizPack) error {
	data, err := json.Marshal(pack)
	if err != nil {
		return fmt.Errorf("marshaling quiz pack: %w", err)
	}
	_, err = d.db.ExecContext(ctx,
		`INSERT INTO quiz_packs (id, name, description, version, data)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name = excluded.name,
		   description = excluded.description,
		   version = excluded.version,
		   data = excluded.data`,
		pack.ID, pack.Name, pack.Description, pack.Version, string(data),
	)
	if err != nil {
		return fmt.Errorf("saving quiz pack: %w", err)
	}
	return nil
}

func (d *DB) GetQuizPack(ctx context.Context, packID string) (*model.QuizPack, error) {
	var data string
	err := d.db.QueryRowContext(ctx,
		`SELECT data FROM quiz_packs WHERE id = ?`, packID,
	).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting quiz pack: %w", err)
	}

	var pack model.QuizPack
	if err := json.Unmarshal([]byte(data), &pack); err != nil {
		return nil, fmt.Errorf("unmarshaling quiz pack: %w", err)
	}
	return &pack, nil
}

func (d *DB) ListQuizPacks(ctx context.Context) ([]model.QuizPack, error) {
	rows, err := d.db.QueryContext(ctx, `SELECT data FROM quiz_packs`)
	if err != nil {
		return nil, fmt.Errorf("listing quiz packs: %w", err)
	}
	defer rows.Close()

	var packs []model.QuizPack
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, fmt.Errorf("scanning quiz pack row: %w", err)
		}
		var pack model.QuizPack
		if err := json.Unmarshal([]byte(data), &pack); err != nil {
			return nil, fmt.Errorf("unmarshaling quiz pack: %w", err)
		}
		packs = append(packs, pack)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating quiz pack rows: %w", err)
	}
	return packs, nil
}

// --- Question State ---

func (d *DB) GetQuestionState(ctx context.Context, userID, packID, questionID string) (*model.QuestionState, error) {
	var s model.QuestionState
	var nextReview, lastReviewed string

	err := d.db.QueryRowContext(ctx,
		`SELECT user_id, pack_id, question_id, ease_factor, interval_days,
		        repetition_count, next_review_at, last_result, last_reviewed_at
		 FROM question_states
		 WHERE user_id = ? AND pack_id = ? AND question_id = ?`,
		userID, packID, questionID,
	).Scan(&s.UserID, &s.PackID, &s.QuestionID, &s.EaseFactor, &s.IntervalDays,
		&s.RepetitionCount, &nextReview, &s.LastResult, &lastReviewed)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting question state: %w", err)
	}

	if nextReview != "" {
		t, err := time.Parse(timeFormat, nextReview)
		if err != nil {
			return nil, fmt.Errorf("parsing next_review_at: %w", err)
		}
		s.NextReviewAt = t
	}
	if lastReviewed != "" {
		t, err := time.Parse(timeFormat, lastReviewed)
		if err != nil {
			return nil, fmt.Errorf("parsing last_reviewed_at: %w", err)
		}
		s.LastReviewedAt = t
	}

	return &s, nil
}

func (d *DB) UpdateQuestionState(ctx context.Context, state model.QuestionState) error {
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO question_states
		   (user_id, pack_id, question_id, ease_factor, interval_days,
		    repetition_count, next_review_at, last_result, last_reviewed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id, pack_id, question_id) DO UPDATE SET
		   ease_factor = excluded.ease_factor,
		   interval_days = excluded.interval_days,
		   repetition_count = excluded.repetition_count,
		   next_review_at = excluded.next_review_at,
		   last_result = excluded.last_result,
		   last_reviewed_at = excluded.last_reviewed_at`,
		state.UserID, state.PackID, state.QuestionID,
		state.EaseFactor, state.IntervalDays, state.RepetitionCount,
		state.NextReviewAt.Format(timeFormat),
		string(state.LastResult),
		state.LastReviewedAt.Format(timeFormat),
	)
	if err != nil {
		return fmt.Errorf("updating question state: %w", err)
	}
	return nil
}

// --- Topic Stats ---

func (d *DB) GetTopicStats(ctx context.Context, userID, packID, topicID string) (*model.TopicStats, error) {
	var s model.TopicStats
	err := d.db.QueryRowContext(ctx,
		`SELECT user_id, pack_id, topic_id, total_attempts, correct_count,
		        rolling_accuracy, current_streak, best_streak
		 FROM topic_stats
		 WHERE user_id = ? AND pack_id = ? AND topic_id = ?`,
		userID, packID, topicID,
	).Scan(&s.UserID, &s.PackID, &s.TopicID, &s.TotalAttempts, &s.CorrectCount,
		&s.RollingAccuracy, &s.CurrentStreak, &s.BestStreak)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting topic stats: %w", err)
	}
	return &s, nil
}

func (d *DB) UpdateTopicStats(ctx context.Context, stats model.TopicStats) error {
	_, err := d.db.ExecContext(ctx,
		`INSERT INTO topic_stats
		   (user_id, pack_id, topic_id, total_attempts, correct_count,
		    rolling_accuracy, current_streak, best_streak)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id, pack_id, topic_id) DO UPDATE SET
		   total_attempts = excluded.total_attempts,
		   correct_count = excluded.correct_count,
		   rolling_accuracy = excluded.rolling_accuracy,
		   current_streak = excluded.current_streak,
		   best_streak = excluded.best_streak`,
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
	rows, err := d.db.QueryContext(ctx,
		`SELECT user_id, pack_id, topic_id, total_attempts, correct_count,
		        rolling_accuracy, current_streak, best_streak
		 FROM topic_stats
		 WHERE user_id = ? AND pack_id = ?`,
		userID, packID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing topic stats: %w", err)
	}
	defer rows.Close()

	var stats []model.TopicStats
	for rows.Next() {
		var s model.TopicStats
		if err := rows.Scan(&s.UserID, &s.PackID, &s.TopicID,
			&s.TotalAttempts, &s.CorrectCount,
			&s.RollingAccuracy, &s.CurrentStreak, &s.BestStreak); err != nil {
			return nil, fmt.Errorf("scanning topic stats row: %w", err)
		}
		stats = append(stats, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating topic stats rows: %w", err)
	}
	return stats, nil
}

// --- Study Sessions ---

func (d *DB) CreateSession(ctx context.Context, session model.StudySession) error {
	attemptsJSON, err := json.Marshal(session.Attempts)
	if err != nil {
		return fmt.Errorf("marshaling attempts: %w", err)
	}

	var endedAt *string
	if session.EndedAt != nil {
		s := session.EndedAt.Format(timeFormat)
		endedAt = &s
	}

	_, err = d.db.ExecContext(ctx,
		`INSERT INTO study_sessions
		   (id, user_id, pack_id, mode, started_at, ended_at,
		    question_count, correct_count, attempts)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.PackID,
		string(session.Mode), session.StartedAt.Format(timeFormat),
		endedAt, session.QuestionCount, session.CorrectCount,
		string(attemptsJSON),
	)
	if err != nil {
		return fmt.Errorf("creating study session: %w", err)
	}
	return nil
}

func (d *DB) GetSession(ctx context.Context, userID, sessionID string) (*model.StudySession, error) {
	var s model.StudySession
	var startedAt string
	var endedAt sql.NullString
	var mode string
	var attemptsJSON string

	err := d.db.QueryRowContext(ctx,
		`SELECT id, user_id, pack_id, mode, started_at, ended_at,
		        question_count, correct_count, attempts
		 FROM study_sessions
		 WHERE user_id = ? AND id = ?`,
		userID, sessionID,
	).Scan(&s.ID, &s.UserID, &s.PackID, &mode, &startedAt, &endedAt,
		&s.QuestionCount, &s.CorrectCount, &attemptsJSON)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting study session: %w", err)
	}

	s.Mode = model.SessionMode(mode)

	t, err := time.Parse(timeFormat, startedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing started_at: %w", err)
	}
	s.StartedAt = t

	if endedAt.Valid {
		t, err := time.Parse(timeFormat, endedAt.String)
		if err != nil {
			return nil, fmt.Errorf("parsing ended_at: %w", err)
		}
		s.EndedAt = &t
	}

	if err := json.Unmarshal([]byte(attemptsJSON), &s.Attempts); err != nil {
		return nil, fmt.Errorf("unmarshaling attempts: %w", err)
	}

	return &s, nil
}

func (d *DB) UpdateSession(ctx context.Context, session model.StudySession) error {
	attemptsJSON, err := json.Marshal(session.Attempts)
	if err != nil {
		return fmt.Errorf("marshaling attempts: %w", err)
	}

	var endedAt *string
	if session.EndedAt != nil {
		s := session.EndedAt.Format(timeFormat)
		endedAt = &s
	}

	_, err = d.db.ExecContext(ctx,
		`UPDATE study_sessions SET
		   ended_at = ?, question_count = ?, correct_count = ?, attempts = ?
		 WHERE user_id = ? AND id = ?`,
		endedAt, session.QuestionCount, session.CorrectCount,
		string(attemptsJSON), session.UserID, session.ID,
	)
	if err != nil {
		return fmt.Errorf("updating study session: %w", err)
	}
	return nil
}

// --- Quiz Sessions ---

func (d *DB) SaveQuizSession(ctx context.Context, session model.QuizSession) error {
	questionIDsJSON, err := json.Marshal(session.QuestionIDs)
	if err != nil {
		return fmt.Errorf("marshaling question IDs: %w", err)
	}
	answersJSON, err := json.Marshal(session.Answers)
	if err != nil {
		return fmt.Errorf("marshaling answers: %w", err)
	}

	_, err = d.db.ExecContext(ctx,
		`INSERT INTO quiz_sessions
		   (id, user_id, pack_id, mode, question_ids, current_index,
		    answers, started_at, time_limit_sec, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   current_index = excluded.current_index,
		   answers = excluded.answers,
		   time_limit_sec = excluded.time_limit_sec,
		   status = excluded.status`,
		session.ID, session.UserID, session.PackID,
		string(session.Mode), string(questionIDsJSON),
		session.CurrentIndex, string(answersJSON),
		session.StartedAt.Format(timeFormat),
		session.TimeLimitSec, string(session.Status),
	)
	if err != nil {
		return fmt.Errorf("saving quiz session: %w", err)
	}
	return nil
}

func (d *DB) GetQuizSession(ctx context.Context, userID, sessionID string) (*model.QuizSession, error) {
	var s model.QuizSession
	var mode, status, startedAt string
	var questionIDsJSON, answersJSON string

	err := d.db.QueryRowContext(ctx,
		`SELECT id, user_id, pack_id, mode, question_ids, current_index,
		        answers, started_at, time_limit_sec, status
		 FROM quiz_sessions
		 WHERE user_id = ? AND id = ?`,
		userID, sessionID,
	).Scan(&s.ID, &s.UserID, &s.PackID, &mode, &questionIDsJSON,
		&s.CurrentIndex, &answersJSON, &startedAt, &s.TimeLimitSec, &status)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting quiz session: %w", err)
	}

	s.Mode = model.SessionMode(mode)
	s.Status = model.QuizSessionStatus(status)

	t, err := time.Parse(timeFormat, startedAt)
	if err != nil {
		return nil, fmt.Errorf("parsing started_at: %w", err)
	}
	s.StartedAt = t

	if err := json.Unmarshal([]byte(questionIDsJSON), &s.QuestionIDs); err != nil {
		return nil, fmt.Errorf("unmarshaling question IDs: %w", err)
	}
	if err := json.Unmarshal([]byte(answersJSON), &s.Answers); err != nil {
		return nil, fmt.Errorf("unmarshaling answers: %w", err)
	}

	return &s, nil
}

// --- User Preferences ---

func (d *DB) GetPreferences(ctx context.Context, userID string) (*model.UserPreferences, error) {
	var p model.UserPreferences
	var focusMode string
	var activePackIDsJSON string
	var notifyInactivity, notifyWeakTopic, notifyMilestones, notifyReadiness, notifyStreak int

	err := d.db.QueryRowContext(ctx,
		`SELECT user_id, delivery_interval_min, max_unanswered, active_pack_ids,
		        focus_mode, notify_inactivity, notify_inactivity_days,
		        notify_weak_topic, notify_weak_topic_pct, notify_milestones,
		        notify_readiness, notify_streak, quiet_hours_start, quiet_hours_end
		 FROM user_preferences
		 WHERE user_id = ?`,
		userID,
	).Scan(&p.UserID, &p.DeliveryIntervalMin, &p.MaxUnanswered, &activePackIDsJSON,
		&focusMode, &notifyInactivity, &p.NotifyInactivityDays,
		&notifyWeakTopic, &p.NotifyWeakTopicPct, &notifyMilestones,
		&notifyReadiness, &notifyStreak, &p.QuietHoursStart, &p.QuietHoursEnd)

	if err == sql.ErrNoRows {
		// Return defaults for a new user.
		defaults := model.DefaultPreferences(userID)
		return &defaults, nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting preferences: %w", err)
	}

	p.FocusMode = model.FocusMode(focusMode)
	p.NotifyInactivity = notifyInactivity != 0
	p.NotifyWeakTopic = notifyWeakTopic != 0
	p.NotifyMilestones = notifyMilestones != 0
	p.NotifyReadiness = notifyReadiness != 0
	p.NotifyStreak = notifyStreak != 0

	if err := json.Unmarshal([]byte(activePackIDsJSON), &p.ActivePackIDs); err != nil {
		return nil, fmt.Errorf("unmarshaling active_pack_ids: %w", err)
	}

	return &p, nil
}

func (d *DB) UpdatePreferences(ctx context.Context, prefs model.UserPreferences) error {
	activePackIDsJSON, err := json.Marshal(prefs.ActivePackIDs)
	if err != nil {
		return fmt.Errorf("marshaling active_pack_ids: %w", err)
	}

	_, err = d.db.ExecContext(ctx,
		`INSERT INTO user_preferences
		   (user_id, delivery_interval_min, max_unanswered, active_pack_ids,
		    focus_mode, notify_inactivity, notify_inactivity_days,
		    notify_weak_topic, notify_weak_topic_pct, notify_milestones,
		    notify_readiness, notify_streak, quiet_hours_start, quiet_hours_end)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   delivery_interval_min = excluded.delivery_interval_min,
		   max_unanswered = excluded.max_unanswered,
		   active_pack_ids = excluded.active_pack_ids,
		   focus_mode = excluded.focus_mode,
		   notify_inactivity = excluded.notify_inactivity,
		   notify_inactivity_days = excluded.notify_inactivity_days,
		   notify_weak_topic = excluded.notify_weak_topic,
		   notify_weak_topic_pct = excluded.notify_weak_topic_pct,
		   notify_milestones = excluded.notify_milestones,
		   notify_readiness = excluded.notify_readiness,
		   notify_streak = excluded.notify_streak,
		   quiet_hours_start = excluded.quiet_hours_start,
		   quiet_hours_end = excluded.quiet_hours_end`,
		prefs.UserID, prefs.DeliveryIntervalMin, prefs.MaxUnanswered,
		string(activePackIDsJSON), string(prefs.FocusMode),
		boolToInt(prefs.NotifyInactivity), prefs.NotifyInactivityDays,
		boolToInt(prefs.NotifyWeakTopic), prefs.NotifyWeakTopicPct,
		boolToInt(prefs.NotifyMilestones),
		boolToInt(prefs.NotifyReadiness),
		boolToInt(prefs.NotifyStreak),
		prefs.QuietHoursStart, prefs.QuietHoursEnd,
	)
	if err != nil {
		return fmt.Errorf("updating preferences: %w", err)
	}
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
```

- [ ] **2.9 — Run tests (TDD green phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test -v ./internal/store/sqlite/...
```

Expected: all tests pass.

- [ ] **2.10 — Verify repository interface conformance**

Add a compile-time check at the top of `sqlite.go` (after the type declaration):

Verify by building:

```bash
cd /Users/samliem/LIFE_IN_UK
go build ./internal/store/sqlite/...
```

Expected: clean build confirms `DB` satisfies `Repository`.

- [ ] **2.11 — Commit: "Add Repository interface and SQLite implementation"**

```bash
cd /Users/samliem/LIFE_IN_UK
git add internal/store/ go.mod go.sum
git commit -m "Add Repository interface and SQLite implementation

Define the full Repository interface scoped by user ID. Implement with
SQLite using modernc.org/sqlite (pure Go). Includes embedded SQL
migrations and comprehensive integration tests against in-memory SQLite."
```

---

## Task 3: SM-2 Spaced Repetition

**Goal:** Implement the SM-2 algorithm for scheduling question reviews and selecting the next question to study.

### Steps

- [ ] **3.1 — Write failing tests FIRST: `internal/core/repetition_test.go`**

```go
// internal/core/repetition_test.go
package core

import (
	"testing"
	"time"

	"github.com/sam-liem/quizbot/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestCalculateNextReview_FirstCorrectAnswer(t *testing.T) {
	now := time.Now().UTC()
	state := model.QuestionState{
		UserID:          "user1",
		PackID:          "pack1",
		QuestionID:      "q001",
		EaseFactor:      2.5,
		IntervalDays:    0,
		RepetitionCount: 0,
		NextReviewAt:    now,
	}

	result := CalculateNextReview(state, true, now)

	assert.Equal(t, 1, result.RepetitionCount)
	assert.Equal(t, 1.0, result.IntervalDays)
	assert.Equal(t, 2.5, result.EaseFactor) // unchanged on first correct
	assert.Equal(t, model.AnswerResultCorrect, result.LastResult)
	assert.Equal(t, now, result.LastReviewedAt)
	assert.WithinDuration(t, now.Add(24*time.Hour), result.NextReviewAt, time.Second)
}

func TestCalculateNextReview_SecondCorrectAnswer(t *testing.T) {
	now := time.Now().UTC()
	state := model.QuestionState{
		UserID:          "user1",
		PackID:          "pack1",
		QuestionID:      "q001",
		EaseFactor:      2.5,
		IntervalDays:    1.0,
		RepetitionCount: 1,
		NextReviewAt:    now,
	}

	result := CalculateNextReview(state, true, now)

	assert.Equal(t, 2, result.RepetitionCount)
	assert.Equal(t, 6.0, result.IntervalDays)
	assert.InDelta(t, 2.6, result.EaseFactor, 0.01)
	assert.Equal(t, model.AnswerResultCorrect, result.LastResult)
}

func TestCalculateNextReview_ThirdCorrectAnswer(t *testing.T) {
	now := time.Now().UTC()
	state := model.QuestionState{
		UserID:          "user1",
		PackID:          "pack1",
		QuestionID:      "q001",
		EaseFactor:      2.6,
		IntervalDays:    6.0,
		RepetitionCount: 2,
		NextReviewAt:    now,
	}

	result := CalculateNextReview(state, true, now)

	assert.Equal(t, 3, result.RepetitionCount)
	// interval = 6.0 * 2.6 = 15.6
	assert.InDelta(t, 15.6, result.IntervalDays, 0.1)
	// ease = 2.6 + (0.1 - 0*0.08 + 0*0.02) = 2.6 + 0.1 = 2.7
	assert.InDelta(t, 2.7, result.EaseFactor, 0.01)
}

func TestCalculateNextReview_WrongAnswer(t *testing.T) {
	now := time.Now().UTC()
	state := model.QuestionState{
		UserID:          "user1",
		PackID:          "pack1",
		QuestionID:      "q001",
		EaseFactor:      2.5,
		IntervalDays:    6.0,
		RepetitionCount: 3,
		NextReviewAt:    now,
	}

	result := CalculateNextReview(state, false, now)

	assert.Equal(t, 0, result.RepetitionCount)
	assert.Equal(t, 1.0, result.IntervalDays)
	assert.InDelta(t, 2.28, result.EaseFactor, 0.01)
	assert.Equal(t, model.AnswerResultWrong, result.LastResult)
	assert.WithinDuration(t, now.Add(24*time.Hour), result.NextReviewAt, time.Second)
}

func TestCalculateNextReview_EaseFloor(t *testing.T) {
	now := time.Now().UTC()
	state := model.QuestionState{
		UserID:          "user1",
		PackID:          "pack1",
		QuestionID:      "q001",
		EaseFactor:      1.3,
		IntervalDays:    1.0,
		RepetitionCount: 1,
		NextReviewAt:    now,
	}

	result := CalculateNextReview(state, false, now)

	assert.Equal(t, 1.3, result.EaseFactor) // must not go below 1.3
}

func TestCalculateNextReview_MultipleWrongAnswers(t *testing.T) {
	now := time.Now().UTC()
	state := model.QuestionState{
		EaseFactor:      2.5,
		IntervalDays:    15.0,
		RepetitionCount: 5,
	}

	// First wrong
	result := CalculateNextReview(state, false, now)
	assert.Equal(t, 1.0, result.IntervalDays)
	assert.Equal(t, 0, result.RepetitionCount)
	ease1 := result.EaseFactor

	// Second wrong
	result = CalculateNextReview(result, false, now)
	assert.Equal(t, 1.0, result.IntervalDays)
	assert.Equal(t, 0, result.RepetitionCount)
	assert.Less(t, result.EaseFactor, ease1) // ease decreases further

	// Third wrong — should not go below 1.3
	result.EaseFactor = 1.35
	result = CalculateNextReview(result, false, now)
	assert.GreaterOrEqual(t, result.EaseFactor, 1.3)
}

func TestSelectNextQuestion_MostOverdue(t *testing.T) {
	now := time.Now().UTC()
	states := []model.QuestionState{
		{QuestionID: "q1", NextReviewAt: now.Add(-1 * time.Hour)},   // 1 hour overdue
		{QuestionID: "q2", NextReviewAt: now.Add(-24 * time.Hour)},  // 24 hours overdue (most)
		{QuestionID: "q3", NextReviewAt: now.Add(1 * time.Hour)},    // not due yet
	}

	selected := SelectNextQuestion(states, now)
	assert.NotNil(t, selected)
	assert.Equal(t, "q2", selected.QuestionID)
}

func TestSelectNextQuestion_NothingDue(t *testing.T) {
	now := time.Now().UTC()
	states := []model.QuestionState{
		{QuestionID: "q1", NextReviewAt: now.Add(1 * time.Hour)},
		{QuestionID: "q2", NextReviewAt: now.Add(2 * time.Hour)},
	}

	selected := SelectNextQuestion(states, now)
	assert.Nil(t, selected)
}

func TestSelectNextQuestion_EmptySlice(t *testing.T) {
	now := time.Now().UTC()
	selected := SelectNextQuestion(nil, now)
	assert.Nil(t, selected)
}

func TestSelectNextQuestion_AllOverdue_PicksMostOverdue(t *testing.T) {
	now := time.Now().UTC()
	states := []model.QuestionState{
		{QuestionID: "q1", NextReviewAt: now.Add(-10 * time.Minute)},
		{QuestionID: "q2", NextReviewAt: now.Add(-5 * time.Hour)},
		{QuestionID: "q3", NextReviewAt: now.Add(-2 * time.Hour)},
	}

	selected := SelectNextQuestion(states, now)
	assert.NotNil(t, selected)
	assert.Equal(t, "q2", selected.QuestionID)
}
```

- [ ] **3.2 — Verify tests fail (TDD red phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./internal/core/... 2>&1 | head -10
```

Expected: compilation errors because `CalculateNextReview` and `SelectNextQuestion` do not exist.

- [ ] **3.3 — Implement `internal/core/repetition.go`**

```go
// internal/core/repetition.go
package core

import (
	"time"

	"github.com/sam-liem/quizbot/internal/model"
)

// CalculateNextReview applies the SM-2 spaced repetition algorithm to
// compute the next review state for a question.
//
// SM-2 rules:
//   - Correct: increase repetition count. Interval = 1 day (first),
//     6 days (second), then previous * ease. Ease increases slightly.
//   - Wrong: reset repetition count to 0 and interval to 1 day.
//     Ease decreases but never below 1.3.
func CalculateNextReview(state model.QuestionState, correct bool, now time.Time) model.QuestionState {
	result := state
	result.LastReviewedAt = now

	if correct {
		result.LastResult = model.AnswerResultCorrect
		result.RepetitionCount++

		switch result.RepetitionCount {
		case 1:
			result.IntervalDays = 1.0
		case 2:
			result.IntervalDays = 6.0
		default:
			result.IntervalDays = state.IntervalDays * state.EaseFactor
		}

		// SM-2 ease adjustment for correct answer:
		// EF' = EF + (0.1 - (5-q)*(0.08+(5-q)*0.02))
		// where q = 5 for a correct answer (perfect response)
		result.EaseFactor = state.EaseFactor + 0.1
	} else {
		result.LastResult = model.AnswerResultWrong
		result.RepetitionCount = 0
		result.IntervalDays = 1.0

		// SM-2 ease adjustment for wrong answer:
		// EF' = EF + (0.1 - (5-q)*(0.08+(5-q)*0.02))
		// where q = 1 for a wrong answer
		// = EF + (0.1 - 4*(0.08 + 4*0.02))
		// = EF + (0.1 - 4*0.16)
		// = EF + (0.1 - 0.64)
		// = EF - 0.54
		// But we commonly use q=2 for a wrong answer:
		// = EF + (0.1 - 3*(0.08 + 3*0.02))
		// = EF + (0.1 - 3*0.14)
		// = EF + (0.1 - 0.42)
		// = EF - 0.32
		// Using the standard SM-2 penalty for incorrect (q=2):
		result.EaseFactor = state.EaseFactor - 0.32
	}

	// Floor: ease factor must never go below 1.3
	if result.EaseFactor < 1.3 {
		result.EaseFactor = 1.3
	}

	// Calculate next review time from interval
	hours := result.IntervalDays * 24
	result.NextReviewAt = now.Add(time.Duration(hours * float64(time.Hour)))

	return result
}

// SelectNextQuestion picks the most overdue question from the given
// states. Returns nil if no question is currently due for review.
// A question is due when its NextReviewAt is at or before now.
func SelectNextQuestion(states []model.QuestionState, now time.Time) *model.QuestionState {
	var mostOverdue *model.QuestionState
	var maxOverdue time.Duration

	for i := range states {
		overdue := now.Sub(states[i].NextReviewAt)
		if overdue <= 0 {
			continue // not yet due
		}
		if mostOverdue == nil || overdue > maxOverdue {
			mostOverdue = &states[i]
			maxOverdue = overdue
		}
	}

	return mostOverdue
}
```

- [ ] **3.4 — Run tests (TDD green phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test -v ./internal/core/...
```

Expected: all tests pass.

- [ ] **3.5 — Commit: "Add SM-2 spaced repetition algorithm"**

```bash
cd /Users/samliem/LIFE_IN_UK
git add internal/core/repetition.go internal/core/repetition_test.go
git commit -m "Add SM-2 spaced repetition algorithm

Implement CalculateNextReview with SM-2 ease/interval adjustments and
SelectNextQuestion that picks the most overdue question. Table-driven
tests cover correct answers, wrong answers, ease floor, and overdue
selection."
```

---

## Task 4: Importer (YAML, JSON, Markdown + Validation)

**Goal:** Build quiz pack importers for three formats (YAML, JSON, Markdown) with schema validation, tested against fixture files.

### Steps

- [ ] **4.1 — Create test fixture: `internal/importer/testdata/valid_pack.yaml`**

```yaml
# internal/importer/testdata/valid_pack.yaml
id: "test-pack"
name: "Test Pack"
description: "A test quiz pack"
version: "1.0.0"
test_format:
  question_count: 2
  pass_mark_pct: 75.0
  time_limit_sec: 300
topics:
  - id: "topic1"
    name: "Topic One"
  - id: "topic2"
    name: "Topic Two"
questions:
  - id: "q001"
    topic_id: "topic1"
    text: "What is 2+2?"
    choices:
      - "3"
      - "4"
      - "5"
      - "6"
    correct_index: 1
    explanation: "2+2 equals 4."
  - id: "q002"
    topic_id: "topic2"
    text: "What color is the sky?"
    choices:
      - "Red"
      - "Green"
      - "Blue"
      - "Yellow"
    correct_index: 2
    explanation: "The sky appears blue due to Rayleigh scattering."
```

- [ ] **4.2 — Create test fixture: `internal/importer/testdata/valid_pack.json`**

```json
{
  "id": "test-pack",
  "name": "Test Pack",
  "description": "A test quiz pack",
  "version": "1.0.0",
  "test_format": {
    "question_count": 2,
    "pass_mark_pct": 75.0,
    "time_limit_sec": 300
  },
  "topics": [
    {"id": "topic1", "name": "Topic One"},
    {"id": "topic2", "name": "Topic Two"}
  ],
  "questions": [
    {
      "id": "q001",
      "topic_id": "topic1",
      "text": "What is 2+2?",
      "choices": ["3", "4", "5", "6"],
      "correct_index": 1,
      "explanation": "2+2 equals 4."
    },
    {
      "id": "q002",
      "topic_id": "topic2",
      "text": "What color is the sky?",
      "choices": ["Red", "Green", "Blue", "Yellow"],
      "correct_index": 2,
      "explanation": "The sky appears blue due to Rayleigh scattering."
    }
  ]
}
```

- [ ] **4.3 — Create test fixture: `internal/importer/testdata/valid_pack.md`**

```markdown
---
id: "test-pack"
name: "Test Pack"
description: "A test quiz pack"
version: "1.0.0"
test_format:
  question_count: 2
  pass_mark_pct: 75.0
  time_limit_sec: 300
topics:
  - id: "topic1"
    name: "Topic One"
  - id: "topic2"
    name: "Topic Two"
---

## Q: What is 2+2?
- 3
- 4
- 5
- 6
> answer: 1
> topic: topic1
> explanation: 2+2 equals 4.

## Q: What color is the sky?
- Red
- Green
- Blue
- Yellow
> answer: 2
> topic: topic2
> explanation: The sky appears blue due to Rayleigh scattering.
```

- [ ] **4.4 — Create test fixture: `internal/importer/testdata/invalid_missing_id.yaml`**

```yaml
# internal/importer/testdata/invalid_missing_id.yaml
name: "Bad Pack"
description: "Missing required id field"
version: "1.0.0"
test_format:
  question_count: 1
  pass_mark_pct: 75.0
  time_limit_sec: 300
topics:
  - id: "topic1"
    name: "Topic One"
questions:
  - id: "q001"
    topic_id: "topic1"
    text: "A question?"
    choices:
      - "A"
      - "B"
    correct_index: 0
```

- [ ] **4.5 — Create test fixture: `internal/importer/testdata/invalid_bad_index.yaml`**

```yaml
# internal/importer/testdata/invalid_bad_index.yaml
id: "bad-index-pack"
name: "Bad Index Pack"
description: "Has out-of-range correct_index"
version: "1.0.0"
test_format:
  question_count: 1
  pass_mark_pct: 75.0
  time_limit_sec: 300
topics:
  - id: "topic1"
    name: "Topic One"
questions:
  - id: "q001"
    topic_id: "topic1"
    text: "A question?"
    choices:
      - "A"
      - "B"
      - "C"
    correct_index: 5
```

- [ ] **4.6 — Write failing tests FIRST: `internal/importer/importer_test.go`**

```go
// internal/importer/importer_test.go
package importer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataPath(name string) string {
	return filepath.Join("testdata", name)
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     Format
		wantErr  bool
	}{
		{"yaml", "pack.yaml", FormatYAML, false},
		{"yml", "pack.yml", FormatYAML, false},
		{"json", "pack.json", FormatJSON, false},
		{"markdown", "pack.md", FormatMarkdown, false},
		{"unknown", "pack.txt", "", true},
		{"no ext", "pack", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectFormat(tt.filename)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestParseFile_YAML(t *testing.T) {
	pack, err := ParseFile(testdataPath("valid_pack.yaml"))
	require.NoError(t, err)

	assert.Equal(t, "test-pack", pack.ID)
	assert.Equal(t, "Test Pack", pack.Name)
	assert.Equal(t, "1.0.0", pack.Version)
	assert.Equal(t, 2, pack.TestFormat.QuestionCount)
	assert.InDelta(t, 75.0, pack.TestFormat.PassMarkPct, 0.01)
	assert.Equal(t, 300, pack.TestFormat.TimeLimitSec)
	assert.Len(t, pack.Topics, 2)
	assert.Len(t, pack.Questions, 2)
	assert.Equal(t, "q001", pack.Questions[0].ID)
	assert.Equal(t, 1, pack.Questions[0].CorrectIndex)
	assert.Equal(t, "2+2 equals 4.", pack.Questions[0].Explanation)
}

func TestParseFile_JSON(t *testing.T) {
	pack, err := ParseFile(testdataPath("valid_pack.json"))
	require.NoError(t, err)

	assert.Equal(t, "test-pack", pack.ID)
	assert.Equal(t, "Test Pack", pack.Name)
	assert.Len(t, pack.Topics, 2)
	assert.Len(t, pack.Questions, 2)
	assert.Equal(t, "q001", pack.Questions[0].ID)
	assert.Equal(t, 1, pack.Questions[0].CorrectIndex)
}

func TestParseFile_Markdown(t *testing.T) {
	pack, err := ParseFile(testdataPath("valid_pack.md"))
	require.NoError(t, err)

	assert.Equal(t, "test-pack", pack.ID)
	assert.Equal(t, "Test Pack", pack.Name)
	assert.Len(t, pack.Topics, 2)
	assert.Len(t, pack.Questions, 2)

	q1 := pack.Questions[0]
	assert.Equal(t, "What is 2+2?", q1.Text)
	assert.Equal(t, []string{"3", "4", "5", "6"}, q1.Choices)
	assert.Equal(t, 1, q1.CorrectIndex)
	assert.Equal(t, "topic1", q1.TopicID)
	assert.Equal(t, "2+2 equals 4.", q1.Explanation)

	q2 := pack.Questions[1]
	assert.Equal(t, "What color is the sky?", q2.Text)
	assert.Equal(t, 2, q2.CorrectIndex)
}

func TestParseFile_InvalidMissingID(t *testing.T) {
	_, err := ParseFile(testdataPath("invalid_missing_id.yaml"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "id")
}

func TestParseFile_InvalidBadIndex(t *testing.T) {
	_, err := ParseFile(testdataPath("invalid_bad_index.yaml"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "correct_index")
}

func TestParseFile_NonexistentFile(t *testing.T) {
	_, err := ParseFile(testdataPath("does_not_exist.yaml"))
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err) || assert.Contains(t, err.Error(), "opening"))
}

func TestParseReader_YAML(t *testing.T) {
	f, err := os.Open(testdataPath("valid_pack.yaml"))
	require.NoError(t, err)
	defer f.Close()

	parser := &YAMLParser{}
	pack, err := parser.Parse(f)
	require.NoError(t, err)
	assert.Equal(t, "test-pack", pack.ID)
}

func TestParseReader_JSON(t *testing.T) {
	f, err := os.Open(testdataPath("valid_pack.json"))
	require.NoError(t, err)
	defer f.Close()

	parser := &JSONParser{}
	pack, err := parser.Parse(f)
	require.NoError(t, err)
	assert.Equal(t, "test-pack", pack.ID)
}

func TestParseReader_Markdown(t *testing.T) {
	f, err := os.Open(testdataPath("valid_pack.md"))
	require.NoError(t, err)
	defer f.Close()

	parser := &MarkdownParser{}
	pack, err := parser.Parse(f)
	require.NoError(t, err)
	assert.Equal(t, "test-pack", pack.ID)
	assert.Len(t, pack.Questions, 2)
}
```

- [ ] **4.7 — Verify tests fail (TDD red phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./internal/importer/... 2>&1 | head -10
```

Expected: compilation errors because the importer types do not exist.

- [ ] **4.8 — Create `internal/importer/validate.go` — validation logic**

```go
// internal/importer/validate.go
package importer

import (
	"fmt"
	"strings"

	"github.com/sam-liem/quizbot/internal/model"
)

// Validate checks a parsed QuizPack for structural correctness.
// Returns a descriptive error if the pack is invalid.
func Validate(pack model.QuizPack) error {
	var errs []string

	if pack.ID == "" {
		errs = append(errs, "pack id is required")
	}
	if pack.Name == "" {
		errs = append(errs, "pack name is required")
	}
	if pack.Version == "" {
		errs = append(errs, "pack version is required")
	}
	if pack.TestFormat.QuestionCount <= 0 {
		errs = append(errs, "test_format.question_count must be positive")
	}
	if pack.TestFormat.PassMarkPct <= 0 || pack.TestFormat.PassMarkPct > 100 {
		errs = append(errs, "test_format.pass_mark_pct must be between 0 and 100")
	}

	// Build a set of valid topic IDs.
	topicIDs := make(map[string]bool, len(pack.Topics))
	for _, topic := range pack.Topics {
		if topic.ID == "" {
			errs = append(errs, "topic id is required")
			continue
		}
		if topic.Name == "" {
			errs = append(errs, fmt.Sprintf("topic %q: name is required", topic.ID))
		}
		if topicIDs[topic.ID] {
			errs = append(errs, fmt.Sprintf("duplicate topic id %q", topic.ID))
		}
		topicIDs[topic.ID] = true
	}

	// Validate questions.
	questionIDs := make(map[string]bool, len(pack.Questions))
	for _, q := range pack.Questions {
		if q.ID == "" {
			errs = append(errs, "question id is required")
			continue
		}
		if questionIDs[q.ID] {
			errs = append(errs, fmt.Sprintf("duplicate question id %q", q.ID))
		}
		questionIDs[q.ID] = true

		if q.Text == "" {
			errs = append(errs, fmt.Sprintf("question %q: text is required", q.ID))
		}
		if len(q.Choices) < 2 {
			errs = append(errs, fmt.Sprintf("question %q: at least 2 choices required", q.ID))
		}
		if q.CorrectIndex < 0 || q.CorrectIndex >= len(q.Choices) {
			errs = append(errs, fmt.Sprintf("question %q: correct_index %d out of range [0, %d)",
				q.ID, q.CorrectIndex, len(q.Choices)))
		}
		if q.TopicID != "" && !topicIDs[q.TopicID] {
			errs = append(errs, fmt.Sprintf("question %q: topic_id %q not found in topics",
				q.ID, q.TopicID))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid quiz pack: %s", strings.Join(errs, "; "))
	}
	return nil
}
```

- [ ] **4.9 — Create `internal/importer/importer.go` — interface, DetectFormat, ParseFile**

```go
// internal/importer/importer.go
package importer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sam-liem/quizbot/internal/model"
)

// Format represents a supported quiz pack file format.
type Format string

const (
	FormatYAML     Format = "yaml"
	FormatJSON     Format = "json"
	FormatMarkdown Format = "markdown"
)

// QuestionParser parses a quiz pack from a reader.
type QuestionParser interface {
	Parse(r io.Reader) (*model.QuizPack, error)
}

// DetectFormat determines the import format from a filename extension.
func DetectFormat(filename string) (Format, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".yaml", ".yml":
		return FormatYAML, nil
	case ".json":
		return FormatJSON, nil
	case ".md", ".markdown":
		return FormatMarkdown, nil
	default:
		return "", fmt.Errorf("unsupported file extension %q: expected .yaml, .yml, .json, or .md", ext)
	}
}

// ParserForFormat returns the appropriate parser for a given format.
func ParserForFormat(format Format) (QuestionParser, error) {
	switch format {
	case FormatYAML:
		return &YAMLParser{}, nil
	case FormatJSON:
		return &JSONParser{}, nil
	case FormatMarkdown:
		return &MarkdownParser{}, nil
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

// ParseFile reads a quiz pack from a file, auto-detecting the format
// from the extension. The pack is validated after parsing.
func ParseFile(path string) (*model.QuizPack, error) {
	format, err := DetectFormat(path)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	parser, err := ParserForFormat(format)
	if err != nil {
		return nil, err
	}

	pack, err := parser.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if err := Validate(*pack); err != nil {
		return nil, err
	}

	return pack, nil
}
```

- [ ] **4.10 — Create `internal/importer/yaml.go` — YAML parser**

```go
// internal/importer/yaml.go
package importer

import (
	"fmt"
	"io"

	"github.com/sam-liem/quizbot/internal/model"
	"gopkg.in/yaml.v3"
)

// YAMLParser implements QuestionParser for YAML-formatted quiz packs.
type YAMLParser struct{}

func (p *YAMLParser) Parse(r io.Reader) (*model.QuizPack, error) {
	var pack model.QuizPack
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&pack); err != nil {
		return nil, fmt.Errorf("decoding YAML: %w", err)
	}
	return &pack, nil
}
```

- [ ] **4.11 — Create `internal/importer/json.go` — JSON parser**

```go
// internal/importer/json.go
package importer

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/sam-liem/quizbot/internal/model"
)

// JSONParser implements QuestionParser for JSON-formatted quiz packs.
type JSONParser struct{}

func (p *JSONParser) Parse(r io.Reader) (*model.QuizPack, error) {
	var pack model.QuizPack
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&pack); err != nil {
		return nil, fmt.Errorf("decoding JSON: %w", err)
	}
	return &pack, nil
}
```

- [ ] **4.12 — Create `internal/importer/markdown.go` — Markdown parser**

```go
// internal/importer/markdown.go
package importer

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/sam-liem/quizbot/internal/model"
	"gopkg.in/yaml.v3"
)

// MarkdownParser implements QuestionParser for Markdown-formatted quiz
// packs. The format uses YAML frontmatter between --- markers for pack
// metadata, followed by questions in the format:
//
//	## Q: question text
//	- choice 1
//	- choice 2
//	> answer: index
//	> topic: id
//	> explanation: text
type MarkdownParser struct{}

func (p *MarkdownParser) Parse(r io.Reader) (*model.QuizPack, error) {
	scanner := bufio.NewScanner(r)

	// Phase 1: Extract YAML frontmatter.
	frontmatter, err := extractFrontmatter(scanner)
	if err != nil {
		return nil, fmt.Errorf("extracting frontmatter: %w", err)
	}

	// Parse frontmatter as YAML to get pack metadata (everything except questions).
	var pack model.QuizPack
	if err := yaml.Unmarshal([]byte(frontmatter), &pack); err != nil {
		return nil, fmt.Errorf("parsing frontmatter YAML: %w", err)
	}

	// Phase 2: Parse questions from the body.
	questions, err := parseMarkdownQuestions(scanner)
	if err != nil {
		return nil, fmt.Errorf("parsing questions: %w", err)
	}
	pack.Questions = questions

	return &pack, nil
}

// extractFrontmatter reads lines between the opening and closing ---
// markers and returns the YAML content.
func extractFrontmatter(scanner *bufio.Scanner) (string, error) {
	// Find the opening ---.
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "---" {
			break
		}
	}

	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

// parseMarkdownQuestions parses question blocks from the body of the
// markdown file (after frontmatter).
func parseMarkdownQuestions(scanner *bufio.Scanner) ([]model.Question, error) {
	var questions []model.Question
	var current *model.Question
	questionCount := 0

	flushCurrent := func() {
		if current != nil {
			questions = append(questions, *current)
			current = nil
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Skip blank lines.
		if trimmed == "" {
			continue
		}

		// New question: ## Q: text
		if strings.HasPrefix(trimmed, "## Q:") || strings.HasPrefix(trimmed, "## Q :") {
			flushCurrent()
			questionCount++
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, "## Q:"))
			text = strings.TrimSpace(strings.TrimPrefix(text, "## Q :"))
			current = &model.Question{
				ID:   fmt.Sprintf("q%03d", questionCount),
				Text: text,
			}
			continue
		}

		if current == nil {
			continue
		}

		// Choice: - text
		if strings.HasPrefix(trimmed, "- ") {
			choice := strings.TrimPrefix(trimmed, "- ")
			current.Choices = append(current.Choices, choice)
			continue
		}

		// Metadata: > key: value
		if strings.HasPrefix(trimmed, "> ") {
			meta := strings.TrimPrefix(trimmed, "> ")
			key, value, found := strings.Cut(meta, ":")
			if !found {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)

			switch key {
			case "answer":
				idx, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("question %q: invalid answer index %q: %w",
						current.ID, value, err)
				}
				current.CorrectIndex = idx
			case "topic":
				current.TopicID = value
			case "explanation":
				current.Explanation = value
			}
			continue
		}
	}

	flushCurrent()

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return questions, nil
}
```

- [ ] **4.13 — Install yaml.v3 dependency**

```bash
cd /Users/samliem/LIFE_IN_UK
go get gopkg.in/yaml.v3
```

- [ ] **4.14 — Run tests (TDD green phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test -v ./internal/importer/...
```

Expected: all tests pass.

- [ ] **4.15 — Commit: "Add quiz pack importers for YAML, JSON, and Markdown"**

```bash
cd /Users/samliem/LIFE_IN_UK
git add internal/importer/ go.mod go.sum
git commit -m "Add quiz pack importers for YAML, JSON, and Markdown

Implement QuestionParser interface with three parsers: YAML, JSON, and
Markdown with YAML frontmatter. Includes format auto-detection from
file extension, schema validation, and test fixtures for valid and
invalid packs."
```

---

## Task 5: Stats Engine

**Goal:** Implement readiness score calculation and topic breakdown, tested with table-driven tests.

### Steps

- [ ] **5.1 — Write failing tests FIRST: `internal/core/stats_test.go`**

```go
// internal/core/stats_test.go
package core

import (
	"testing"

	"github.com/sam-liem/quizbot/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateReadiness_AllTopicsPerfect(t *testing.T) {
	pack := model.QuizPack{
		Topics: []model.Topic{
			{ID: "t1", Name: "Topic 1"},
			{ID: "t2", Name: "Topic 2"},
		},
		Questions: []model.Question{
			{ID: "q1", TopicID: "t1"},
			{ID: "q2", TopicID: "t1"},
			{ID: "q3", TopicID: "t2"},
			{ID: "q4", TopicID: "t2"},
		},
	}

	topicStats := []model.TopicStats{
		{TopicID: "t1", TotalAttempts: 10, CorrectCount: 10, RollingAccuracy: 1.0},
		{TopicID: "t2", TotalAttempts: 10, CorrectCount: 10, RollingAccuracy: 1.0},
	}

	readiness := CalculateReadiness(topicStats, pack)
	assert.InDelta(t, 1.0, readiness, 0.01)
}

func TestCalculateReadiness_MixedAccuracy(t *testing.T) {
	pack := model.QuizPack{
		Topics: []model.Topic{
			{ID: "t1", Name: "Topic 1"},
			{ID: "t2", Name: "Topic 2"},
		},
		Questions: []model.Question{
			{ID: "q1", TopicID: "t1"},
			{ID: "q2", TopicID: "t1"},
			{ID: "q3", TopicID: "t1"}, // 3 questions in t1
			{ID: "q4", TopicID: "t2"}, // 1 question in t2
		},
	}

	topicStats := []model.TopicStats{
		{TopicID: "t1", RollingAccuracy: 0.8},
		{TopicID: "t2", RollingAccuracy: 0.5},
	}

	// Weighted: (3/4)*0.8 + (1/4)*0.5 = 0.6 + 0.125 = 0.725
	readiness := CalculateReadiness(topicStats, pack)
	assert.InDelta(t, 0.725, readiness, 0.01)
}

func TestCalculateReadiness_NoStats(t *testing.T) {
	pack := model.QuizPack{
		Topics: []model.Topic{
			{ID: "t1", Name: "Topic 1"},
		},
		Questions: []model.Question{
			{ID: "q1", TopicID: "t1"},
		},
	}

	readiness := CalculateReadiness(nil, pack)
	assert.Equal(t, 0.0, readiness)
}

func TestCalculateReadiness_EmptyPack(t *testing.T) {
	pack := model.QuizPack{}
	readiness := CalculateReadiness(nil, pack)
	assert.Equal(t, 0.0, readiness)
}

func TestCalculateReadiness_TopicWithNoStats(t *testing.T) {
	pack := model.QuizPack{
		Topics: []model.Topic{
			{ID: "t1", Name: "Topic 1"},
			{ID: "t2", Name: "Topic 2"},
		},
		Questions: []model.Question{
			{ID: "q1", TopicID: "t1"},
			{ID: "q2", TopicID: "t2"},
		},
	}

	// Only t1 has stats; t2 should be treated as 0% accuracy.
	topicStats := []model.TopicStats{
		{TopicID: "t1", RollingAccuracy: 0.8},
	}

	// Weighted: (1/2)*0.8 + (1/2)*0.0 = 0.4
	readiness := CalculateReadiness(topicStats, pack)
	assert.InDelta(t, 0.4, readiness, 0.01)
}

func TestGetTopicBreakdown(t *testing.T) {
	topicStats := []model.TopicStats{
		{
			TopicID: "history", TotalAttempts: 20, CorrectCount: 15,
			RollingAccuracy: 0.75, CurrentStreak: 3, BestStreak: 8,
		},
		{
			TopicID: "values", TotalAttempts: 10, CorrectCount: 9,
			RollingAccuracy: 0.9, CurrentStreak: 5, BestStreak: 5,
		},
		{
			TopicID: "government", TotalAttempts: 5, CorrectCount: 2,
			RollingAccuracy: 0.4, CurrentStreak: 0, BestStreak: 2,
		},
	}

	breakdown := GetTopicBreakdown(topicStats)
	require.Len(t, breakdown, 3)

	// Should be sorted weakest first.
	assert.Equal(t, "government", breakdown[0].TopicID)
	assert.InDelta(t, 0.4, breakdown[0].Accuracy, 0.01)
	assert.Equal(t, 5, breakdown[0].TotalAttempts)
	assert.Equal(t, 2, breakdown[0].CorrectCount)

	assert.Equal(t, "history", breakdown[1].TopicID)
	assert.InDelta(t, 0.75, breakdown[1].Accuracy, 0.01)

	assert.Equal(t, "values", breakdown[2].TopicID)
	assert.InDelta(t, 0.9, breakdown[2].Accuracy, 0.01)
}

func TestGetTopicBreakdown_Empty(t *testing.T) {
	breakdown := GetTopicBreakdown(nil)
	assert.Empty(t, breakdown)
}

func TestGetTopicBreakdown_SingleTopic(t *testing.T) {
	topicStats := []model.TopicStats{
		{
			TopicID: "history", TotalAttempts: 10, CorrectCount: 7,
			RollingAccuracy: 0.7, CurrentStreak: 2, BestStreak: 4,
		},
	}

	breakdown := GetTopicBreakdown(topicStats)
	require.Len(t, breakdown, 1)
	assert.Equal(t, "history", breakdown[0].TopicID)
	assert.InDelta(t, 0.7, breakdown[0].Accuracy, 0.01)
	assert.Equal(t, 10, breakdown[0].TotalAttempts)
	assert.Equal(t, 7, breakdown[0].CorrectCount)
	assert.Equal(t, 2, breakdown[0].CurrentStreak)
	assert.Equal(t, 4, breakdown[0].BestStreak)
}
```

- [ ] **5.2 — Verify tests fail (TDD red phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test ./internal/core/... 2>&1 | head -10
```

Expected: compilation errors because `CalculateReadiness`, `GetTopicBreakdown`, and `TopicSummary` do not exist.

- [ ] **5.3 — Implement `internal/core/stats.go`**

```go
// internal/core/stats.go
package core

import (
	"sort"

	"github.com/sam-liem/quizbot/internal/model"
)

// TopicSummary is a display-oriented summary of a single topic's
// performance, used in stats breakdowns.
type TopicSummary struct {
	TopicID       string  `json:"topic_id"`
	Accuracy      float64 `json:"accuracy"`
	TotalAttempts int     `json:"total_attempts"`
	CorrectCount  int     `json:"correct_count"`
	CurrentStreak int     `json:"current_streak"`
	BestStreak    int     `json:"best_streak"`
}

// CalculateReadiness computes an estimated pass probability as a
// weighted average of topic accuracies, where each topic's weight is
// proportional to the number of questions it contributes to the pack.
//
// Topics with no stats are treated as 0% accuracy.
func CalculateReadiness(topicStats []model.TopicStats, pack model.QuizPack) float64 {
	totalQuestions := len(pack.Questions)
	if totalQuestions == 0 {
		return 0.0
	}

	// Count questions per topic.
	questionsByTopic := make(map[string]int)
	for _, q := range pack.Questions {
		questionsByTopic[q.TopicID]++
	}

	// Build a lookup for topic stats by topic ID.
	statsByTopic := make(map[string]model.TopicStats)
	for _, s := range topicStats {
		statsByTopic[s.TopicID] = s
	}

	// Weighted average of topic accuracies.
	var weightedSum float64
	for topicID, count := range questionsByTopic {
		weight := float64(count) / float64(totalQuestions)
		accuracy := 0.0
		if s, ok := statsByTopic[topicID]; ok {
			accuracy = s.RollingAccuracy
		}
		weightedSum += weight * accuracy
	}

	return weightedSum
}

// GetTopicBreakdown converts raw topic stats into display-oriented
// summaries, sorted weakest-first (lowest accuracy first).
func GetTopicBreakdown(topicStats []model.TopicStats) []TopicSummary {
	if len(topicStats) == 0 {
		return nil
	}

	summaries := make([]TopicSummary, len(topicStats))
	for i, s := range topicStats {
		summaries[i] = TopicSummary{
			TopicID:       s.TopicID,
			Accuracy:      s.RollingAccuracy,
			TotalAttempts: s.TotalAttempts,
			CorrectCount:  s.CorrectCount,
			CurrentStreak: s.CurrentStreak,
			BestStreak:    s.BestStreak,
		}
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Accuracy < summaries[j].Accuracy
	})

	return summaries
}
```

- [ ] **5.4 — Run tests (TDD green phase)**

```bash
cd /Users/samliem/LIFE_IN_UK
go test -v ./internal/core/...
```

Expected: all tests pass (both repetition and stats tests).

- [ ] **5.5 — Commit: "Add stats engine with readiness score and topic breakdown"**

```bash
cd /Users/samliem/LIFE_IN_UK
git add internal/core/stats.go internal/core/stats_test.go
git commit -m "Add stats engine with readiness score and topic breakdown

Implement CalculateReadiness as a weighted average of topic accuracies
by question distribution, and GetTopicBreakdown for display-oriented
topic summaries sorted weakest-first. Table-driven tests cover perfect
scores, mixed accuracy, missing stats, and empty packs."
```
