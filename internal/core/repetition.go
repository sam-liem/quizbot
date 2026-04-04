package core

import (
	"time"

	"github.com/sam-liem/quizbot/internal/model"
)

const (
	easeFactor      = 1.3  // minimum ease factor (floor)
	easeIncrement   = 0.1  // ease increase on correct
	easeDecrement   = 0.32 // ease decrease on wrong
	intervalFirst   = 1.0  // days after rep 1
	intervalSecond  = 6.0  // days after rep 2
)

// CalculateNextReview applies the SM-2 spaced repetition algorithm to the
// given QuestionState and returns the updated state.
//
// Correct answer:
//   - RepetitionCount increments by 1.
//   - IntervalDays: 1 day (rep 1), 6 days (rep 2), then previous * EaseFactor.
//   - EaseFactor increases by 0.1 (applied after interval is computed, from rep 2 onwards).
//   - LastResult = correct, LastReviewedAt = now.
//   - NextReviewAt = now + interval.
//
// Wrong answer:
//   - RepetitionCount resets to 0.
//   - IntervalDays resets to 1.0.
//   - EaseFactor decreases by 0.32, floored at 1.3.
//   - LastResult = wrong, LastReviewedAt = now.
//   - NextReviewAt = now + 1 day.
func CalculateNextReview(state model.QuestionState, correct bool, now time.Time) model.QuestionState {
	next := state

	if correct {
		next.RepetitionCount = state.RepetitionCount + 1

		switch next.RepetitionCount {
		case 1:
			next.IntervalDays = intervalFirst
			// Ease unchanged on rep 1
		case 2:
			next.IntervalDays = intervalSecond
			next.EaseFactor = state.EaseFactor + easeIncrement
		default:
			// Use previous ease factor to compute interval, then increment ease
			next.IntervalDays = state.IntervalDays * state.EaseFactor
			next.EaseFactor = state.EaseFactor + easeIncrement
		}

		next.LastResult = model.AnswerResultCorrect
	} else {
		next.RepetitionCount = 0
		next.IntervalDays = intervalFirst

		newEase := state.EaseFactor - easeDecrement
		if newEase < easeFactor {
			newEase = easeFactor
		}
		next.EaseFactor = newEase

		next.LastResult = model.AnswerResultWrong
	}

	next.LastReviewedAt = now
	next.NextReviewAt = now.Add(time.Duration(next.IntervalDays * float64(24*time.Hour)))

	return next
}

// SelectNextQuestion returns a pointer to the most overdue QuestionState (i.e.
// the one with the earliest NextReviewAt that is at or before now).
// Returns nil if no questions are due or the slice is empty.
func SelectNextQuestion(states []model.QuestionState, now time.Time) *model.QuestionState {
	var selected *model.QuestionState

	for i := range states {
		s := &states[i]
		if s.NextReviewAt.After(now) {
			continue // not due yet
		}
		if selected == nil || s.NextReviewAt.Before(selected.NextReviewAt) {
			selected = s
		}
	}

	return selected
}
