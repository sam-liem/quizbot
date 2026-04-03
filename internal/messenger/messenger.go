package messenger

import (
	"github.com/sam-liem/quizbot/internal/core"
	"github.com/sam-liem/quizbot/internal/model"
)

// Messenger is the full interface for sending and receiving messages.
type Messenger interface {
	SendQuestion(chatID string, question model.Question) (string, error)
	SendFeedback(chatID string, correct bool, explanation string) error
	SendStats(chatID string, readiness float64, breakdown []core.TopicSummary) error
	SendNotification(chatID string, notification model.Notification) error
	ListenForAnswers() <-chan core.AnswerEvent
	ListenForCommands() <-chan core.CommandEvent
}
