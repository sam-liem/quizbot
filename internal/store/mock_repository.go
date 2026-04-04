package store

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/sam-liem/quizbot/internal/model"
)

// Compile-time interface check.
var _ Repository = (*MockRepository)(nil)

// MockRepository is an in-memory implementation of Repository for testing.
type MockRepository struct {
	mu             sync.Mutex
	QuizPacks      map[string]model.QuizPack
	QuestionStates map[string]model.QuestionState
	TopicStats     map[string]model.TopicStats
	StudySessions  map[string]model.StudySession
	QuizSessions   map[string]model.QuizSession
	Preferences    map[string]model.UserPreferences
}

// NewMockRepository creates an initialised MockRepository.
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

// --- Key helpers ---

func questionStateKey(userID, packID, questionID string) string {
	return strings.Join([]string{userID, packID, questionID}, "|")
}

func topicStatsKey(userID, packID, topicID string) string {
	return strings.Join([]string{userID, packID, topicID}, "|")
}

func sessionKey(userID, sessionID string) string {
	return strings.Join([]string{userID, sessionID}, "|")
}

// --- Quiz packs ---

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
		return nil, fmt.Errorf("quiz pack %q not found", packID)
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

// --- Question state ---

func (m *MockRepository) GetQuestionState(_ context.Context, userID, packID, questionID string) (*model.QuestionState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := questionStateKey(userID, packID, questionID)
	state, ok := m.QuestionStates[key]
	if !ok {
		return nil, nil
	}
	return &state, nil
}

func (m *MockRepository) UpdateQuestionState(_ context.Context, state model.QuestionState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := questionStateKey(state.UserID, state.PackID, state.QuestionID)
	m.QuestionStates[key] = state
	return nil
}

// --- Topic stats ---

func (m *MockRepository) GetTopicStats(_ context.Context, userID, packID, topicID string) (*model.TopicStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := topicStatsKey(userID, packID, topicID)
	stats, ok := m.TopicStats[key]
	if !ok {
		return nil, nil
	}
	return &stats, nil
}

func (m *MockRepository) UpdateTopicStats(_ context.Context, stats model.TopicStats) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := topicStatsKey(stats.UserID, stats.PackID, stats.TopicID)
	m.TopicStats[key] = stats
	return nil
}

// ListTopicStats returns all topic stats for the given userID and packID.
func (m *MockRepository) ListTopicStats(_ context.Context, userID, packID string) ([]model.TopicStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := strings.Join([]string{userID, packID, ""}, "|")
	var results []model.TopicStats
	for key, ts := range m.TopicStats {
		if strings.HasPrefix(key, prefix) {
			results = append(results, ts)
		}
	}
	return results, nil
}

// --- Study sessions ---

func (m *MockRepository) CreateSession(_ context.Context, session model.StudySession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := sessionKey(session.UserID, session.ID)
	m.StudySessions[key] = session
	return nil
}

func (m *MockRepository) GetSession(_ context.Context, userID, sessionID string) (*model.StudySession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := sessionKey(userID, sessionID)
	session, ok := m.StudySessions[key]
	if !ok {
		return nil, fmt.Errorf("study session %q not found for user %q", sessionID, userID)
	}
	return &session, nil
}

func (m *MockRepository) UpdateSession(_ context.Context, session model.StudySession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := sessionKey(session.UserID, session.ID)
	m.StudySessions[key] = session
	return nil
}

// --- Quiz sessions ---

func (m *MockRepository) SaveQuizSession(_ context.Context, session model.QuizSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := sessionKey(session.UserID, session.ID)
	m.QuizSessions[key] = session
	return nil
}

func (m *MockRepository) GetQuizSession(_ context.Context, userID, sessionID string) (*model.QuizSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := sessionKey(userID, sessionID)
	session, ok := m.QuizSessions[key]
	if !ok {
		return nil, fmt.Errorf("quiz session %q not found for user %q", sessionID, userID)
	}
	return &session, nil
}

// --- User preferences ---

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
