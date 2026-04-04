package model

type UserPreferences struct {
	UserID               string    `json:"user_id"`
	DeliveryIntervalMin  int       `json:"delivery_interval_min"`
	MaxUnanswered        int       `json:"max_unanswered"`
	ActivePackIDs        []string  `json:"active_pack_ids"`
	FocusMode            FocusMode `json:"focus_mode"`
	NotifyInactivity     bool      `json:"notify_inactivity"`
	NotifyInactivityDays int       `json:"notify_inactivity_days"`
	NotifyWeakTopic      bool      `json:"notify_weak_topic"`
	NotifyWeakTopicPct   float64   `json:"notify_weak_topic_pct"`
	NotifyMilestones     bool      `json:"notify_milestones"`
	NotifyReadiness      bool      `json:"notify_readiness"`
	NotifyStreak         bool      `json:"notify_streak"`
	QuietHoursStart      string    `json:"quiet_hours_start"`
	QuietHoursEnd        string    `json:"quiet_hours_end"`
}

func DefaultPreferences(userID string) UserPreferences {
	return UserPreferences{
		UserID:               userID,
		DeliveryIntervalMin:  60,
		MaxUnanswered:        3,
		ActivePackIDs:        []string{},
		FocusMode:            FocusModeSingle,
		NotifyInactivity:     true,
		NotifyInactivityDays: 2,
		NotifyWeakTopic:      true,
		NotifyWeakTopicPct:   50.0,
		NotifyMilestones:     true,
		NotifyReadiness:      true,
		NotifyStreak:         true,
		QuietHoursStart:      "22:00",
		QuietHoursEnd:        "08:00",
	}
}
