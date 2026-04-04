package core_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sam-liem/quizbot/internal/core"
	"github.com/sam-liem/quizbot/internal/model"
	"github.com/sam-liem/quizbot/internal/store"
)

// ---------------------------------------------------------------------------
// Local mock messenger — avoids import cycle with internal/messenger
// ---------------------------------------------------------------------------

type mockMessenger struct {
	mu        sync.Mutex
	questions []model.Question
	msgCounter int
}

func (m *mockMessenger) SendQuestion(_ string, q model.Question) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgCounter++
	m.questions = append(m.questions, q)
	return "msg-id", nil
}

func (m *mockMessenger) SendNotification(_ string, _ model.Notification) error {
	return nil
}

func (m *mockMessenger) sentCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.questions)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// newTestRepo builds a repo with one pack containing nQuestions questions, and
// sets user preferences.
func newTestRepo(t *testing.T, userID string, prefs model.UserPreferences, nQuestions int) *store.MockRepository {
	t.Helper()
	repo := store.NewMockRepository()

	questions := make([]model.Question, nQuestions)
	for i := 0; i < nQuestions; i++ {
		questions[i] = model.Question{
			ID:           fmt.Sprintf("q%d", i+1),
			TopicID:      "topic1",
			Text:         fmt.Sprintf("Question %d", i+1),
			Choices:      []string{"A", "B", "C", "D"},
			CorrectIndex: 0,
		}
	}

	pack := model.QuizPack{
		ID:        "pack1",
		Name:      "Test Pack",
		Questions: questions,
	}
	if err := repo.SaveQuizPack(context.Background(), pack); err != nil {
		t.Fatalf("SaveQuizPack: %v", err)
	}

	prefs.UserID = userID
	prefs.ActivePackIDs = []string{"pack1"}
	if err := repo.UpdatePreferences(context.Background(), prefs); err != nil {
		t.Fatalf("UpdatePreferences: %v", err)
	}

	return repo
}

func defaultPrefs() model.UserPreferences {
	p := model.DefaultPreferences("user1")
	p.DeliveryIntervalMin = 0 // no throttle in tests
	p.MaxUnanswered = 10
	p.QuietHoursStart = ""
	p.QuietHoursEnd = ""
	return p
}

func defaultConfig(userID string) core.SchedulerConfig {
	return core.SchedulerConfig{
		UserID:       userID,
		ChatID:       "chat1",
		TickInterval: 20 * time.Millisecond,
		SkipTimeout:  500 * time.Millisecond, // generous default
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestScheduler_StartAndStop(t *testing.T) {
	t.Parallel()

	prefs := defaultPrefs()
	repo := newTestRepo(t, "user1", prefs, 5)
	msng := &mockMessenger{}
	notifier := core.NewNotifier(repo)

	cfg := defaultConfig("user1")
	s := core.NewScheduler(core.NewQuizEngine(repo), msng, notifier, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	s.Start(ctx)

	// Wait long enough for at least one tick.
	time.Sleep(150 * time.Millisecond)

	s.Stop()

	if msng.sentCount() < 1 {
		t.Errorf("expected at least 1 question sent, got %d", msng.sentCount())
	}
}

func TestScheduler_PausesAtMaxUnanswered(t *testing.T) {
	t.Parallel()

	prefs := defaultPrefs()
	prefs.MaxUnanswered = 2
	repo := newTestRepo(t, "user2", prefs, 10)
	msng := &mockMessenger{}
	notifier := core.NewNotifier(repo)

	cfg := defaultConfig("user2")
	cfg.TickInterval = 10 * time.Millisecond

	s := core.NewScheduler(core.NewQuizEngine(repo), msng, notifier, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()

	s.Start(ctx)
	time.Sleep(250 * time.Millisecond)
	s.Stop()

	sent := msng.sentCount()
	if sent > prefs.MaxUnanswered {
		t.Errorf("expected at most %d questions sent (max_unanswered), got %d", prefs.MaxUnanswered, sent)
	}
}

func TestScheduler_ResumesAfterAnswer(t *testing.T) {
	t.Parallel()

	prefs := defaultPrefs()
	prefs.MaxUnanswered = 1
	repo := newTestRepo(t, "user3", prefs, 10)
	msng := &mockMessenger{}
	notifier := core.NewNotifier(repo)

	cfg := defaultConfig("user3")
	cfg.TickInterval = 20 * time.Millisecond

	s := core.NewScheduler(core.NewQuizEngine(repo), msng, notifier, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()

	s.Start(ctx)

	// Allow first question to be sent.
	time.Sleep(80 * time.Millisecond)
	countBefore := msng.sentCount()
	if countBefore < 1 {
		t.Fatalf("no questions sent before answering")
	}

	// Record answer for the first sent question.
	msng.mu.Lock()
	firstQ := msng.questions[0]
	msng.mu.Unlock()

	s.RecordAnswer(firstQ.ID)

	// Allow more questions to be sent after the answer.
	time.Sleep(150 * time.Millisecond)
	s.Stop()

	countAfter := msng.sentCount()
	if countAfter <= countBefore {
		t.Errorf("expected more questions sent after answer; before=%d after=%d", countBefore, countAfter)
	}
}

func TestScheduler_SkipsAfterTimeout(t *testing.T) {
	t.Parallel()

	prefs := defaultPrefs()
	prefs.MaxUnanswered = 1
	repo := newTestRepo(t, "user4", prefs, 10)
	msng := &mockMessenger{}
	notifier := core.NewNotifier(repo)

	cfg := defaultConfig("user4")
	cfg.TickInterval = 20 * time.Millisecond
	cfg.SkipTimeout = 80 * time.Millisecond // short timeout so questions get skipped

	s := core.NewScheduler(core.NewQuizEngine(repo), msng, notifier, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Millisecond)
	defer cancel()

	s.Start(ctx)

	// Wait long enough for at least two distinct questions to have been sent
	// (first one times out, second is issued).
	time.Sleep(400 * time.Millisecond)
	s.Stop()

	sent := msng.sentCount()
	if sent < 2 {
		t.Errorf("expected at least 2 questions sent (skip triggered new send), got %d", sent)
	}
}

func TestScheduler_ContextCancellation(t *testing.T) {
	t.Parallel()

	prefs := defaultPrefs()
	repo := newTestRepo(t, "user5", prefs, 5)
	msng := &mockMessenger{}
	notifier := core.NewNotifier(repo)

	cfg := defaultConfig("user5")

	s := core.NewScheduler(core.NewQuizEngine(repo), msng, notifier, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	time.Sleep(30 * time.Millisecond)
	cancel() // cancel context directly

	// Stop should return without hanging.
	done := make(chan struct{})
	go func() {
		s.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return after context cancellation")
	}
}
