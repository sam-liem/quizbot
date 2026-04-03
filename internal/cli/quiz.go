package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/sam-liem/quizbot/internal/core"
	"github.com/sam-liem/quizbot/internal/model"
)

// RunQuizStart starts an interactive quiz session.
func (a *App) RunQuizStart(packID string, mode model.SessionMode, count int, topicID string, r io.Reader, w io.Writer) error {
	ctx := context.Background()

	opts := core.QuizOptions{
		QuestionCount: count,
		TopicID:       topicID,
	}

	session, err := a.engine.StartQuiz(ctx, a.userID, packID, mode, opts)
	if err != nil {
		return fmt.Errorf("starting quiz: %w", err)
	}

	return a.runInteractiveQuiz(ctx, session.ID, r, w)
}

// RunQuizResume resumes an in-progress quiz session.
// Since the repository interface doesn't expose a "list sessions" method,
// resume is not yet supported.
func (a *App) RunQuizResume(r io.Reader, w io.Writer) error {
	_ = r
	_ = w
	return fmt.Errorf("no in-progress session found (use 'quiz start' to begin a new quiz)")
}

// runInteractiveQuiz drives the question-answer loop for a quiz session.
func (a *App) runInteractiveQuiz(ctx context.Context, sessionID string, r io.Reader, w io.Writer) error {
	scanner := bufio.NewScanner(r)
	questionNum := 0

	for {
		question, err := a.engine.NextQuestion(ctx, a.userID, sessionID)
		if err != nil {
			return fmt.Errorf("getting next question: %w", err)
		}

		if question == nil {
			break
		}

		questionNum++

		// Get total question count from session status.
		status, err := a.engine.GetSessionStatus(ctx, a.userID, sessionID)
		if err != nil {
			return fmt.Errorf("getting session status: %w", err)
		}

		_, _ = fmt.Fprintf(w, "\nQ%d/%d: %s\n", questionNum, status.TotalQuestions, question.Text)
		for i, choice := range question.Choices {
			_, _ = fmt.Fprintf(w, "  %d) %s\n", i+1, choice)
		}

		_, _ = fmt.Fprint(w, "Your answer: ")

		if !scanner.Scan() {
			return fmt.Errorf("unexpected end of input")
		}

		input := strings.TrimSpace(scanner.Text())
		answerNum, err := strconv.Atoi(input)
		if err != nil || answerNum < 1 || answerNum > len(question.Choices) {
			_, _ = fmt.Fprintf(w, "Invalid input. Please enter a number between 1 and %d.\n", len(question.Choices))
			questionNum-- // retry same question
			continue
		}

		answerIndex := answerNum - 1

		result, err := a.engine.SubmitAnswer(ctx, a.userID, sessionID, question.ID, answerIndex)
		if err != nil {
			return fmt.Errorf("submitting answer: %w", err)
		}

		if result.Correct {
			_, _ = fmt.Fprintln(w, "Correct!")
		} else {
			correctChoice := question.Choices[result.CorrectIndex]
			_, _ = fmt.Fprintf(w, "Wrong. The correct answer is: %s\n", correctChoice)
		}

		if result.Explanation != "" {
			_, _ = fmt.Fprintf(w, "Explanation: %s\n", result.Explanation)
		}
	}

	// Show final score.
	status, err := a.engine.GetSessionStatus(ctx, a.userID, sessionID)
	if err != nil {
		return fmt.Errorf("getting final status: %w", err)
	}

	pct := 0.0
	if status.TotalQuestions > 0 {
		pct = float64(status.Correct) / float64(status.TotalQuestions) * 100.0
	}

	_, _ = fmt.Fprintf(w, "\nScore: %d/%d (%.0f%%)\n", status.Correct, status.TotalQuestions, pct)

	return nil
}
