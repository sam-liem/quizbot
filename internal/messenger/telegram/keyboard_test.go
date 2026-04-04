package telegram_test

import (
	"strings"
	"testing"
	"time"

	"github.com/sam-liem/quizbot/internal/core"
	"github.com/sam-liem/quizbot/internal/messenger/telegram"
	"github.com/sam-liem/quizbot/internal/model"
)

func TestBuildQuestionKeyboard(t *testing.T) {
	choices := []string{"Option A", "Option B", "Option C", "Option D"}
	kb := telegram.BuildQuestionKeyboard(choices)

	// Should have one row with 4 buttons.
	if len(kb.InlineKeyboard) != 1 {
		t.Fatalf("expected 1 row, got %d", len(kb.InlineKeyboard))
	}
	row := kb.InlineKeyboard[0]
	if len(row) != 4 {
		t.Fatalf("expected 4 buttons, got %d", len(row))
	}

	labels := []string{"A", "B", "C", "D"}
	callbackData := []string{"0", "1", "2", "3"}
	for i, btn := range row {
		if btn.Text != labels[i] {
			t.Errorf("button %d: expected label %q, got %q", i, labels[i], btn.Text)
		}
		if btn.CallbackData == nil || *btn.CallbackData != callbackData[i] {
			var got string
			if btn.CallbackData != nil {
				got = *btn.CallbackData
			}
			t.Errorf("button %d: expected callback data %q, got %q", i, callbackData[i], got)
		}
	}
}

func TestFormatQuestionText(t *testing.T) {
	q := model.Question{
		ID:   "q1",
		Text: "What is the capital of the UK?",
		Choices: []string{
			"London",
			"Manchester",
			"Edinburgh",
			"Cardiff",
		},
		CorrectIndex: 0,
	}
	got := telegram.FormatQuestionText(1, 3, q)

	if !strings.Contains(got, "Q1/3") {
		t.Errorf("expected Q1/3 in output, got: %s", got)
	}
	if !strings.Contains(got, q.Text) {
		t.Errorf("expected question text in output, got: %s", got)
	}
}

func TestFormatFeedback_Correct(t *testing.T) {
	got := telegram.FormatFeedback(true, "Well done!")
	if !strings.Contains(got, "Correct") {
		t.Errorf("expected 'Correct' in output, got: %s", got)
	}
}

func TestFormatFeedback_Wrong(t *testing.T) {
	explanation := "The capital is London."
	got := telegram.FormatFeedback(false, explanation)
	if !strings.Contains(got, "Wrong") {
		t.Errorf("expected 'Wrong' in output, got: %s", got)
	}
	if !strings.Contains(got, explanation) {
		t.Errorf("expected explanation in output, got: %s", got)
	}
}

func TestFormatStats(t *testing.T) {
	readiness := 0.75
	breakdown := []core.TopicSummary{
		{TopicID: "topic-1", Accuracy: 0.6, TotalAttempts: 10, CorrectCount: 6},
		{TopicID: "topic-2", Accuracy: 0.9, TotalAttempts: 20, CorrectCount: 18},
	}
	got := telegram.FormatStats(readiness, breakdown)
	if !strings.Contains(got, "75") {
		t.Errorf("expected readiness percentage (75) in output, got: %s", got)
	}
}

func TestFormatNotification(t *testing.T) {
	n := model.Notification{
		Type:    model.NotificationInactivity,
		Message: "You haven't practiced in 3 days!",
		SentAt:  time.Now(),
	}
	got := telegram.FormatNotification(n)
	if !strings.Contains(got, n.Message) {
		t.Errorf("expected notification message in output, got: %s", got)
	}
}

func TestParseCallbackData(t *testing.T) {
	idx, err := telegram.ParseCallbackData("2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 2 {
		t.Errorf("expected 2, got %d", idx)
	}
}

func TestParseCallbackData_Invalid(t *testing.T) {
	_, err := telegram.ParseCallbackData("invalid")
	if err == nil {
		t.Error("expected error for invalid callback data, got nil")
	}
}

func TestParseCommand(t *testing.T) {
	cmd, args := telegram.ParseCommand("/quiz --mock")
	if cmd != "quiz" {
		t.Errorf("expected command 'quiz', got %q", cmd)
	}
	if len(args) != 1 || args[0] != "--mock" {
		t.Errorf("expected args [--mock], got %v", args)
	}
}

func TestParseCommand_NoArgs(t *testing.T) {
	cmd, args := telegram.ParseCommand("/start")
	if cmd != "start" {
		t.Errorf("expected command 'start', got %q", cmd)
	}
	if len(args) != 0 {
		t.Errorf("expected no args, got %v", args)
	}
}
