package core

import (
	"testing"

	"github.com/sam-liem/quizbot/internal/model"
)

// helpers

func makeTopicStats(topicID string, totalAttempts, correctCount int, rollingAccuracy float64, currentStreak, bestStreak int) model.TopicStats {
	return model.TopicStats{
		TopicID:         topicID,
		TotalAttempts:   totalAttempts,
		CorrectCount:    correctCount,
		RollingAccuracy: rollingAccuracy,
		CurrentStreak:   currentStreak,
		BestStreak:      bestStreak,
	}
}

func makePackWithTopics(topicQCounts map[string]int) model.QuizPack {
	pack := model.QuizPack{}
	qID := 0
	for topicID, count := range topicQCounts {
		for i := 0; i < count; i++ {
			qID++
			pack.Questions = append(pack.Questions, model.Question{
				TopicID: topicID,
			})
			_ = qID
		}
	}
	return pack
}

// CalculateReadiness tests

func TestCalculateReadiness_AllTopicsPerfect(t *testing.T) {
	// 2 topics, 2 questions each, both 100% accuracy → readiness = 1.0
	stats := []model.TopicStats{
		makeTopicStats("t1", 10, 10, 1.0, 5, 5),
		makeTopicStats("t2", 8, 8, 1.0, 4, 4),
	}
	pack := makePackWithTopics(map[string]int{"t1": 2, "t2": 2})

	got := CalculateReadiness(stats, pack)
	if got != 1.0 {
		t.Errorf("expected readiness 1.0, got %f", got)
	}
}

func TestCalculateReadiness_MixedAccuracy(t *testing.T) {
	// t1: 3 questions at 80%, t2: 1 question at 50%
	// weighted = (3/4)*0.8 + (1/4)*0.5 = 0.6 + 0.125 = 0.725
	stats := []model.TopicStats{
		makeTopicStats("t1", 10, 8, 0.8, 2, 3),
		makeTopicStats("t2", 4, 2, 0.5, 1, 2),
	}
	pack := makePackWithTopics(map[string]int{"t1": 3, "t2": 1})

	got := CalculateReadiness(stats, pack)
	want := 0.725
	if abs(got-want) > 1e-9 {
		t.Errorf("expected readiness %f, got %f", want, got)
	}
}

func TestCalculateReadiness_NoStats(t *testing.T) {
	// nil topicStats → 0.0
	pack := makePackWithTopics(map[string]int{"t1": 2})

	got := CalculateReadiness(nil, pack)
	if got != 0.0 {
		t.Errorf("expected 0.0, got %f", got)
	}
}

func TestCalculateReadiness_EmptyPack(t *testing.T) {
	// empty pack → 0.0
	stats := []model.TopicStats{
		makeTopicStats("t1", 10, 10, 1.0, 5, 5),
	}
	pack := model.QuizPack{}

	got := CalculateReadiness(stats, pack)
	if got != 0.0 {
		t.Errorf("expected 0.0, got %f", got)
	}
}

func TestCalculateReadiness_TopicWithNoStats(t *testing.T) {
	// t1 at 80% (1 question), t2 has no stats (1 question, treated as 0%)
	// weighted = (1/2)*0.8 + (1/2)*0.0 = 0.4
	stats := []model.TopicStats{
		makeTopicStats("t1", 10, 8, 0.8, 2, 3),
	}
	pack := makePackWithTopics(map[string]int{"t1": 1, "t2": 1})

	got := CalculateReadiness(stats, pack)
	want := 0.4
	if abs(got-want) > 1e-9 {
		t.Errorf("expected readiness %f, got %f", want, got)
	}
}

// GetTopicBreakdown tests

func TestGetTopicBreakdown(t *testing.T) {
	// 3 topics with different accuracies → sorted weakest first
	stats := []model.TopicStats{
		makeTopicStats("t1", 10, 8, 0.8, 3, 5),
		makeTopicStats("t2", 6, 3, 0.5, 1, 2),
		makeTopicStats("t3", 4, 4, 1.0, 4, 4),
	}

	result := GetTopicBreakdown(stats)

	if len(result) != 3 {
		t.Fatalf("expected 3 summaries, got %d", len(result))
	}

	// Sorted weakest first: t2(0.5), t1(0.8), t3(1.0)
	expectedOrder := []struct {
		topicID  string
		accuracy float64
	}{
		{"t2", 0.5},
		{"t1", 0.8},
		{"t3", 1.0},
	}

	for i, exp := range expectedOrder {
		s := result[i]
		if s.TopicID != exp.topicID {
			t.Errorf("result[%d]: expected topicID %q, got %q", i, exp.topicID, s.TopicID)
		}
		if s.Accuracy != exp.accuracy {
			t.Errorf("result[%d]: expected accuracy %f, got %f", i, exp.accuracy, s.Accuracy)
		}
	}

	// Verify all fields for t2 (result[0])
	t2 := result[0]
	if t2.TotalAttempts != 6 {
		t.Errorf("t2 TotalAttempts: expected 6, got %d", t2.TotalAttempts)
	}
	if t2.CorrectCount != 3 {
		t.Errorf("t2 CorrectCount: expected 3, got %d", t2.CorrectCount)
	}
	if t2.CurrentStreak != 1 {
		t.Errorf("t2 CurrentStreak: expected 1, got %d", t2.CurrentStreak)
	}
	if t2.BestStreak != 2 {
		t.Errorf("t2 BestStreak: expected 2, got %d", t2.BestStreak)
	}
}

func TestGetTopicBreakdown_Empty(t *testing.T) {
	// nil input → empty (nil) result
	result := GetTopicBreakdown(nil)
	if result != nil {
		t.Errorf("expected nil result for nil input, got %v", result)
	}
}

func TestGetTopicBreakdown_SingleTopic(t *testing.T) {
	stats := []model.TopicStats{
		makeTopicStats("t1", 5, 4, 0.8, 2, 3),
	}

	result := GetTopicBreakdown(stats)

	if len(result) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(result))
	}
	s := result[0]
	if s.TopicID != "t1" {
		t.Errorf("expected topicID t1, got %q", s.TopicID)
	}
	if s.Accuracy != 0.8 {
		t.Errorf("expected accuracy 0.8, got %f", s.Accuracy)
	}
	if s.TotalAttempts != 5 {
		t.Errorf("expected TotalAttempts 5, got %d", s.TotalAttempts)
	}
	if s.CorrectCount != 4 {
		t.Errorf("expected CorrectCount 4, got %d", s.CorrectCount)
	}
	if s.CurrentStreak != 2 {
		t.Errorf("expected CurrentStreak 2, got %d", s.CurrentStreak)
	}
	if s.BestStreak != 3 {
		t.Errorf("expected BestStreak 3, got %d", s.BestStreak)
	}
}

// abs is a helper for float comparison.
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
