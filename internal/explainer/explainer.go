package explainer

import "github.com/sam-liem/quizbot/internal/model"

// Explainer generates explanations for quiz answers.
type Explainer interface {
	Explain(question model.Question, userAnswer, correctAnswer int) string
}
