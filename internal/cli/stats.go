package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/sam-liem/quizbot/internal/core"
)

// RunStats displays readiness score and topic breakdown for a pack.
func (a *App) RunStats(packID, topicID string, detailed bool, w io.Writer) error {
	ctx := context.Background()

	pack, err := a.repo.GetQuizPack(ctx, packID)
	if err != nil {
		return fmt.Errorf("pack not found: %s", packID)
	}

	topicStats, err := a.repo.ListTopicStats(ctx, a.userID, packID)
	if err != nil {
		return fmt.Errorf("loading topic stats: %w", err)
	}

	readiness := core.CalculateReadiness(topicStats, *pack)
	_, _ = fmt.Fprintf(w, "Pack: %s (%s)\n", pack.Name, pack.ID)
	_, _ = fmt.Fprintf(w, "Readiness: %.1f%%\n", readiness*100)

	breakdown := core.GetTopicBreakdown(topicStats)
	if len(breakdown) == 0 {
		_, _ = fmt.Fprintln(w, "\nNo topic stats yet.")
		return nil
	}

	// Build a topic name lookup.
	topicNames := make(map[string]string, len(pack.Topics))
	for _, t := range pack.Topics {
		topicNames[t.ID] = t.Name
	}

	_, _ = fmt.Fprintln(w, "\nTopics (weakest first):")
	for _, ts := range breakdown {
		if topicID != "" && ts.TopicID != topicID {
			continue
		}
		name := topicNames[ts.TopicID]
		if name == "" {
			name = ts.TopicID
		}
		_, _ = fmt.Fprintf(w, "  %s: %.1f%% (%d/%d)\n", name, ts.Accuracy*100, ts.CorrectCount, ts.TotalAttempts)
		if detailed {
			_, _ = fmt.Fprintf(w, "    Streak: %d (best: %d)\n", ts.CurrentStreak, ts.BestStreak)
		}
	}

	return nil
}
