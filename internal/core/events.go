package core

// AnswerEvent is emitted by the messenger when a user answers a question.
type AnswerEvent struct {
	ChatID     string
	MessageID  string
	QuestionID string
	AnswerIndex int
}

// CommandEvent is emitted by the messenger when a user issues a command.
type CommandEvent struct {
	ChatID  string
	Command string
	Args    []string
}
