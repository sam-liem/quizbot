package model

type SessionMode string

const (
	SessionModeMock      SessionMode = "mock"
	SessionModePractice  SessionMode = "practice"
	SessionModeScheduled SessionMode = "scheduled"
)

type QuizSessionStatus string

const (
	QuizSessionStatusInProgress QuizSessionStatus = "in_progress"
	QuizSessionStatusCompleted  QuizSessionStatus = "completed"
	QuizSessionStatusAbandoned  QuizSessionStatus = "abandoned"
)

type AnswerResult string

const (
	AnswerResultCorrect AnswerResult = "correct"
	AnswerResultWrong   AnswerResult = "wrong"
	AnswerResultSkipped AnswerResult = "skipped"
)

type FocusMode string

const (
	FocusModeSingle      FocusMode = "single"
	FocusModeInterleaved FocusMode = "interleaved"
)
