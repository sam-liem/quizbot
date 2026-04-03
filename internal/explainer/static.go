package explainer

import (
	"fmt"
	"strings"

	"github.com/sam-liem/quizbot/internal/model"
)

// StaticExplainer returns the pre-authored explanation from the question.
// If the question has no explanation, returns a generic message.
type StaticExplainer struct{}

func (s *StaticExplainer) Explain(question model.Question, userAnswer, correctAnswer int) string {
	if strings.TrimSpace(question.Explanation) != "" {
		return question.Explanation
	}
	return fmt.Sprintf("The correct answer is: %s", question.Choices[correctAnswer])
}
