package model

import "time"

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
