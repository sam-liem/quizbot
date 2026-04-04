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

// QuestionHistoryFilter controls which question states are returned and how they
// are ordered by ListQuestionStates.
//
//   - PackID is required.
//   - TopicID filtering is applied in Go after loading pack data (question_states
//     has no topic_id column in v1).
//   - Result filters on the last_result column (empty string = all results).
//   - SortBy accepts "next_review", "last_reviewed", or "ease_factor";
//     defaults to "last_reviewed" if empty.
//   - SortDesc reverses the sort order.
type QuestionHistoryFilter struct {
	PackID   string       // required
	TopicID  string       // optional; matched in Go against pack question list
	Result   AnswerResult // optional; empty = all results
	SortBy   string       // "next_review", "last_reviewed", "ease_factor"
	SortDesc bool
}
