package core

import (
	"context"
	"fmt"
	"time"

	"github.com/sam-liem/quizbot/internal/model"
	"github.com/sam-liem/quizbot/internal/store"
)

// milestones are the total-attempts thresholds that generate a celebration.
var milestones = []int{50, 100, 250, 500, 1000}

// Notifier evaluates notification conditions for users and returns any
// notifications that should be sent.
type Notifier struct {
	repo store.Repository
}

// NewNotifier creates a Notifier backed by the given repository.
func NewNotifier(repo store.Repository) *Notifier {
	return &Notifier{repo: repo}
}

// CheckNotifications evaluates all notification conditions for userID at the
// given time and returns the set of notifications that should be sent.
//
// Returns an empty slice (not an error) when the user is within quiet hours.
func (n *Notifier) CheckNotifications(ctx context.Context, userID string, now time.Time) ([]model.Notification, error) {
	prefs, err := n.repo.GetPreferences(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("CheckNotifications: getting preferences: %w", err)
	}

	// Quiet hours check: suppress all notifications.
	if inQuietHours(now, prefs.QuietHoursStart, prefs.QuietHoursEnd) {
		return nil, nil
	}

	var notifications []model.Notification

	// 1. Inactivity reminder.
	if prefs.NotifyInactivity {
		n, err := n.checkInactivity(ctx, userID, prefs, now)
		if err != nil {
			return nil, fmt.Errorf("CheckNotifications: inactivity check: %w", err)
		}
		if n != nil {
			notifications = append(notifications, *n)
		}
	}

	// 2. Weak topic alert.
	if prefs.NotifyWeakTopic {
		weak, err := n.checkWeakTopics(ctx, userID, prefs, now)
		if err != nil {
			return nil, fmt.Errorf("CheckNotifications: weak topic check: %w", err)
		}
		notifications = append(notifications, weak...)
	}

	// 3. Milestone celebration.
	if prefs.NotifyMilestones {
		m, err := n.checkMilestone(ctx, userID, prefs, now)
		if err != nil {
			return nil, fmt.Errorf("CheckNotifications: milestone check: %w", err)
		}
		if m != nil {
			notifications = append(notifications, *m)
		}
	}

	// 4. Streak warning.
	if prefs.NotifyStreak {
		streak, err := n.checkStreak(ctx, userID, prefs, now)
		if err != nil {
			return nil, fmt.Errorf("CheckNotifications: streak check: %w", err)
		}
		notifications = append(notifications, streak...)
	}

	// 5. Readiness update.
	if prefs.NotifyReadiness {
		r, err := n.checkReadiness(ctx, userID, prefs, now)
		if err != nil {
			return nil, fmt.Errorf("CheckNotifications: readiness check: %w", err)
		}
		if r != nil {
			notifications = append(notifications, *r)
		}
	}

	return notifications, nil
}

// checkInactivity returns an inactivity notification when the most recent
// question review across all active packs is older than NotifyInactivityDays.
// Returns nil when there are no active packs.
func (n *Notifier) checkInactivity(ctx context.Context, userID string, prefs *model.UserPreferences, now time.Time) (*model.Notification, error) {
	if len(prefs.ActivePackIDs) == 0 {
		return nil, nil
	}

	threshold := now.AddDate(0, 0, -prefs.NotifyInactivityDays)
	var mostRecent time.Time
	hasAnyState := false

	for _, packID := range prefs.ActivePackIDs {
		pack, err := n.repo.GetQuizPack(ctx, packID)
		if err != nil {
			// Pack may not exist; skip rather than error.
			continue
		}
		for _, q := range pack.Questions {
			state, err := n.repo.GetQuestionState(ctx, userID, packID, q.ID)
			if err != nil {
				return nil, fmt.Errorf("getting question state for %q: %w", q.ID, err)
			}
			if state == nil {
				continue
			}
			hasAnyState = true
			if state.LastReviewedAt.After(mostRecent) {
				mostRecent = state.LastReviewedAt
			}
		}
	}

	// No question state at all means the user hasn't started — trigger inactivity
	// only if there are questions available (i.e. packs exist with questions).
	// If no state exists, treat it as never reviewed → trigger inactivity.
	if !hasAnyState || mostRecent.Before(threshold) {
		return &model.Notification{
			Type:    model.NotificationInactivity,
			Message: fmt.Sprintf("No study activity in the last %d days. Time to review!", prefs.NotifyInactivityDays),
			SentAt:  now,
		}, nil
	}

	return nil, nil
}

// checkWeakTopics returns one notification per topic whose RollingAccuracy is
// below prefs.NotifyWeakTopicPct/100.
func (n *Notifier) checkWeakTopics(ctx context.Context, userID string, prefs *model.UserPreferences, now time.Time) ([]model.Notification, error) {
	threshold := prefs.NotifyWeakTopicPct / 100.0
	var notifications []model.Notification

	for _, packID := range prefs.ActivePackIDs {
		allStats, err := n.repo.ListTopicStats(ctx, userID, packID)
		if err != nil {
			return nil, fmt.Errorf("listing topic stats for pack %q: %w", packID, err)
		}
		for _, ts := range allStats {
			if ts.RollingAccuracy < threshold {
				notifications = append(notifications, model.Notification{
					Type:    model.NotificationWeakTopic,
					Message: fmt.Sprintf("Weak topic detected in pack %q, topic %q (accuracy %.0f%%). Consider extra practice.", packID, ts.TopicID, ts.RollingAccuracy*100),
					SentAt:  now,
				})
			}
		}
	}

	return notifications, nil
}

// checkMilestone returns a milestone notification when the total attempts across
// all active packs exactly equals one of the milestone values.
func (n *Notifier) checkMilestone(ctx context.Context, userID string, prefs *model.UserPreferences, now time.Time) (*model.Notification, error) {
	var total int

	for _, packID := range prefs.ActivePackIDs {
		allStats, err := n.repo.ListTopicStats(ctx, userID, packID)
		if err != nil {
			return nil, fmt.Errorf("listing topic stats for pack %q: %w", packID, err)
		}
		for _, ts := range allStats {
			total += ts.TotalAttempts
		}
	}

	for _, m := range milestones {
		if total == m {
			return &model.Notification{
				Type:    model.NotificationMilestone,
				Message: fmt.Sprintf("Milestone reached! You have answered %d questions. Keep it up!", total),
				SentAt:  now,
			}, nil
		}
	}

	return nil, nil
}

// checkStreak returns streak-at-risk notifications for any topic with
// CurrentStreak >= 5 that has not been reviewed today.
func (n *Notifier) checkStreak(ctx context.Context, userID string, prefs *model.UserPreferences, now time.Time) ([]model.Notification, error) {
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var notifications []model.Notification

	for _, packID := range prefs.ActivePackIDs {
		// Determine whether any question in the pack was reviewed today.
		pack, err := n.repo.GetQuizPack(ctx, packID)
		if err != nil {
			continue
		}

		reviewedToday := false
		for _, q := range pack.Questions {
			state, err := n.repo.GetQuestionState(ctx, userID, packID, q.ID)
			if err != nil {
				return nil, fmt.Errorf("getting question state for streak check: %w", err)
			}
			if state != nil && !state.LastReviewedAt.Before(todayStart) {
				reviewedToday = true
				break
			}
		}

		if reviewedToday {
			continue
		}

		// Check topic streaks.
		allStats, err := n.repo.ListTopicStats(ctx, userID, packID)
		if err != nil {
			return nil, fmt.Errorf("listing topic stats for pack %q: %w", packID, err)
		}
		for _, ts := range allStats {
			if ts.CurrentStreak >= 5 {
				notifications = append(notifications, model.Notification{
					Type:    model.NotificationStreak,
					Message: fmt.Sprintf("Your %d-question streak in topic %q is at risk! Study today to keep it going.", ts.CurrentStreak, ts.TopicID),
					SentAt:  now,
				})
			}
		}
	}

	return notifications, nil
}

// checkReadiness returns a readiness summary notification. In v1 it always
// fires when called; frequency capping is handled by the scheduler.
func (n *Notifier) checkReadiness(ctx context.Context, userID string, prefs *model.UserPreferences, now time.Time) (*model.Notification, error) {
	var totalReadiness float64
	packCount := 0

	for _, packID := range prefs.ActivePackIDs {
		pack, err := n.repo.GetQuizPack(ctx, packID)
		if err != nil {
			continue
		}
		allStats, err := n.repo.ListTopicStats(ctx, userID, packID)
		if err != nil {
			return nil, fmt.Errorf("listing topic stats for readiness: %w", err)
		}
		readiness := CalculateReadiness(allStats, *pack)
		totalReadiness += readiness
		packCount++
	}

	var avgReadiness float64
	if packCount > 0 {
		avgReadiness = totalReadiness / float64(packCount)
	}

	return &model.Notification{
		Type:    model.NotificationReadiness,
		Message: fmt.Sprintf("Your estimated readiness is %.0f%%. Keep studying!", avgReadiness*100),
		SentAt:  now,
	}, nil
}

// inQuietHours returns true when t falls within the quiet hours window
// defined by start and end strings in "HH:MM" format.
//
// Supports overnight ranges (e.g. 22:00–08:00). Returns false when either
// start or end is empty.
func inQuietHours(t time.Time, start, end string) bool {
	if start == "" || end == "" {
		return false
	}

	startH, startM, err1 := parseHHMM(start)
	endH, endM, err2 := parseHHMM(end)
	if err1 != nil || err2 != nil {
		return false
	}

	// Express everything as minutes-since-midnight for easy comparison.
	nowMins := t.Hour()*60 + t.Minute()
	startMins := startH*60 + startM
	endMins := endH*60 + endM

	if startMins < endMins {
		// Same-day window: [startMins, endMins)
		return nowMins >= startMins && nowMins < endMins
	}

	// Overnight window: [startMins, midnight) ∪ [midnight, endMins)
	return nowMins >= startMins || nowMins < endMins
}

// parseHHMM parses a "HH:MM" string into hours and minutes.
func parseHHMM(s string) (int, int, error) {
	var h, m int
	_, err := fmt.Sscanf(s, "%d:%d", &h, &m)
	if err != nil {
		return 0, 0, fmt.Errorf("parseHHMM: invalid format %q: %w", s, err)
	}
	return h, m, nil
}
