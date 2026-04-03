package core

import (
	"testing"
	"time"

	"github.com/sam-liem/quizbot/internal/model"
	"github.com/stretchr/testify/assert"
)

var baseTime = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

// newState creates a QuestionState with default SM-2 initial values.
func newState() model.QuestionState {
	return model.QuestionState{
		UserID:          "user1",
		QuestionID:      "q1",
		PackID:          "pack1",
		EaseFactor:      2.5,
		IntervalDays:    0,
		RepetitionCount: 0,
		NextReviewAt:    baseTime,
		LastResult:      "",
		LastReviewedAt:  time.Time{},
	}
}

// ---- CalculateNextReview tests ----

func TestCalculateNextReview_FirstCorrect(t *testing.T) {
	state := newState()
	result := CalculateNextReview(state, true, baseTime)

	assert.Equal(t, 1, result.RepetitionCount, "rep count should be 1")
	assert.InDelta(t, 1.0, result.IntervalDays, 0.001, "first correct interval should be 1 day")
	assert.InDelta(t, 2.5, result.EaseFactor, 0.001, "ease should be unchanged at 2.5 after first rep (no increase on rep 1)")
	assert.Equal(t, model.AnswerResultCorrect, result.LastResult)
	assert.Equal(t, baseTime, result.LastReviewedAt)
	assert.Equal(t, baseTime.Add(24*time.Hour), result.NextReviewAt)
}

func TestCalculateNextReview_SecondCorrect(t *testing.T) {
	// State after first correct
	state := newState()
	state.RepetitionCount = 1
	state.IntervalDays = 1.0
	state.EaseFactor = 2.5

	result := CalculateNextReview(state, true, baseTime)

	assert.Equal(t, 2, result.RepetitionCount)
	assert.InDelta(t, 6.0, result.IntervalDays, 0.001, "second correct interval should be 6 days")
	assert.InDelta(t, 2.6, result.EaseFactor, 0.001, "ease should increase by 0.1 to 2.6")
	assert.Equal(t, model.AnswerResultCorrect, result.LastResult)
	assert.Equal(t, baseTime.Add(6*24*time.Hour), result.NextReviewAt)
}

func TestCalculateNextReview_ThirdCorrect(t *testing.T) {
	// State after second correct: rep=2, interval=6.0, ease=2.6
	state := newState()
	state.RepetitionCount = 2
	state.IntervalDays = 6.0
	state.EaseFactor = 2.6

	result := CalculateNextReview(state, true, baseTime)

	assert.Equal(t, 3, result.RepetitionCount)
	// interval = prev * ease = 6.0 * 2.6 = 15.6
	assert.InDelta(t, 15.6, result.IntervalDays, 0.001, "third correct interval should be prev*ease=15.6")
	assert.InDelta(t, 2.7, result.EaseFactor, 0.001, "ease should increase by 0.1 to 2.7")
	assert.Equal(t, model.AnswerResultCorrect, result.LastResult)
	expectedNext := baseTime.Add(time.Duration(15.6 * float64(24*time.Hour)))
	assert.Equal(t, expectedNext, result.NextReviewAt)
}

func TestCalculateNextReview_WrongAnswer(t *testing.T) {
	// State with some progress
	state := newState()
	state.RepetitionCount = 3
	state.IntervalDays = 15.6
	state.EaseFactor = 2.7

	result := CalculateNextReview(state, false, baseTime)

	assert.Equal(t, 0, result.RepetitionCount, "wrong answer resets rep count to 0")
	assert.InDelta(t, 1.0, result.IntervalDays, 0.001, "wrong answer resets interval to 1 day")
	// ease = 2.7 - 0.32 = 2.38
	assert.InDelta(t, 2.38, result.EaseFactor, 0.001, "ease should decrease by 0.32")
	assert.Equal(t, model.AnswerResultWrong, result.LastResult)
	assert.Equal(t, baseTime, result.LastReviewedAt)
	assert.Equal(t, baseTime.Add(24*time.Hour), result.NextReviewAt)
}

func TestCalculateNextReview_EaseFloorOnWrong(t *testing.T) {
	// Ease is already at minimum 1.3
	state := newState()
	state.EaseFactor = 1.3
	state.RepetitionCount = 2
	state.IntervalDays = 6.0

	result := CalculateNextReview(state, false, baseTime)

	assert.InDelta(t, 1.3, result.EaseFactor, 0.001, "ease should not go below 1.3")
}

func TestCalculateNextReview_EaseFloorApproach(t *testing.T) {
	// Ease would drop below 1.3: 1.5 - 0.32 = 1.18, should floor to 1.3
	state := newState()
	state.EaseFactor = 1.5
	state.RepetitionCount = 1
	state.IntervalDays = 1.0

	result := CalculateNextReview(state, false, baseTime)

	assert.InDelta(t, 1.3, result.EaseFactor, 0.001, "ease should floor at 1.3")
}

func TestCalculateNextReview_MultipleWrongKeepsFloor(t *testing.T) {
	state := newState()
	state.EaseFactor = 1.3

	// Apply wrong multiple times
	for i := 0; i < 5; i++ {
		state = CalculateNextReview(state, false, baseTime)
	}

	assert.InDelta(t, 1.3, state.EaseFactor, 0.001, "ease should stay at 1.3 after multiple wrong answers")
}

func TestCalculateNextReview_LastReviewedAtSet(t *testing.T) {
	state := newState()
	now := time.Date(2026, 3, 15, 9, 30, 0, 0, time.UTC)

	result := CalculateNextReview(state, true, now)

	assert.Equal(t, now, result.LastReviewedAt)
}

// ---- SelectNextQuestion tests ----

func TestSelectNextQuestion_ReturnsNilForEmpty(t *testing.T) {
	result := SelectNextQuestion([]model.QuestionState{}, baseTime)
	assert.Nil(t, result)
}

func TestSelectNextQuestion_ReturnsNilWhenNothingDue(t *testing.T) {
	states := []model.QuestionState{
		{QuestionID: "q1", NextReviewAt: baseTime.Add(1 * time.Hour)},
		{QuestionID: "q2", NextReviewAt: baseTime.Add(2 * time.Hour)},
	}
	result := SelectNextQuestion(states, baseTime)
	assert.Nil(t, result)
}

func TestSelectNextQuestion_ReturnsDueQuestion(t *testing.T) {
	states := []model.QuestionState{
		{QuestionID: "q1", NextReviewAt: baseTime.Add(-1 * time.Hour)},  // due
		{QuestionID: "q2", NextReviewAt: baseTime.Add(10 * time.Hour)},  // not due
	}
	result := SelectNextQuestion(states, baseTime)
	assert.NotNil(t, result)
	assert.Equal(t, "q1", result.QuestionID)
}

func TestSelectNextQuestion_ReturnsMostOverdue(t *testing.T) {
	states := []model.QuestionState{
		{QuestionID: "q1", NextReviewAt: baseTime.Add(-1 * time.Hour)},   // due, 1h overdue
		{QuestionID: "q2", NextReviewAt: baseTime.Add(-5 * time.Hour)},   // due, 5h overdue (most overdue)
		{QuestionID: "q3", NextReviewAt: baseTime.Add(-30 * time.Minute)}, // due, 30m overdue
	}
	result := SelectNextQuestion(states, baseTime)
	assert.NotNil(t, result)
	assert.Equal(t, "q2", result.QuestionID, "should pick the most overdue (earliest NextReviewAt)")
}

func TestSelectNextQuestion_AllOverduePicsEarliest(t *testing.T) {
	t1 := baseTime.Add(-24 * time.Hour)
	t2 := baseTime.Add(-48 * time.Hour) // earliest = most overdue
	t3 := baseTime.Add(-12 * time.Hour)

	states := []model.QuestionState{
		{QuestionID: "q1", NextReviewAt: t1},
		{QuestionID: "q2", NextReviewAt: t2},
		{QuestionID: "q3", NextReviewAt: t3},
	}
	result := SelectNextQuestion(states, baseTime)
	assert.NotNil(t, result)
	assert.Equal(t, "q2", result.QuestionID)
}

func TestSelectNextQuestion_ExactlyDue(t *testing.T) {
	// NextReviewAt == now should be considered due
	states := []model.QuestionState{
		{QuestionID: "q1", NextReviewAt: baseTime},
	}
	result := SelectNextQuestion(states, baseTime)
	assert.NotNil(t, result)
	assert.Equal(t, "q1", result.QuestionID)
}
