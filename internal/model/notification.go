package model

import "time"

type NotificationType string

const (
	NotificationInactivity NotificationType = "inactivity"
	NotificationWeakTopic  NotificationType = "weak_topic"
	NotificationMilestone  NotificationType = "milestone"
	NotificationReadiness  NotificationType = "readiness"
	NotificationStreak     NotificationType = "streak"
)

type Notification struct {
	Type    NotificationType `json:"type"`
	Message string           `json:"message"`
	SentAt  time.Time        `json:"sent_at"`
}
