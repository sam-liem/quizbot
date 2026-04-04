package messenger

import (
	"fmt"
	"sync"

	"github.com/sam-liem/quizbot/internal/core"
	"github.com/sam-liem/quizbot/internal/model"
)

// Compile-time interface check.
var _ Messenger = (*MockMessenger)(nil)

// sentQuestion records a question sent to a chat.
type sentQuestion struct {
	ChatID   string
	Question model.Question
}

// sentFeedback records feedback sent to a chat.
type sentFeedback struct {
	ChatID      string
	Correct     bool
	Explanation string
}

// sentStats records stats sent to a chat.
type sentStats struct {
	ChatID    string
	Readiness float64
	Breakdown []core.TopicSummary
}

// sentNotification records a notification sent to a chat.
type sentNotification struct {
	ChatID       string
	Notification model.Notification
}

// MockMessenger is a thread-safe in-memory implementation of Messenger for testing.
type MockMessenger struct {
	mu            sync.Mutex
	questions     []sentQuestion
	feedbacks     []sentFeedback
	stats         []sentStats
	notifications []sentNotification
	msgCounter    int

	AnswerCh  chan core.AnswerEvent
	CommandCh chan core.CommandEvent
}

// NewMockMessenger creates a ready-to-use MockMessenger with buffered channels.
func NewMockMessenger() *MockMessenger {
	return &MockMessenger{
		AnswerCh:  make(chan core.AnswerEvent, 64),
		CommandCh: make(chan core.CommandEvent, 64),
	}
}

// SendQuestion records the question and returns a synthetic message ID.
func (m *MockMessenger) SendQuestion(chatID string, question model.Question) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgCounter++
	m.questions = append(m.questions, sentQuestion{ChatID: chatID, Question: question})
	return fmt.Sprintf("msg-%d", m.msgCounter), nil
}

// SendFeedback records the feedback.
func (m *MockMessenger) SendFeedback(chatID string, correct bool, explanation string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.feedbacks = append(m.feedbacks, sentFeedback{ChatID: chatID, Correct: correct, Explanation: explanation})
	return nil
}

// SendStats records the stats.
func (m *MockMessenger) SendStats(chatID string, readiness float64, breakdown []core.TopicSummary) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stats = append(m.stats, sentStats{ChatID: chatID, Readiness: readiness, Breakdown: breakdown})
	return nil
}

// SendNotification records the notification.
func (m *MockMessenger) SendNotification(chatID string, notification model.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = append(m.notifications, sentNotification{ChatID: chatID, Notification: notification})
	return nil
}

// ListenForAnswers returns the answer event channel.
func (m *MockMessenger) ListenForAnswers() <-chan core.AnswerEvent {
	return m.AnswerCh
}

// ListenForCommands returns the command event channel.
func (m *MockMessenger) ListenForCommands() <-chan core.CommandEvent {
	return m.CommandCh
}

// GetSentQuestions returns a snapshot of all questions sent so far.
func (m *MockMessenger) GetSentQuestions() []sentQuestion {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]sentQuestion, len(m.questions))
	copy(out, m.questions)
	return out
}

// GetSentNotifications returns a snapshot of all notifications sent so far.
func (m *MockMessenger) GetSentNotifications() []sentNotification {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]sentNotification, len(m.notifications))
	copy(out, m.notifications)
	return out
}
