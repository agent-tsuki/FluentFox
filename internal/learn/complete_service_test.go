package learn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/srs"
)

func sessionLog(s *StudySession, card *SRSCard, rating *srs.Rating, correct bool, activeMs int, at time.Duration) ReviewLog {
	reason := common.GradeReasonNormal
	if !correct {
		reason = common.GradeReasonWrong
	}
	return ReviewLog{
		ID: uuid.New(), UserID: testUserID, CardID: card.ID, SessionID: s.ID, ClientEventID: uuid.New(),
		QuestionType: common.QuestionKanaMCQRomaji, AnsweredAt: s.StartedAt.Add(at), IsCorrect: correct,
		ActiveMs: activeMs, Rating: rating, GradeReason: reason,
	}
}

func TestCompleteSessionService(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(r *fakeRepo) uuid.UUID
		wantErr error
		check   func(t *testing.T, r *fakeRepo, got *SessionResult)
	}{
		{
			name: "totals, labels and next due come from the session's answers",
			setup: func(r *fakeRepo) uuid.UUID {
				s := r.addSession(common.SessionKindReview, nil)
				a := r.addCard(reviewCard(kanaA))
				o := r.addCard(reviewCard(kanaO))
				r.logs = []ReviewLog{
					sessionLog(s, a, ptr(srs.RatingAgain), false, 2000, time.Minute),
					sessionLog(s, o, ptr(srs.RatingGood), true, 1500, 2*time.Minute),
					// A later practice answer on the requeued card does not change its label.
					sessionLog(s, a, ptr(srs.RatingGood), true, 1000, 3*time.Minute),
				}
				return s.ID
			},
			check: func(t *testing.T, r *fakeRepo, got *SessionResult) {
				if got.AnswerCount != 3 || got.CorrectCount != 2 || got.ActiveMs != 4500 {
					t.Errorf("totals = %d/%d/%d, want 3/2/4500", got.AnswerCount, got.CorrectCount, got.ActiveMs)
				}
				if got.Counts != (GradeCounts{Good: 1, Missed: 1}) {
					t.Errorf("counts = %+v, want one good and one missed", got.Counts)
				}
				if len(got.Cards) != 2 || got.Cards[0].Char != "あ" || got.Cards[0].Label != "missed" || got.Cards[0].NextDue == nil {
					t.Errorf("cards = %+v, want あ missed first with a next due", got.Cards)
				}
				if got.Session.CompletedAt == nil || r.completedSession == nil {
					t.Error("session not marked completed")
				}
			},
		},
		{
			name: "an analytics answer before the graded one is not the card's result",
			setup: func(r *fakeRepo) uuid.UUID {
				s := r.addSession(common.SessionKindReview, nil)
				a := r.addCard(reviewCard(kanaA))
				r.logs = []ReviewLog{
					sessionLog(s, a, nil, true, 900, time.Minute),
					sessionLog(s, a, ptr(srs.RatingHard), true, 5000, 2*time.Minute),
				}
				return s.ID
			},
			check: func(t *testing.T, _ *fakeRepo, got *SessionResult) {
				if len(got.Cards) != 1 || got.Cards[0].Label != "shaky" || got.Cards[0].ActiveMs != 5000 {
					t.Errorf("cards = %+v, want the shaky graded answer", got.Cards)
				}
			},
		},
		{
			name: "completing twice returns the stored totals without updating",
			setup: func(r *fakeRepo) uuid.UUID {
				s := r.addSession(common.SessionKindLesson, nil)
				s.CompletedAt = ptr(testNow.Add(-time.Minute))
				s.AnswerCount, s.CorrectCount = 7, 6
				return s.ID
			},
			check: func(t *testing.T, r *fakeRepo, got *SessionResult) {
				if r.completedSession != nil {
					t.Error("a completed session was updated again")
				}
				if got.AnswerCount != 7 || got.CorrectCount != 6 || len(got.Cards) != 0 {
					t.Errorf("result = %+v, want the stored totals", got)
				}
			},
		},
		{
			name: "a session of another user is not found",
			setup: func(r *fakeRepo) uuid.UUID {
				s := r.addSession(common.SessionKindReview, nil)
				s.UserID = otherUser
				return s.ID
			},
			wantErr: exceptions.ErrStudySessionNotFound,
		},
		{
			name:    "an unknown session is not found",
			setup:   func(*fakeRepo) uuid.UUID { return uuid.New() },
			wantErr: exceptions.ErrStudySessionNotFound,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			id := tc.setup(repo)
			svc := NewCompleteSessionService(repo, NewTestScoreService(repo, NewCatalog(&fakeContent{}), nil, zap.NewNop()), zap.NewNop())
			svc.now = func() time.Time { return testNow }
			got, err := svc.Complete(context.Background(), testUserID, id)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				tc.check(t, repo, &got)
			}
		})
	}
}
