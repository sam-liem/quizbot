package cli

import (
	"context"
	"fmt"

	"github.com/sam-liem/quizbot/internal/core"
	"github.com/sam-liem/quizbot/internal/model"
	"github.com/sam-liem/quizbot/internal/store"
)

// App holds the dependencies for all CLI commands.
type App struct {
	repo   store.Repository
	engine *core.QuizEngine
	userID string
}

// NewApp creates a new App with the given repository and user ID.
func NewApp(repo store.Repository, userID string) *App {
	return &App{
		repo:   repo,
		engine: core.NewQuizEngine(repo),
		userID: userID,
	}
}

// getPreferences loads the user's preferences from the repository.
func (a *App) getPreferences(ctx context.Context) (*model.UserPreferences, error) {
	prefs, err := a.repo.GetPreferences(ctx, a.userID)
	if err != nil {
		return nil, fmt.Errorf("loading preferences: %w", err)
	}
	return prefs, nil
}

// savePreferences persists the user's preferences to the repository.
func (a *App) savePreferences(ctx context.Context, prefs model.UserPreferences) error {
	if err := a.repo.UpdatePreferences(ctx, prefs); err != nil {
		return fmt.Errorf("saving preferences: %w", err)
	}
	return nil
}
