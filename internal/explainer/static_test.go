package explainer

import (
	"testing"

	"github.com/sam-liem/quizbot/internal/model"
)

func TestStaticExplainer_WithExplanation(t *testing.T) {
	s := &StaticExplainer{}
	q := model.Question{
		Text:        "What year did the Battle of Hastings occur?",
		Choices:     []string{"1055", "1066", "1077", "1088"},
		CorrectIndex: 1,
		Explanation: "The Battle of Hastings took place in 1066.",
	}

	got := s.Explain(q, 0, 1)
	want := "The Battle of Hastings took place in 1066."
	if got != want {
		t.Errorf("Explain() = %q, want %q", got, want)
	}
}

func TestStaticExplainer_EmptyExplanation(t *testing.T) {
	s := &StaticExplainer{}
	q := model.Question{
		Text:        "What is the capital of the UK?",
		Choices:     []string{"Edinburgh", "London", "Cardiff", "Belfast"},
		CorrectIndex: 1,
		Explanation: "",
	}

	got := s.Explain(q, 0, 1)
	want := "The correct answer is: London"
	if got != want {
		t.Errorf("Explain() = %q, want %q", got, want)
	}
}

func TestStaticExplainer_WhitespaceExplanation(t *testing.T) {
	s := &StaticExplainer{}
	q := model.Question{
		Text:        "What is the capital of the UK?",
		Choices:     []string{"Edinburgh", "London", "Cardiff", "Belfast"},
		CorrectIndex: 1,
		Explanation: "   \t\n  ",
	}

	got := s.Explain(q, 0, 1)
	want := "The correct answer is: London"
	if got != want {
		t.Errorf("Explain() = %q, want %q", got, want)
	}
}

func TestStaticExplainer_CorrectAndWrongAnswer(t *testing.T) {
	s := &StaticExplainer{}
	explanation := "Parliament is the supreme legislative body."
	q := model.Question{
		Text:        "What is the UK's supreme legislative body?",
		Choices:     []string{"The Crown", "Parliament", "The Cabinet", "The Supreme Court"},
		CorrectIndex: 1,
		Explanation: explanation,
	}

	// Wrong answer
	got := s.Explain(q, 0, 1)
	if got != explanation {
		t.Errorf("wrong answer: Explain() = %q, want %q", got, explanation)
	}

	// Correct answer
	got = s.Explain(q, 1, 1)
	if got != explanation {
		t.Errorf("correct answer: Explain() = %q, want %q", got, explanation)
	}
}
