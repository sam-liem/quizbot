package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sam-liem/quizbot/config"
	"github.com/sam-liem/quizbot/internal/cli"
	"github.com/sam-liem/quizbot/internal/core"
	"github.com/sam-liem/quizbot/internal/messenger/telegram"
	"github.com/sam-liem/quizbot/internal/store/sqlite"
)

const defaultUserID = "default"

func main() {
	// 1. Load config.
	cfgPath := os.Getenv("QUIZBOT_CONFIG")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		slog.Error("loading config", "error", err)
		os.Exit(1)
	}

	// 2. Set up structured logger based on config log level.
	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	// 3. Open SQLite database.
	db, err := sqlite.Open(cfg.SQLitePath)
	if err != nil {
		slog.Error("opening database", "path", cfg.SQLitePath, "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("closing database", "error", err)
		}
	}()

	// 4. Create core components.
	engine := core.NewQuizEngine(db)
	notifier := core.NewNotifier(db)

	// 5. Create TelegramMessenger.
	tgMessenger, err := telegram.NewTelegramMessenger(cfg.TelegramBotToken)
	if err != nil {
		slog.Error("creating telegram messenger", "error", err)
		os.Exit(1)
	}

	// 6. Create Scheduler.
	schedulerCfg := core.SchedulerConfig{
		UserID:       defaultUserID,
		ChatID:       "",
		TickInterval: time.Minute,
		SkipTimeout:  24 * time.Hour,
	}
	scheduler := core.NewScheduler(engine, tgMessenger, notifier, schedulerCfg)

	// 7. Create CLI App.
	app := cli.NewApp(db, defaultUserID)

	// 8. Set up graceful shutdown: listen for SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 9. Start Telegram bot listening in a goroutine.
	tgMessenger.StartListening(ctx)

	// 10. Start scheduler in background.
	scheduler.Start(ctx)

	// 11. Execute cobra CLI command.
	rootCmd := cli.NewRootCommand(app)
	if err := rootCmd.Execute(); err != nil {
		slog.Error("command failed", "error", err)
	}

	// Wait for shutdown signal if the CLI command returns (e.g. running as a daemon).
	<-ctx.Done()

	// 12. On shutdown signal: stop scheduler, stop telegram.
	slog.Info("shutting down")
	scheduler.Stop()
	tgMessenger.StopListening()
	slog.Info("shutdown complete")
}
