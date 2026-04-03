package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sam-liem/quizbot/internal/model"
	"github.com/sam-liem/quizbot/internal/store"
)

// SchedulerMessenger is the minimal messenger interface needed by the Scheduler.
// Defined here to avoid a circular import with internal/messenger.
type SchedulerMessenger interface {
	SendQuestion(chatID string, question model.Question) (string, error)
	SendNotification(chatID string, notification model.Notification) error
}

// SchedulerConfig holds per-user configuration for the scheduler run loop.
type SchedulerConfig struct {
	UserID       string
	ChatID       string
	TickInterval time.Duration
	SkipTimeout  time.Duration
}

// pendingQuestion tracks a sent question that has not yet been answered.
type pendingQuestion struct {
	questionID string
	sentAt     time.Time
}

// Scheduler delivers questions at the configured interval, respects
// max_unanswered limits, and skips questions that time out.
type Scheduler struct {
	engine    *QuizEngine
	messenger SchedulerMessenger
	notifier  *Notifier
	repo      store.Repository
	cfg       SchedulerConfig

	mu      sync.Mutex
	pending []pendingQuestion

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewScheduler creates a Scheduler.
func NewScheduler(engine *QuizEngine, messenger SchedulerMessenger, notifier *Notifier, cfg SchedulerConfig) *Scheduler {
	return &Scheduler{
		engine:    engine,
		messenger: messenger,
		notifier:  notifier,
		repo:      engine.Repo(),
		cfg:       cfg,
	}
}

// Start launches the scheduler goroutine. The scheduler stops when ctx is
// cancelled or Stop is called.
func (s *Scheduler) Start(ctx context.Context) {
	childCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	s.wg.Add(1)
	go s.run(childCtx)
}

// Stop cancels the scheduler's internal context and waits for the goroutine
// to finish.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// RecordAnswer removes the given question from the pending list, unblocking
// the scheduler if it was paused at max_unanswered.
func (s *Scheduler) RecordAnswer(questionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	updated := s.pending[:0]
	for _, p := range s.pending {
		if p.questionID != questionID {
			updated = append(updated, p)
		}
	}
	s.pending = updated
}

// run is the main scheduler loop.
func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.cfg.TickInterval)
	defer ticker.Stop()

	var lastSentAt time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.tick(ctx, now, &lastSentAt)
		}
	}
}

// tick executes one scheduler cycle.
func (s *Scheduler) tick(ctx context.Context, now time.Time, lastSentAt *time.Time) {
	// 1. Expire timed-out pending questions.
	s.expireTimedOut(ctx, now)

	// 2. Load user preferences.
	prefs, err := s.repo.GetPreferences(ctx, s.cfg.UserID)
	if err != nil {
		// Non-fatal: skip this tick.
		return
	}

	// 3. Check pending count vs max_unanswered — pause if at limit.
	s.mu.Lock()
	pendingCount := len(s.pending)
	s.mu.Unlock()

	if pendingCount >= prefs.MaxUnanswered {
		return
	}

	// 4. Enforce delivery interval.
	if prefs.DeliveryIntervalMin > 0 && !lastSentAt.IsZero() {
		minInterval := time.Duration(prefs.DeliveryIntervalMin) * time.Minute
		if now.Sub(*lastSentAt) < minInterval {
			return
		}
	}

	// 5. Select next question via spaced repetition across active packs.
	question, err := s.selectNextQuestion(ctx, prefs)
	if err != nil || question == nil {
		return
	}

	// 6. Send question via messenger.
	_, err = s.messenger.SendQuestion(s.cfg.ChatID, *question)
	if err != nil {
		return
	}

	// 7. Track as pending.
	s.mu.Lock()
	s.pending = append(s.pending, pendingQuestion{
		questionID: question.ID,
		sentAt:     now,
	})
	s.mu.Unlock()

	*lastSentAt = now
}

// expireTimedOut marks timed-out pending questions as skipped (updates their
// spaced-repetition state with IntervalDays=1) and removes them from the
// pending list.
func (s *Scheduler) expireTimedOut(ctx context.Context, now time.Time) {
	if s.cfg.SkipTimeout <= 0 {
		return
	}

	s.mu.Lock()
	var expired []string
	active := s.pending[:0]
	for _, p := range s.pending {
		if now.Sub(p.sentAt) >= s.cfg.SkipTimeout {
			expired = append(expired, p.questionID)
		} else {
			active = append(active, p)
		}
	}
	s.pending = active
	s.mu.Unlock()

	for _, qID := range expired {
		s.markSkipped(ctx, qID)
	}
}

// markSkipped updates the question state to reflect a skip (reset interval to 1 day).
func (s *Scheduler) markSkipped(ctx context.Context, questionID string) {
	// Find which pack owns this question.
	prefs, err := s.repo.GetPreferences(ctx, s.cfg.UserID)
	if err != nil {
		return
	}

	for _, packID := range prefs.ActivePackIDs {
		pack, err := s.repo.GetQuizPack(ctx, packID)
		if err != nil {
			continue
		}
		for _, q := range pack.Questions {
			if q.ID != questionID {
				continue
			}
			// Found — update state.
			state, err := s.repo.GetQuestionState(ctx, s.cfg.UserID, packID, questionID)
			if err != nil {
				return
			}
			var current model.QuestionState
			if state != nil {
				current = *state
			} else {
				current = model.QuestionState{
					UserID:     s.cfg.UserID,
					PackID:     packID,
					QuestionID: questionID,
					EaseFactor: 2.5,
				}
			}
			// Treat skip as wrong — resets interval to 1 day.
			updated := CalculateNextReview(current, false, time.Now())
			_ = s.repo.UpdateQuestionState(ctx, updated)
			return
		}
	}
}

// selectNextQuestion picks the most overdue question across all active packs
// that is not currently in the pending list.
func (s *Scheduler) selectNextQuestion(ctx context.Context, prefs *model.UserPreferences) (*model.Question, error) {
	s.mu.Lock()
	pendingIDs := make(map[string]bool, len(s.pending))
	for _, p := range s.pending {
		pendingIDs[p.questionID] = true
	}
	s.mu.Unlock()

	now := time.Now()
	var bestState *model.QuestionState
	var bestQuestion *model.Question

	for _, packID := range prefs.ActivePackIDs {
		pack, err := s.repo.GetQuizPack(ctx, packID)
		if err != nil {
			continue
		}

		// Collect question states for this pack.
		states := make([]model.QuestionState, 0, len(pack.Questions))
		questionByID := make(map[string]*model.Question, len(pack.Questions))

		for i := range pack.Questions {
			q := &pack.Questions[i]
			if pendingIDs[q.ID] {
				continue // skip already-pending
			}
			questionByID[q.ID] = q

			state, err := s.repo.GetQuestionState(ctx, s.cfg.UserID, packID, q.ID)
			if err != nil {
				continue
			}
			if state == nil {
				// Unseen question — treat as immediately due.
				state = &model.QuestionState{
					UserID:       s.cfg.UserID,
					PackID:       packID,
					QuestionID:   q.ID,
					NextReviewAt: time.Time{}, // zero time → always overdue
				}
			}
			states = append(states, *state)
		}

		candidate := SelectNextQuestion(states, now)
		if candidate == nil {
			// No due questions in this pack — find any unseen question.
			for _, q := range pack.Questions {
				if pendingIDs[q.ID] {
					continue
				}
				state, err := s.repo.GetQuestionState(ctx, s.cfg.UserID, packID, q.ID)
				if err != nil || state != nil {
					continue
				}
				// Unseen: use it.
				candidate = &model.QuestionState{
					UserID:       s.cfg.UserID,
					PackID:       packID,
					QuestionID:   q.ID,
					NextReviewAt: time.Time{},
				}
				break
			}
		}
		if candidate == nil {
			continue
		}

		if bestState == nil || candidate.NextReviewAt.Before(bestState.NextReviewAt) {
			if q, ok := questionByID[candidate.QuestionID]; ok {
				bestState = candidate
				bestQuestion = q
			} else {
				// Candidate is from states slice; locate the question.
				for i := range pack.Questions {
					if pack.Questions[i].ID == candidate.QuestionID {
						q2 := pack.Questions[i]
						bestState = candidate
						bestQuestion = &q2
						break
					}
				}
			}
		}
	}

	if bestQuestion == nil {
		return nil, fmt.Errorf("no questions available")
	}
	return bestQuestion, nil
}
