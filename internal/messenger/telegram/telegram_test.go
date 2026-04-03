package telegram_test

import (
	"github.com/sam-liem/quizbot/internal/messenger"
	"github.com/sam-liem/quizbot/internal/messenger/telegram"
)

// Compile-time interface check: TelegramMessenger must implement messenger.Messenger.
var _ messenger.Messenger = (*telegram.TelegramMessenger)(nil)
