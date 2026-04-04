package cli_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sam-liem/quizbot/internal/cli"
	"github.com/sam-liem/quizbot/internal/model"
	"github.com/sam-liem/quizbot/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testUserID = "test-user"

// testPack returns a sample quiz pack for testing.
func testPack() model.QuizPack {
	return model.QuizPack{
		ID:          "life-in-uk",
		Name:        "Life in the UK",
		Description: "Life in the UK test prep",
		Version:     "1.0",
		TestFormat: model.TestFormat{
			QuestionCount: 2,
			PassMarkPct:   75,
			TimeLimitSec:  45,
		},
		Topics: []model.Topic{
			{ID: "history", Name: "History"},
			{ID: "govt", Name: "Government"},
		},
		Questions: []model.Question{
			{
				ID:           "q1",
				TopicID:      "history",
				Text:         "When was the Magna Carta signed?",
				Choices:      []string{"1215", "1066", "1485", "1603"},
				CorrectIndex: 0,
				Explanation:  "The Magna Carta was signed in 1215.",
			},
			{
				ID:           "q2",
				TopicID:      "govt",
				Text:         "Who is the head of state?",
				Choices:      []string{"Prime Minister", "The Monarch", "Speaker of the House"},
				CorrectIndex: 1,
				Explanation:  "The Monarch is the head of state.",
			},
			{
				ID:           "q3",
				TopicID:      "history",
				Text:         "When was the Battle of Hastings?",
				Choices:      []string{"1066", "1215", "1485"},
				CorrectIndex: 0,
				Explanation:  "The Battle of Hastings was in 1066.",
			},
		},
	}
}

// newTestApp creates an App backed by a MockRepository with an optional test pack.
func newTestApp(withPack bool) (*cli.App, *store.MockRepository) {
	repo := store.NewMockRepository()
	if withPack {
		_ = repo.SaveQuizPack(context.Background(), testPack())
	}
	app := cli.NewApp(repo, testUserID)
	return app, repo
}

// --- Packs tests ---

func TestPacksList(t *testing.T) {
	app, _ := newTestApp(true)
	var buf bytes.Buffer

	err := app.RunPacksList(&buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "life-in-uk")
	assert.Contains(t, output, "Life in the UK")
	assert.Contains(t, output, "3 questions")
}

func TestPacksList_Empty(t *testing.T) {
	app, _ := newTestApp(false)
	var buf bytes.Buffer

	err := app.RunPacksList(&buf)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "No quiz packs found.")
}

func TestPacksActivate(t *testing.T) {
	app, repo := newTestApp(true)
	var buf bytes.Buffer

	err := app.RunPacksActivate("life-in-uk", &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Activated pack: life-in-uk")

	// Verify the pack is in active list.
	prefs, err := repo.GetPreferences(context.Background(), testUserID)
	require.NoError(t, err)
	assert.Contains(t, prefs.ActivePackIDs, "life-in-uk")
}

func TestPacksActivate_NotFound(t *testing.T) {
	app, _ := newTestApp(false)
	var buf bytes.Buffer

	err := app.RunPacksActivate("nonexistent", &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pack not found")
}

func TestPacksDeactivate(t *testing.T) {
	app, repo := newTestApp(true)
	var buf bytes.Buffer

	// Activate first, then deactivate.
	err := app.RunPacksActivate("life-in-uk", &buf)
	require.NoError(t, err)

	buf.Reset()
	err = app.RunPacksDeactivate("life-in-uk", &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Deactivated pack: life-in-uk")

	// Verify the pack is removed from active list.
	prefs, err := repo.GetPreferences(context.Background(), testUserID)
	require.NoError(t, err)
	assert.NotContains(t, prefs.ActivePackIDs, "life-in-uk")
}

// --- Config tests ---

func TestConfigList(t *testing.T) {
	app, _ := newTestApp(false)
	var buf bytes.Buffer

	err := app.RunConfigList(&buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "delivery_interval_min=60")
	assert.Contains(t, output, "max_unanswered=3")
	assert.Contains(t, output, "focus_mode=single")
	assert.Contains(t, output, "notify_inactivity=true")
	assert.Contains(t, output, "quiet_hours_start=22:00")
	assert.Contains(t, output, "quiet_hours_end=08:00")
}

func TestConfigGet(t *testing.T) {
	app, _ := newTestApp(false)
	var buf bytes.Buffer

	err := app.RunConfigGet("delivery_interval_min", &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "delivery_interval_min=60")
}

func TestConfigGet_UnknownKey(t *testing.T) {
	app, _ := newTestApp(false)
	var buf bytes.Buffer

	err := app.RunConfigGet("nonexistent_key", &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown config key")
}

func TestConfigSet(t *testing.T) {
	app, repo := newTestApp(false)
	var buf bytes.Buffer

	err := app.RunConfigSet("delivery_interval_min", "30", &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "Set delivery_interval_min=30")

	// Verify the value was persisted.
	prefs, err := repo.GetPreferences(context.Background(), testUserID)
	require.NoError(t, err)
	assert.Equal(t, 30, prefs.DeliveryIntervalMin)
}

func TestConfigSet_InvalidValue(t *testing.T) {
	app, _ := newTestApp(false)
	var buf bytes.Buffer

	err := app.RunConfigSet("delivery_interval_min", "not-a-number", &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
}

// --- Stats tests ---

func TestStats(t *testing.T) {
	app, repo := newTestApp(true)

	// Seed some topic stats.
	_ = repo.UpdateTopicStats(context.Background(), model.TopicStats{
		UserID:          testUserID,
		PackID:          "life-in-uk",
		TopicID:         "history",
		TotalAttempts:   10,
		CorrectCount:    8,
		RollingAccuracy: 0.8,
		CurrentStreak:   3,
		BestStreak:      5,
	})
	_ = repo.UpdateTopicStats(context.Background(), model.TopicStats{
		UserID:          testUserID,
		PackID:          "life-in-uk",
		TopicID:         "govt",
		TotalAttempts:   5,
		CorrectCount:    3,
		RollingAccuracy: 0.6,
		CurrentStreak:   1,
		BestStreak:      2,
	})

	var buf bytes.Buffer
	err := app.RunStats("life-in-uk", "", false, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Pack: Life in the UK")
	assert.Contains(t, output, "Readiness:")
	assert.Contains(t, output, "History")
	assert.Contains(t, output, "Government")
}

func TestStats_NoPack(t *testing.T) {
	app, _ := newTestApp(false)
	var buf bytes.Buffer

	err := app.RunStats("nonexistent", "", false, &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pack not found")
}

// --- Import tests ---

func TestImport(t *testing.T) {
	app, repo := newTestApp(false)

	// Create a temp YAML file.
	yamlContent := `id: test-pack
name: Test Pack
description: A test pack
version: "1.0"
test_format:
  question_count: 1
  pass_mark_pct: 75
  time_limit_sec: 30
topics:
  - id: t1
    name: Topic One
questions:
  - id: q1
    topic_id: t1
    text: "What is 1+1?"
    choices:
      - "1"
      - "2"
      - "3"
    correct_index: 1
    explanation: "1+1=2"
`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-pack.yaml")
	err := os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	var buf bytes.Buffer
	err = app.RunImport(tmpFile, "", &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Imported pack: Test Pack")
	assert.Contains(t, output, "1 questions")

	// Verify pack is in repo.
	pack, err := repo.GetQuizPack(context.Background(), "test-pack")
	require.NoError(t, err)
	assert.Equal(t, "Test Pack", pack.Name)
	assert.Len(t, pack.Questions, 1)
}

func TestImport_PathTraversal(t *testing.T) {
	app, _ := newTestApp(false)
	var buf bytes.Buffer

	err := app.RunImport("../../etc/passwd", "", &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal")
}

func TestImport_NonexistentFile(t *testing.T) {
	app, _ := newTestApp(false)
	var buf bytes.Buffer

	err := app.RunImport("/tmp/does-not-exist-quizbot-test.yaml", "", &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- Quiz tests ---

func TestQuizStart(t *testing.T) {
	app, _ := newTestApp(true)

	// Simulate answering 2 questions (mock mode with count=2).
	// The pack has 3 questions, mock selects 2 random ones.
	// We answer both correctly by trying all possible correct answers.
	// Since question order is random, we input the correct answer index+1 for each.
	// For simplicity, use practice mode which has deterministic order.
	input := "1\n2\n1\n"
	reader := strings.NewReader(input)

	var buf bytes.Buffer
	err := app.RunQuizStart("life-in-uk", model.SessionModePractice, 0, "", reader, &buf)
	require.NoError(t, err)

	output := buf.String()
	// Should contain question numbers and score.
	assert.Contains(t, output, "Q1/")
	assert.Contains(t, output, "Score:")
}

func TestQuizStart_WithTopicFilter(t *testing.T) {
	app, _ := newTestApp(true)

	// Filter to "history" topic only (2 questions: q1 and q3).
	input := "1\n1\n"
	reader := strings.NewReader(input)

	var buf bytes.Buffer
	err := app.RunQuizStart("life-in-uk", model.SessionModePractice, 0, "history", reader, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Q1/2")
	assert.Contains(t, output, "Q2/2")
	assert.Contains(t, output, "Score:")
}

func TestQuizStart_PackNotFound(t *testing.T) {
	app, _ := newTestApp(false)

	reader := strings.NewReader("")
	var buf bytes.Buffer
	err := app.RunQuizStart("nonexistent", model.SessionModePractice, 0, "", reader, &buf)
	require.Error(t, err)
}
