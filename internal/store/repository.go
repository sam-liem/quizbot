package store

import (
	"context"

	"github.com/sam-liem/quizbot/internal/model"
)

// Repository defines all data access operations, scoped by user ID for multi-tenancy.
type Repository interface {
	// Quiz packs
	SaveQuizPack(ctx context.Context, pack model.QuizPack) error
	GetQuizPack(ctx context.Context, packID string) (*model.QuizPack, error)
	ListQuizPacks(ctx context.Context) ([]model.QuizPack, error)

	// Question state (spaced repetition)
	GetQuestionState(ctx context.Context, userID, packID, questionID string) (*model.QuestionState, error)
	UpdateQuestionState(ctx context.Context, state model.QuestionState) error

	// Topic stats
	GetTopicStats(ctx context.Context, userID, packID, topicID string) (*model.TopicStats, error)
	UpdateTopicStats(ctx context.Context, stats model.TopicStats) error
	ListTopicStats(ctx context.Context, userID, packID string) ([]model.TopicStats, error)

	// Study sessions
	CreateSession(ctx context.Context, session model.StudySession) error
	GetSession(ctx context.Context, userID, sessionID string) (*model.StudySession, error)
	UpdateSession(ctx context.Context, session model.StudySession) error

	// Quiz sessions (active quiz state)
	SaveQuizSession(ctx context.Context, session model.QuizSession) error
	GetQuizSession(ctx context.Context, userID, sessionID string) (*model.QuizSession, error)
	ListQuizSessions(ctx context.Context, userID string, status model.QuizSessionStatus) ([]model.QuizSession, error)

	// User preferences
	GetPreferences(ctx context.Context, userID string) (*model.UserPreferences, error)
	UpdatePreferences(ctx context.Context, prefs model.UserPreferences) error
}
