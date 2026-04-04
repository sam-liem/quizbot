package core

import (
	"sort"

	"github.com/sam-liem/quizbot/internal/model"
)

// TopicSummary is a display-oriented summary for stats breakdowns.
type TopicSummary struct {
	TopicID       string  `json:"topic_id"`
	Accuracy      float64 `json:"accuracy"`
	TotalAttempts int     `json:"total_attempts"`
	CorrectCount  int     `json:"correct_count"`
	CurrentStreak int     `json:"current_streak"`
	BestStreak    int     `json:"best_streak"`
}

// CalculateReadiness computes estimated pass probability as weighted
// average of topic accuracies by question distribution.
// Topics with no stats = 0% accuracy.
func CalculateReadiness(topicStats []model.TopicStats, pack model.QuizPack) float64 {
	if len(pack.Questions) == 0 {
		return 0.0
	}

	// Count questions per topic.
	questionCount := make(map[string]int)
	for _, q := range pack.Questions {
		questionCount[q.TopicID]++
	}

	total := float64(len(pack.Questions))

	// Build lookup of topic stats by TopicID.
	statsLookup := make(map[string]model.TopicStats, len(topicStats))
	for _, ts := range topicStats {
		statsLookup[ts.TopicID] = ts
	}

	var readiness float64
	for topicID, count := range questionCount {
		weight := float64(count) / total
		accuracy := 0.0
		if ts, ok := statsLookup[topicID]; ok {
			accuracy = ts.RollingAccuracy
		}
		readiness += weight * accuracy
	}

	return readiness
}

// GetTopicBreakdown converts raw topic stats to sorted summaries (weakest first).
func GetTopicBreakdown(topicStats []model.TopicStats) []TopicSummary {
	if len(topicStats) == 0 {
		return nil
	}

	summaries := make([]TopicSummary, len(topicStats))
	for i, ts := range topicStats {
		summaries[i] = TopicSummary{
			TopicID:       ts.TopicID,
			Accuracy:      ts.RollingAccuracy,
			TotalAttempts: ts.TotalAttempts,
			CorrectCount:  ts.CorrectCount,
			CurrentStreak: ts.CurrentStreak,
			BestStreak:    ts.BestStreak,
		}
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Accuracy < summaries[j].Accuracy
	})

	return summaries
}
