package cli

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/sam-liem/quizbot/internal/model"
)

// supportedConfigKeys is the set of config keys that can be read/written.
var supportedConfigKeys = map[string]bool{
	"delivery_interval_min":  true,
	"max_unanswered":         true,
	"focus_mode":             true,
	"notify_inactivity":      true,
	"notify_inactivity_days": true,
	"notify_weak_topic":      true,
	"notify_weak_topic_pct":  true,
	"notify_milestones":      true,
	"notify_readiness":       true,
	"notify_streak":          true,
	"quiet_hours_start":      true,
	"quiet_hours_end":        true,
}

// RunConfigList lists all user preferences as key=value pairs.
func (a *App) RunConfigList(w io.Writer) error {
	ctx := context.Background()

	prefs, err := a.getPreferences(ctx)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "delivery_interval_min=%d\n", prefs.DeliveryIntervalMin)
	_, _ = fmt.Fprintf(w, "max_unanswered=%d\n", prefs.MaxUnanswered)
	_, _ = fmt.Fprintf(w, "focus_mode=%s\n", prefs.FocusMode)
	_, _ = fmt.Fprintf(w, "notify_inactivity=%t\n", prefs.NotifyInactivity)
	_, _ = fmt.Fprintf(w, "notify_inactivity_days=%d\n", prefs.NotifyInactivityDays)
	_, _ = fmt.Fprintf(w, "notify_weak_topic=%t\n", prefs.NotifyWeakTopic)
	_, _ = fmt.Fprintf(w, "notify_weak_topic_pct=%.1f\n", prefs.NotifyWeakTopicPct)
	_, _ = fmt.Fprintf(w, "notify_milestones=%t\n", prefs.NotifyMilestones)
	_, _ = fmt.Fprintf(w, "notify_readiness=%t\n", prefs.NotifyReadiness)
	_, _ = fmt.Fprintf(w, "notify_streak=%t\n", prefs.NotifyStreak)
	_, _ = fmt.Fprintf(w, "quiet_hours_start=%s\n", prefs.QuietHoursStart)
	_, _ = fmt.Fprintf(w, "quiet_hours_end=%s\n", prefs.QuietHoursEnd)

	return nil
}

// RunConfigGet gets a single preference value.
func (a *App) RunConfigGet(key string, w io.Writer) error {
	if !supportedConfigKeys[key] {
		return fmt.Errorf("unknown config key: %s", key)
	}

	ctx := context.Background()

	prefs, err := a.getPreferences(ctx)
	if err != nil {
		return err
	}

	val := getPreferenceValue(prefs, key)
	_, _ = fmt.Fprintf(w, "%s=%s\n", key, val)
	return nil
}

// RunConfigSet sets a single preference value.
func (a *App) RunConfigSet(key, value string, w io.Writer) error {
	if !supportedConfigKeys[key] {
		return fmt.Errorf("unknown config key: %s", key)
	}

	ctx := context.Background()

	prefs, err := a.getPreferences(ctx)
	if err != nil {
		return err
	}

	if err := setPreferenceValue(prefs, key, value); err != nil {
		return fmt.Errorf("invalid value for %s: %w", key, err)
	}

	if err := a.savePreferences(ctx, *prefs); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "Set %s=%s\n", key, value)
	return nil
}

// getPreferenceValue returns the string representation of a preference value.
func getPreferenceValue(prefs *model.UserPreferences, key string) string {
	switch key {
	case "delivery_interval_min":
		return strconv.Itoa(prefs.DeliveryIntervalMin)
	case "max_unanswered":
		return strconv.Itoa(prefs.MaxUnanswered)
	case "focus_mode":
		return string(prefs.FocusMode)
	case "notify_inactivity":
		return strconv.FormatBool(prefs.NotifyInactivity)
	case "notify_inactivity_days":
		return strconv.Itoa(prefs.NotifyInactivityDays)
	case "notify_weak_topic":
		return strconv.FormatBool(prefs.NotifyWeakTopic)
	case "notify_weak_topic_pct":
		return fmt.Sprintf("%.1f", prefs.NotifyWeakTopicPct)
	case "notify_milestones":
		return strconv.FormatBool(prefs.NotifyMilestones)
	case "notify_readiness":
		return strconv.FormatBool(prefs.NotifyReadiness)
	case "notify_streak":
		return strconv.FormatBool(prefs.NotifyStreak)
	case "quiet_hours_start":
		return prefs.QuietHoursStart
	case "quiet_hours_end":
		return prefs.QuietHoursEnd
	default:
		return ""
	}
}

// setPreferenceValue parses value and sets the corresponding preference field.
func setPreferenceValue(prefs *model.UserPreferences, key, value string) error {
	switch key {
	case "delivery_interval_min":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("expected integer: %w", err)
		}
		if v < 1 {
			return fmt.Errorf("must be at least 1")
		}
		prefs.DeliveryIntervalMin = v

	case "max_unanswered":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("expected integer: %w", err)
		}
		if v < 1 {
			return fmt.Errorf("must be at least 1")
		}
		prefs.MaxUnanswered = v

	case "focus_mode":
		lower := strings.ToLower(value)
		switch model.FocusMode(lower) {
		case model.FocusModeSingle, model.FocusModeInterleaved:
			prefs.FocusMode = model.FocusMode(lower)
		default:
			return fmt.Errorf("must be 'single' or 'interleaved'")
		}

	case "notify_inactivity":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("expected boolean: %w", err)
		}
		prefs.NotifyInactivity = v

	case "notify_inactivity_days":
		v, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("expected integer: %w", err)
		}
		if v < 1 {
			return fmt.Errorf("must be at least 1")
		}
		prefs.NotifyInactivityDays = v

	case "notify_weak_topic":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("expected boolean: %w", err)
		}
		prefs.NotifyWeakTopic = v

	case "notify_weak_topic_pct":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("expected number: %w", err)
		}
		if v < 0 || v > 100 {
			return fmt.Errorf("must be between 0 and 100")
		}
		prefs.NotifyWeakTopicPct = v

	case "notify_milestones":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("expected boolean: %w", err)
		}
		prefs.NotifyMilestones = v

	case "notify_readiness":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("expected boolean: %w", err)
		}
		prefs.NotifyReadiness = v

	case "notify_streak":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("expected boolean: %w", err)
		}
		prefs.NotifyStreak = v

	case "quiet_hours_start":
		prefs.QuietHoursStart = value

	case "quiet_hours_end":
		prefs.QuietHoursEnd = value

	default:
		return fmt.Errorf("unknown key: %s", key)
	}

	return nil
}
