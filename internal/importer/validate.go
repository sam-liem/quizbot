package importer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sam-liem/quizbot/internal/model"
)

// Validate checks a QuizPack for structural correctness and returns a joined
// error containing all validation failures, or nil if the pack is valid.
func Validate(pack model.QuizPack) error {
	var errs []string

	// Required top-level fields.
	if strings.TrimSpace(pack.ID) == "" {
		errs = append(errs, "id is required")
	}
	if strings.TrimSpace(pack.Name) == "" {
		errs = append(errs, "name is required")
	}
	if strings.TrimSpace(pack.Version) == "" {
		errs = append(errs, "version is required")
	}

	// TestFormat constraints.
	if pack.TestFormat.QuestionCount <= 0 {
		errs = append(errs, "test_format.question_count must be > 0")
	}
	if pack.TestFormat.PassMarkPct < 0 || pack.TestFormat.PassMarkPct > 100 {
		errs = append(errs, fmt.Sprintf("test_format.pass_mark_pct must be between 0 and 100, got %.2f", pack.TestFormat.PassMarkPct))
	}

	// Topic validation.
	topicIDs := make(map[string]struct{}, len(pack.Topics))
	for i, topic := range pack.Topics {
		if strings.TrimSpace(topic.ID) == "" {
			errs = append(errs, fmt.Sprintf("topics[%d]: id is required", i))
		} else {
			if _, dup := topicIDs[topic.ID]; dup {
				errs = append(errs, fmt.Sprintf("topics[%d]: duplicate topic id %q", i, topic.ID))
			}
			topicIDs[topic.ID] = struct{}{}
		}
		if strings.TrimSpace(topic.Name) == "" {
			errs = append(errs, fmt.Sprintf("topics[%d]: name is required", i))
		}
	}

	// Question validation.
	questionIDs := make(map[string]struct{}, len(pack.Questions))
	for i, q := range pack.Questions {
		prefix := fmt.Sprintf("questions[%d] (id=%q)", i, q.ID)

		if strings.TrimSpace(q.ID) == "" {
			prefix = fmt.Sprintf("questions[%d]", i)
			errs = append(errs, fmt.Sprintf("%s: id is required", prefix))
		} else {
			if _, dup := questionIDs[q.ID]; dup {
				errs = append(errs, fmt.Sprintf("%s: duplicate question id %q", prefix, q.ID))
			}
			questionIDs[q.ID] = struct{}{}
		}

		if strings.TrimSpace(q.Text) == "" {
			errs = append(errs, fmt.Sprintf("%s: text is required", prefix))
		}
		if len(q.Choices) < 2 {
			errs = append(errs, fmt.Sprintf("%s: must have at least 2 choices, got %d", prefix, len(q.Choices)))
		}
		if q.CorrectIndex < 0 || q.CorrectIndex >= len(q.Choices) {
			errs = append(errs, fmt.Sprintf("%s: correct_index %d is out of bounds [0, %d)", prefix, q.CorrectIndex, len(q.Choices)))
		}
		if strings.TrimSpace(q.TopicID) == "" {
			errs = append(errs, fmt.Sprintf("%s: topic_id is required", prefix))
		} else if _, ok := topicIDs[q.TopicID]; !ok {
			errs = append(errs, fmt.Sprintf("%s: topic_id %q does not reference a valid topic", prefix, q.TopicID))
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}
