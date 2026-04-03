package importer

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sam-liem/quizbot/internal/model"
)

// MarkdownParser implements QuestionParser for Markdown-formatted quiz packs.
// The format uses YAML frontmatter for pack metadata and question blocks like:
//
//	## Q: <question text>
//	- <choice>
//	- <choice>
//	> answer: <N>
//	> topic: <topic-id>
//	> explanation: <text>
//
// Question IDs are auto-generated as q001, q002, etc.
type MarkdownParser struct{}

// Parse reads a Markdown quiz pack from r.
func (p *MarkdownParser) Parse(r io.Reader) (*model.QuizPack, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading markdown: %w", err)
	}

	frontmatter, body, err := splitFrontmatter(string(content))
	if err != nil {
		return nil, fmt.Errorf("splitting frontmatter: %w", err)
	}

	var pack model.QuizPack
	if err := yaml.Unmarshal([]byte(frontmatter), &pack); err != nil {
		return nil, fmt.Errorf("parsing frontmatter: %w", err)
	}

	questions, err := parseQuestionBlocks(body)
	if err != nil {
		return nil, fmt.Errorf("parsing questions: %w", err)
	}
	pack.Questions = questions

	return &pack, nil
}

// splitFrontmatter separates YAML frontmatter (between leading --- markers)
// from the body of the document. It returns an error if the markers are absent.
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	lines := strings.Split(content, "\n")

	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", fmt.Errorf("missing opening '---' frontmatter marker")
	}

	// Find closing marker.
	closeIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			closeIdx = i
			break
		}
	}
	if closeIdx == -1 {
		return "", "", fmt.Errorf("missing closing '---' frontmatter marker")
	}

	frontmatter = strings.Join(lines[1:closeIdx], "\n")
	body = strings.Join(lines[closeIdx+1:], "\n")
	return frontmatter, body, nil
}

// parseQuestionBlocks parses the question blocks from the body of a Markdown
// quiz pack. Each block starts with "## Q: <text>".
func parseQuestionBlocks(body string) ([]model.Question, error) {
	scanner := bufio.NewScanner(strings.NewReader(body))

	var questions []model.Question
	var current *model.Question

	flush := func() {
		if current != nil {
			questions = append(questions, *current)
			current = nil
		}
	}

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "## Q:") {
			flush()
			text := strings.TrimSpace(strings.TrimPrefix(line, "## Q:"))
			idx := len(questions) + 1
			current = &model.Question{
				ID:   fmt.Sprintf("q%03d", idx),
				Text: text,
			}
			continue
		}

		if current == nil {
			continue
		}

		// Choice line.
		if strings.HasPrefix(line, "- ") {
			choice := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			current.Choices = append(current.Choices, choice)
			continue
		}

		// Metadata lines (blockquote-style).
		if strings.HasPrefix(line, "> ") {
			meta := strings.TrimSpace(strings.TrimPrefix(line, "> "))
			key, value, found := strings.Cut(meta, ":")
			if !found {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)

			switch key {
			case "answer":
				n, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("question %q: invalid answer index %q: %w", current.ID, value, err)
				}
				current.CorrectIndex = n
			case "topic":
				current.TopicID = value
			case "explanation":
				current.Explanation = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning markdown body: %w", err)
	}

	flush()
	return questions, nil
}
