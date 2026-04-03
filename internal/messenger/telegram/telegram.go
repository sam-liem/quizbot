package telegram

import (
	"context"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sam-liem/quizbot/internal/core"
	"github.com/sam-liem/quizbot/internal/model"
)

// TelegramMessenger implements messenger.Messenger (and core.SchedulerMessenger)
// using the Telegram Bot API.
type TelegramMessenger struct {
	bot       *tgbotapi.BotAPI
	answerCh  chan core.AnswerEvent
	commandCh chan core.CommandEvent
	stopCh    chan struct{}
}

// NewTelegramMessenger creates a TelegramMessenger authenticated with the given bot token.
func NewTelegramMessenger(token string) (*TelegramMessenger, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("creating telegram bot: %w", err)
	}
	return &TelegramMessenger{
		bot:       bot,
		answerCh:  make(chan core.AnswerEvent, 64),
		commandCh: make(chan core.CommandEvent, 64),
		stopCh:    make(chan struct{}),
	}, nil
}

// SendQuestion sends a question with an inline A/B/C/D keyboard to the given chat.
// It returns the sent message ID as a string.
func (t *TelegramMessenger) SendQuestion(chatID string, question model.Question) (string, error) {
	cid, err := parseChatID(chatID)
	if err != nil {
		return "", fmt.Errorf("sending question: %w", err)
	}

	text := FormatQuestionText(1, 1, question)
	kb := BuildQuestionKeyboard(question.Choices)

	msg := tgbotapi.NewMessage(cid, text)
	msg.ReplyMarkup = kb

	sent, err := t.bot.Send(msg)
	if err != nil {
		return "", fmt.Errorf("sending question to chat %s: %w", chatID, err)
	}
	return strconv.Itoa(sent.MessageID), nil
}

// SendFeedback sends formatted feedback to the given chat.
func (t *TelegramMessenger) SendFeedback(chatID string, correct bool, explanation string) error {
	cid, err := parseChatID(chatID)
	if err != nil {
		return fmt.Errorf("sending feedback: %w", err)
	}

	text := FormatFeedback(correct, explanation)
	msg := tgbotapi.NewMessage(cid, text)
	_, err = t.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("sending feedback to chat %s: %w", chatID, err)
	}
	return nil
}

// SendStats sends a formatted stats summary to the given chat.
func (t *TelegramMessenger) SendStats(chatID string, readiness float64, breakdown []core.TopicSummary) error {
	cid, err := parseChatID(chatID)
	if err != nil {
		return fmt.Errorf("sending stats: %w", err)
	}

	text := FormatStats(readiness, breakdown)
	msg := tgbotapi.NewMessage(cid, text)
	_, err = t.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("sending stats to chat %s: %w", chatID, err)
	}
	return nil
}

// SendNotification sends a formatted notification to the given chat.
func (t *TelegramMessenger) SendNotification(chatID string, notification model.Notification) error {
	cid, err := parseChatID(chatID)
	if err != nil {
		return fmt.Errorf("sending notification: %w", err)
	}

	text := FormatNotification(notification)
	msg := tgbotapi.NewMessage(cid, text)
	_, err = t.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("sending notification to chat %s: %w", chatID, err)
	}
	return nil
}

// ListenForAnswers returns the channel on which answer events are dispatched.
func (t *TelegramMessenger) ListenForAnswers() <-chan core.AnswerEvent {
	return t.answerCh
}

// ListenForCommands returns the channel on which command events are dispatched.
func (t *TelegramMessenger) ListenForCommands() <-chan core.CommandEvent {
	return t.commandCh
}

// StartListening starts a goroutine that receives Telegram updates and dispatches
// callback queries to answerCh and text commands to commandCh.
// The goroutine exits when ctx is cancelled or StopListening is called.
func (t *TelegramMessenger) StartListening(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := t.bot.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case <-ctx.Done():
				t.bot.StopReceivingUpdates()
				return
			case <-t.stopCh:
				t.bot.StopReceivingUpdates()
				return
			case update, ok := <-updates:
				if !ok {
					return
				}
				t.handleUpdate(update)
			}
		}
	}()
}

// StopListening signals the update loop to stop.
func (t *TelegramMessenger) StopListening() {
	select {
	case t.stopCh <- struct{}{}:
	default:
	}
}

// handleUpdate routes a single Telegram update to the appropriate channel.
func (t *TelegramMessenger) handleUpdate(update tgbotapi.Update) {
	switch {
	case update.CallbackQuery != nil:
		t.handleCallbackQuery(update.CallbackQuery)
	case update.Message != nil && update.Message.IsCommand():
		t.handleCommand(update.Message)
	}
}

// handleCallbackQuery dispatches an inline button press as an AnswerEvent.
func (t *TelegramMessenger) handleCallbackQuery(cq *tgbotapi.CallbackQuery) {
	answerIndex, err := ParseCallbackData(cq.Data)
	if err != nil {
		return
	}

	chatID := strconv.FormatInt(cq.Message.Chat.ID, 10)
	messageID := strconv.Itoa(cq.Message.MessageID)

	event := core.AnswerEvent{
		ChatID:      chatID,
		MessageID:   messageID,
		QuestionID:  "", // populated by quiz engine when correlating message IDs
		AnswerIndex: answerIndex,
	}

	select {
	case t.answerCh <- event:
	default:
		// Channel full — drop the event to avoid blocking.
	}

	// Acknowledge the callback query to remove the loading spinner.
	callback := tgbotapi.NewCallback(cq.ID, "")
	_, _ = t.bot.Request(callback)
}

// handleCommand dispatches a text command as a CommandEvent.
func (t *TelegramMessenger) handleCommand(msg *tgbotapi.Message) {
	cmd, args := ParseCommand(msg.Text)
	if cmd == "" {
		return
	}

	chatID := strconv.FormatInt(msg.Chat.ID, 10)
	event := core.CommandEvent{
		ChatID:  chatID,
		Command: cmd,
		Args:    args,
	}

	select {
	case t.commandCh <- event:
	default:
		// Channel full — drop the event to avoid blocking.
	}
}

// parseChatID converts a string chat ID to an int64 as required by tgbotapi.
func parseChatID(chatID string) (int64, error) {
	cid, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid chat ID %q: %w", chatID, err)
	}
	return cid, nil
}
