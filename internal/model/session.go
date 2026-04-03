package model

import "time"

type QuestionAttempt struct {
	QuestionID  string    `json:"question_id"`
	AnswerIndex int       `json:"answer_index"`
	Correct     bool      `json:"correct"`
	TimeTakenMs int       `json:"time_taken_ms"`
	AnsweredAt  time.Time `json:"answered_at"`
}

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
