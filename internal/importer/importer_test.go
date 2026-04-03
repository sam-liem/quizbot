package importer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetectFormat verifies format detection by file extension.
func TestDetectFormat(t *testing.T) {
	tests := []struct {
		filename    string
		wantFormat  Format
		wantErr     bool
	}{
		{"pack.yaml", FormatYAML, false},
		{"pack.yml", FormatYAML, false},
		{"pack.json", FormatJSON, false},
		{"pack.md", FormatMarkdown, false},
		{"pack.txt", "", true},
		{"noextension", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.filename, func(t *testing.T) {
			got, err := DetectFormat(tc.filename)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantFormat, got)
			}
		})
	}
}

// TestParseFile_YAML verifies parsing a valid YAML quiz pack file.
func TestParseFile_YAML(t *testing.T) {
	pack, err := ParseFile("testdata/valid_pack.yaml")
	require.NoError(t, err)
	require.NotNil(t, pack)

	assert.Equal(t, "test-pack", pack.ID)
	assert.Equal(t, "Test Pack", pack.Name)
	assert.Equal(t, "1.0.0", pack.Version)
	assert.Len(t, pack.Topics, 2)
	assert.Len(t, pack.Questions, 2)
	assert.Equal(t, "q001", pack.Questions[0].ID)
	assert.Equal(t, 1, pack.Questions[0].CorrectIndex)
	assert.Equal(t, "q002", pack.Questions[1].ID)
	assert.Equal(t, 2, pack.Questions[1].CorrectIndex)
}

// TestParseFile_JSON verifies parsing a valid JSON quiz pack file.
func TestParseFile_JSON(t *testing.T) {
	pack, err := ParseFile("testdata/valid_pack.json")
	require.NoError(t, err)
	require.NotNil(t, pack)

	assert.Equal(t, "test-pack", pack.ID)
	assert.Equal(t, "Test Pack", pack.Name)
	assert.Equal(t, "1.0.0", pack.Version)
	assert.Len(t, pack.Topics, 2)
	assert.Len(t, pack.Questions, 2)
	assert.Equal(t, "q001", pack.Questions[0].ID)
	assert.Equal(t, 1, pack.Questions[0].CorrectIndex)
}

// TestParseFile_Markdown verifies parsing a valid Markdown quiz pack file.
func TestParseFile_Markdown(t *testing.T) {
	pack, err := ParseFile("testdata/valid_pack.md")
	require.NoError(t, err)
	require.NotNil(t, pack)

	assert.Equal(t, "test-pack", pack.ID)
	assert.Equal(t, "Test Pack", pack.Name)
	assert.Equal(t, "1.0.0", pack.Version)
	assert.Len(t, pack.Topics, 2)
	assert.Len(t, pack.Questions, 2)

	q0 := pack.Questions[0]
	assert.Equal(t, "q001", q0.ID)
	assert.Equal(t, "What is 2+2?", q0.Text)
	assert.Equal(t, "topic1", q0.TopicID)
	assert.Equal(t, 1, q0.CorrectIndex)
	assert.Equal(t, []string{"3", "4", "5", "6"}, q0.Choices)
	assert.Equal(t, "2+2 equals 4.", q0.Explanation)

	q1 := pack.Questions[1]
	assert.Equal(t, "q002", q1.ID)
	assert.Equal(t, "What color is the sky?", q1.Text)
	assert.Equal(t, "topic2", q1.TopicID)
	assert.Equal(t, 2, q1.CorrectIndex)
}

// TestParseFile_InvalidMissingID verifies validation catches missing ID.
func TestParseFile_InvalidMissingID(t *testing.T) {
	_, err := ParseFile("testdata/invalid_missing_id.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id")
}

// TestParseFile_InvalidBadIndex verifies validation catches out-of-bounds correct_index.
func TestParseFile_InvalidBadIndex(t *testing.T) {
	_, err := ParseFile("testdata/invalid_bad_index.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "correct_index")
}

// TestParseFile_NonexistentFile verifies error on missing file.
func TestParseFile_NonexistentFile(t *testing.T) {
	_, err := ParseFile("testdata/does_not_exist.yaml")
	require.Error(t, err)
}

// TestParseReader_YAML verifies parsing a YAML reader directly.
func TestParseReader_YAML(t *testing.T) {
	input := `id: "rdr-pack"
name: "Reader Pack"
version: "1.0.0"
test_format:
  question_count: 1
  pass_mark_pct: 50.0
  time_limit_sec: 60
topics:
  - id: "t1"
    name: "T1"
questions:
  - id: "q001"
    topic_id: "t1"
    text: "Is Go great?"
    choices: ["No", "Yes"]
    correct_index: 1
`
	p, err := ParserForFormat(FormatYAML)
	require.NoError(t, err)

	pack, err := p.Parse(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "rdr-pack", pack.ID)
	assert.Len(t, pack.Questions, 1)
}

// TestParseReader_JSON verifies parsing a JSON reader directly.
func TestParseReader_JSON(t *testing.T) {
	input := `{
  "id": "rdr-pack",
  "name": "Reader Pack",
  "version": "1.0.0",
  "test_format": {"question_count": 1, "pass_mark_pct": 50.0, "time_limit_sec": 60},
  "topics": [{"id": "t1", "name": "T1"}],
  "questions": [
    {"id": "q001", "topic_id": "t1", "text": "Is Go great?", "choices": ["No", "Yes"], "correct_index": 1}
  ]
}`
	p, err := ParserForFormat(FormatJSON)
	require.NoError(t, err)

	pack, err := p.Parse(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "rdr-pack", pack.ID)
	assert.Len(t, pack.Questions, 1)
}

// TestParseReader_Markdown verifies parsing a Markdown reader directly.
func TestParseReader_Markdown(t *testing.T) {
	input := `---
id: "rdr-pack"
name: "Reader Pack"
version: "1.0.0"
test_format:
  question_count: 1
  pass_mark_pct: 50.0
  time_limit_sec: 60
topics:
  - id: "t1"
    name: "T1"
---

## Q: Is Go great?
- No
- Yes
> answer: 1
> topic: t1
> explanation: Go is great.
`
	p, err := ParserForFormat(FormatMarkdown)
	require.NoError(t, err)

	pack, err := p.Parse(strings.NewReader(input))
	require.NoError(t, err)
	assert.Equal(t, "rdr-pack", pack.ID)
	assert.Len(t, pack.Questions, 1)
	assert.Equal(t, "Is Go great?", pack.Questions[0].Text)
	assert.Equal(t, 1, pack.Questions[0].CorrectIndex)
}
