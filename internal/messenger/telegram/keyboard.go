package telegram

import (
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sam-liem/quizbot/internal/core"
	"github.com/sam-liem/quizbot/internal/model"
)

// BuildQuestionKeyboard creates an inline keyboard with one row of A/B/C/D buttons.
// Each button's callback data is the choice index as a string ("0", "1", "2", "3").
func BuildQuestionKeyboard(choices []string) tgbotapi.InlineKeyboardMarkup {
	labels := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	buttons := make([]tgbotapi.InlineKeyboardButton, len(choices))
	for i := range choices {
		label := strconv.Itoa(i + 1)
		if i < len(labels) {
			label = labels[i]
		}
		data := strconv.Itoa(i)
		buttons[i] = tgbotapi.InlineKeyboardButton{
			Text:         label,
			CallbackData: &data,
		}
	}
	return tgbotapi.NewInlineKeyboardMarkup(buttons)
}

// FormatQuestionText formats a question for display.
// Output format: "Q{index}/{total}: {question text}\n\nA) choice0\nB) choice1\n..."
func FormatQuestionText(index, total int, question model.Question) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Q%d/%d: %s", index, total, question.Text)
	labels := []string{"A", "B", "C", "D"}
	for i, choice := range question.Choices {
		label := strconv.Itoa(i)
		if i < len(labels) {
			label = labels[i]
		}
		fmt.Fprintf(&sb, "\n%s) %s", label, choice)
	}
	return sb.String()
}

// FormatFeedback formats answer feedback.
// Correct: "✓ Correct!\n{explanation}"
// Wrong:   "✗ Wrong.\n{explanation}"
func FormatFeedback(correct bool, explanation string) string {
	var sb strings.Builder
	if correct {
		sb.WriteString("✓ Correct!")
	} else {
		sb.WriteString("✗ Wrong.")
	}
	if explanation != "" {
		sb.WriteString("\n")
		sb.WriteString(explanation)
	}
	return sb.String()
}

// FormatStats formats a readiness and topic breakdown summary.
func FormatStats(readiness float64, breakdown []core.TopicSummary) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Readiness: %.0f%%\n", readiness*100)
	if len(breakdown) > 0 {
		sb.WriteString("\nTopic Breakdown:\n")
		for _, ts := range breakdown {
			fmt.Fprintf(&sb, "  %s: %.0f%% (%d/%d)\n",
				ts.TopicID,
				ts.Accuracy*100,
				ts.CorrectCount,
				ts.TotalAttempts,
			)
		}
	}
	return sb.String()
}

// FormatNotification formats a notification for display.
func FormatNotification(notification model.Notification) string {
	return notification.Message
}

// ParseCallbackData parses a button callback data string to an answer index.
func ParseCallbackData(data string) (int, error) {
	idx, err := strconv.Atoi(data)
	if err != nil {
		return 0, fmt.Errorf("parsing callback data %q: %w", data, err)
	}
	return idx, nil
}

// ParseCommand parses a Telegram command string (e.g. "/quiz --mock") into a
// command name (without the leading '/') and a slice of arguments.
func ParseCommand(text string) (command string, args []string) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "", nil
	}
	cmd := parts[0]
	cmd = strings.TrimPrefix(cmd, "/")
	// Strip the @BotName suffix if present.
	if idx := strings.Index(cmd, "@"); idx != -1 {
		cmd = cmd[:idx]
	}
	if len(parts) > 1 {
		return cmd, parts[1:]
	}
	return cmd, []string{}
}
