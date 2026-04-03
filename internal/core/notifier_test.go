package core_test

import (
	"context"
	"testing"
	"time"

	"github.com/sam-liem/quizbot/internal/core"
	"github.com/sam-liem/quizbot/internal/model"
	"github.com/sam-liem/quizbot/internal/store"
)

const notifierTestUserID = "user-1"

// activePrefs returns a fully-enabled UserPreferences with no quiet hours.
func activePrefs(userID string) model.UserPreferences {
	return model.UserPreferences{
		UserID:               userID,
		ActivePackIDs:        []string{"pack-1"},
		NotifyInactivity:     true,
		NotifyInactivityDays: 3,
		NotifyWeakTopic:      true,
		NotifyWeakTopicPct:   50.0,
		NotifyMilestones:     true,
		NotifyReadiness:      true,
		NotifyStreak:         true,
		QuietHoursStart:      "",
		QuietHoursEnd:        "",
	}
}

// seedPack stores a minimal quiz pack in the repo.
func seedPack(t *testing.T, repo *store.MockRepository) {
	t.Helper()
	pack := model.QuizPack{
		ID:   "pack-1",
		Name: "Test Pack",
		Topics: []model.Topic{
			{ID: "topic-1", Name: "Topic One"},
		},
		Questions: []model.Question{
			{ID: "q-1", TopicID: "topic-1"},
			{ID: "q-2", TopicID: "topic-1"},
		},
	}
	if err := repo.SaveQuizPack(context.Background(), pack); err != nil {
		t.Fatalf("seedPack: %v", err)
	}
}

// TestCheckNotifications_InactivityReminder verifies that when no review
// has happened within NotifyInactivityDays, an inactivity notification is returned.
func TestCheckNotifications_InactivityReminder(t *testing.T) {
	repo := store.NewMockRepository()
	seedPack(t, repo)

	prefs := activePrefs(notifierTestUserID)
	prefs.NotifyWeakTopic = false
	prefs.NotifyMilestones = false
	prefs.NotifyReadiness = false
	prefs.NotifyStreak = false
	repo.Preferences[notifierTestUserID] = prefs

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	oldReview := now.AddDate(0, 0, -5) // 5 days ago, beyond the 3-day threshold

	if err := repo.UpdateQuestionState(context.Background(), model.QuestionState{
		UserID:         notifierTestUserID,
		PackID:         "pack-1",
		QuestionID:     "q-1",
		LastReviewedAt: oldReview,
	}); err != nil {
		t.Fatalf("UpdateQuestionState: %v", err)
	}

	notifier := core.NewNotifier(repo)
	notifications, err := notifier.CheckNotifications(context.Background(), notifierTestUserID, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasType(notifications, model.NotificationInactivity) {
		t.Errorf("expected inactivity notification, got %v", types(notifications))
	}
}

// TestCheckNotifications_InactivityDisabled verifies no inactivity notification
// when NotifyInactivity is false.
func TestCheckNotifications_InactivityDisabled(t *testing.T) {
	repo := store.NewMockRepository()
	seedPack(t, repo)

	prefs := activePrefs(notifierTestUserID)
	prefs.NotifyInactivity = false
	prefs.NotifyWeakTopic = false
	prefs.NotifyMilestones = false
	prefs.NotifyReadiness = false
	prefs.NotifyStreak = false
	repo.Preferences[notifierTestUserID] = prefs

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	oldReview := now.AddDate(0, 0, -5)

	if err := repo.UpdateQuestionState(context.Background(), model.QuestionState{
		UserID:         notifierTestUserID,
		PackID:         "pack-1",
		QuestionID:     "q-1",
		LastReviewedAt: oldReview,
	}); err != nil {
		t.Fatalf("UpdateQuestionState: %v", err)
	}

	notifier := core.NewNotifier(repo)
	notifications, err := notifier.CheckNotifications(context.Background(), notifierTestUserID, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hasType(notifications, model.NotificationInactivity) {
		t.Errorf("expected no inactivity notification, got one")
	}
}

// TestCheckNotifications_WeakTopic verifies that a topic below the accuracy
// threshold generates a weak topic alert.
func TestCheckNotifications_WeakTopic(t *testing.T) {
	repo := store.NewMockRepository()
	seedPack(t, repo)

	prefs := activePrefs(notifierTestUserID)
	prefs.NotifyInactivity = false
	prefs.NotifyMilestones = false
	prefs.NotifyReadiness = false
	prefs.NotifyStreak = false
	prefs.NotifyWeakTopicPct = 50.0 // threshold: below 50% is weak
	repo.Preferences[notifierTestUserID] = prefs

	// RollingAccuracy 0.3 is below 50%/100 = 0.5
	if err := repo.UpdateTopicStats(context.Background(), model.TopicStats{
		UserID:          notifierTestUserID,
		PackID:          "pack-1",
		TopicID:         "topic-1",
		TotalAttempts:   10,
		CorrectCount:    3,
		RollingAccuracy: 0.3,
	}); err != nil {
		t.Fatalf("UpdateTopicStats: %v", err)
	}

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	notifier := core.NewNotifier(repo)
	notifications, err := notifier.CheckNotifications(context.Background(), notifierTestUserID, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasType(notifications, model.NotificationWeakTopic) {
		t.Errorf("expected weak topic notification, got %v", types(notifications))
	}
}

// TestCheckNotifications_WeakTopicDisabled verifies no weak topic alert when disabled.
func TestCheckNotifications_WeakTopicDisabled(t *testing.T) {
	repo := store.NewMockRepository()
	seedPack(t, repo)

	prefs := activePrefs(notifierTestUserID)
	prefs.NotifyInactivity = false
	prefs.NotifyWeakTopic = false
	prefs.NotifyMilestones = false
	prefs.NotifyReadiness = false
	prefs.NotifyStreak = false
	repo.Preferences[notifierTestUserID] = prefs

	if err := repo.UpdateTopicStats(context.Background(), model.TopicStats{
		UserID:          notifierTestUserID,
		PackID:          "pack-1",
		TopicID:         "topic-1",
		TotalAttempts:   10,
		CorrectCount:    3,
		RollingAccuracy: 0.3,
	}); err != nil {
		t.Fatalf("UpdateTopicStats: %v", err)
	}

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	notifier := core.NewNotifier(repo)
	notifications, err := notifier.CheckNotifications(context.Background(), notifierTestUserID, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hasType(notifications, model.NotificationWeakTopic) {
		t.Errorf("expected no weak topic notification, got one")
	}
}

// TestCheckNotifications_Milestone verifies that exactly 100 total attempts
// across all topics generates a milestone notification.
func TestCheckNotifications_Milestone(t *testing.T) {
	repo := store.NewMockRepository()
	seedPack(t, repo)

	prefs := activePrefs(notifierTestUserID)
	prefs.NotifyInactivity = false
	prefs.NotifyWeakTopic = false
	prefs.NotifyReadiness = false
	prefs.NotifyStreak = false
	repo.Preferences[notifierTestUserID] = prefs

	// Two topics summing to exactly 100 attempts
	if err := repo.UpdateTopicStats(context.Background(), model.TopicStats{
		UserID:        notifierTestUserID,
		PackID:        "pack-1",
		TopicID:       "topic-1",
		TotalAttempts: 60,
		CorrectCount:  40,
	}); err != nil {
		t.Fatalf("UpdateTopicStats: %v", err)
	}
	if err := repo.UpdateTopicStats(context.Background(), model.TopicStats{
		UserID:        notifierTestUserID,
		PackID:        "pack-2",
		TopicID:       "topic-2",
		TotalAttempts: 40,
		CorrectCount:  20,
	}); err != nil {
		t.Fatalf("UpdateTopicStats: %v", err)
	}

	// Make pack-2 also active
	prefs.ActivePackIDs = []string{"pack-1", "pack-2"}
	repo.Preferences[notifierTestUserID] = prefs

	// Seed pack-2 in the repo
	if err := repo.SaveQuizPack(context.Background(), model.QuizPack{
		ID:   "pack-2",
		Name: "Pack Two",
		Topics: []model.Topic{
			{ID: "topic-2", Name: "Topic Two"},
		},
	}); err != nil {
		t.Fatalf("SaveQuizPack pack-2: %v", err)
	}

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	notifier := core.NewNotifier(repo)
	notifications, err := notifier.CheckNotifications(context.Background(), notifierTestUserID, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasType(notifications, model.NotificationMilestone) {
		t.Errorf("expected milestone notification, got %v", types(notifications))
	}
}

// TestCheckNotifications_StreakWarning verifies that a streak >= 5 with no
// review today generates a streak warning.
func TestCheckNotifications_StreakWarning(t *testing.T) {
	repo := store.NewMockRepository()
	seedPack(t, repo)

	prefs := activePrefs(notifierTestUserID)
	prefs.NotifyInactivity = false
	prefs.NotifyWeakTopic = false
	prefs.NotifyMilestones = false
	prefs.NotifyReadiness = false
	repo.Preferences[notifierTestUserID] = prefs

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	yesterday := now.AddDate(0, 0, -1)

	// Topic has a streak of 5 but last reviewed yesterday (not today)
	if err := repo.UpdateTopicStats(context.Background(), model.TopicStats{
		UserID:        notifierTestUserID,
		PackID:        "pack-1",
		TopicID:       "topic-1",
		TotalAttempts: 20,
		CurrentStreak: 5,
	}); err != nil {
		t.Fatalf("UpdateTopicStats: %v", err)
	}

	// QuestionState last reviewed yesterday
	if err := repo.UpdateQuestionState(context.Background(), model.QuestionState{
		UserID:         notifierTestUserID,
		PackID:         "pack-1",
		QuestionID:     "q-1",
		LastReviewedAt: yesterday,
	}); err != nil {
		t.Fatalf("UpdateQuestionState: %v", err)
	}

	notifier := core.NewNotifier(repo)
	notifications, err := notifier.CheckNotifications(context.Background(), notifierTestUserID, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasType(notifications, model.NotificationStreak) {
		t.Errorf("expected streak notification, got %v", types(notifications))
	}
}

// TestCheckNotifications_QuietHours verifies that no notifications are returned
// when the current time falls within quiet hours.
func TestCheckNotifications_QuietHours(t *testing.T) {
	repo := store.NewMockRepository()
	seedPack(t, repo)

	prefs := activePrefs(notifierTestUserID)
	prefs.QuietHoursStart = "22:00"
	prefs.QuietHoursEnd = "08:00"
	repo.Preferences[notifierTestUserID] = prefs

	// 23:00 — within quiet hours (22:00–08:00)
	now := time.Date(2026, 4, 3, 23, 0, 0, 0, time.UTC)

	// Add stale review to trigger inactivity if quiet hours weren't respected
	if err := repo.UpdateQuestionState(context.Background(), model.QuestionState{
		UserID:         notifierTestUserID,
		PackID:         "pack-1",
		QuestionID:     "q-1",
		LastReviewedAt: now.AddDate(0, 0, -5),
	}); err != nil {
		t.Fatalf("UpdateQuestionState: %v", err)
	}

	notifier := core.NewNotifier(repo)
	notifications, err := notifier.CheckNotifications(context.Background(), notifierTestUserID, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(notifications) != 0 {
		t.Errorf("expected no notifications during quiet hours, got %v", types(notifications))
	}
}

// TestCheckNotifications_ReadinessUpdate verifies that a readiness summary is
// generated when NotifyReadiness is enabled.
func TestCheckNotifications_ReadinessUpdate(t *testing.T) {
	repo := store.NewMockRepository()
	seedPack(t, repo)

	prefs := activePrefs(notifierTestUserID)
	prefs.NotifyInactivity = false
	prefs.NotifyWeakTopic = false
	prefs.NotifyMilestones = false
	prefs.NotifyStreak = false
	repo.Preferences[notifierTestUserID] = prefs

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	notifier := core.NewNotifier(repo)
	notifications, err := notifier.CheckNotifications(context.Background(), notifierTestUserID, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !hasType(notifications, model.NotificationReadiness) {
		t.Errorf("expected readiness notification, got %v", types(notifications))
	}
}

// TestCheckNotifications_NoActivePacksNoNotifications verifies that with no active
// packs, no inactivity/weak-topic/streak notifications are generated.
func TestCheckNotifications_NoActivePacksNoNotifications(t *testing.T) {
	repo := store.NewMockRepository()

	prefs := activePrefs(notifierTestUserID)
	prefs.ActivePackIDs = []string{} // no active packs
	prefs.NotifyReadiness = false    // disable readiness to keep result truly empty
	repo.Preferences[notifierTestUserID] = prefs

	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	notifier := core.NewNotifier(repo)
	notifications, err := notifier.CheckNotifications(context.Background(), notifierTestUserID, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, n := range notifications {
		switch n.Type {
		case model.NotificationInactivity, model.NotificationWeakTopic, model.NotificationStreak:
			t.Errorf("unexpected notification %q with no active packs", n.Type)
		}
	}
}

// --- helpers ---

func hasType(ns []model.Notification, t model.NotificationType) bool {
	for _, n := range ns {
		if n.Type == t {
			return true
		}
	}
	return false
}

func types(ns []model.Notification) []model.NotificationType {
	out := make([]model.NotificationType, len(ns))
	for i, n := range ns {
		out[i] = n.Type
	}
	return out
}
