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

func choiceAnswer(card uuid.UUID, qt common.QuestionType, chosen string) AnswerRequestItem {
	return AnswerRequestItem{
		ClientEventID: uuid.New(), CardID: card, QuestionType: string(qt),
		ShownAt: testNow.Add(-time.Minute), DurationMs: 1500, Attempts: 1, ChosenOption: ptr(chosen),
	}
}

func reviewCard(item uuid.UUID) SRSCard {
	last := testNow.Add(-10 * 24 * time.Hour)
	return SRSCard{
		ID: uuid.New(), ItemType: common.SRSItemKana, ItemID: item, Facet: common.CardFacetRecognition,
		State: srs.StateReview, Stability: 10, Difficulty: 5, Reps: 4, ScheduledDays: 10,
		Due: testNow.Add(-time.Hour), LastReviewAt: &last,
	}
}

type submitCase struct {
	name    string
	setup   func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem)
	wantErr error
	check   func(t *testing.T, r *fakeRepo, res []AnswerResult)
}

func runSubmitCases(t *testing.T, tests []submitCase) {
	t.Helper()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			sessionID, answers := tc.setup(repo)
			svc := NewSubmitAnswersService(repo, NewCatalog(&fakeContent{}), zap.NewNop())
			svc.now = func() time.Time { return testNow }
			got, err := svc.Submit(context.Background(), testUserID, sessionID, &SubmitAnswersRequest{Answers: answers})
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if len(repo.logs) != 0 {
					t.Errorf("a rejected batch stored %d logs", len(repo.logs))
				}
				return
			}
			if len(got.Results) != len(answers) {
				t.Fatalf("results = %d, want %d", len(got.Results), len(answers))
			}
			tc.check(t, repo, got.Results)
		})
	}
}

func TestSubmitAnswersRejects(t *testing.T) {
	hiragana := common.LearningTrackHiragana
	runSubmitCases(t, []submitCase{
		{
			name: "a session of another user is not found",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindReview, nil)
				s.UserID = otherUser
				c := r.addCard(reviewCard(kanaA))
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "a")}
			},
			wantErr: exceptions.ErrStudySessionNotFound,
		},
		{
			name: "a completed session takes no answers",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindReview, nil)
				s.CompletedAt = ptr(testNow)
				c := r.addCard(reviewCard(kanaA))
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "a")}
			},
			wantErr: exceptions.ErrStudySessionCompleted,
		},
		{
			name: "a review session cannot create cards",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindReview, nil)
				id := cardID(testUserID, common.SRSItemKana, kanaA, common.CardFacetRecognition)
				return s.ID, []AnswerRequestItem{choiceAnswer(id, common.QuestionKanaMCQRomaji, "a")}
			},
			wantErr: exceptions.ErrSRSCardNotFound,
		},
		{
			name: "a lesson cannot create a card for another track's item",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindLesson, &hiragana)
				id := cardID(testUserID, common.SRSItemKana, kanaKA, common.CardFacetRecognition)
				return s.ID, []AnswerRequestItem{choiceAnswer(id, common.QuestionKanaMCQRomaji, "a")}
			},
			wantErr: exceptions.ErrSRSCardNotFound,
		},
		{
			name: "a question type must fit the card's facet",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindReview, nil)
				c := r.addCard(reviewCard(kanaA))
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, common.QuestionKanjiMeaningMCQ, "a")}
			},
			wantErr: exceptions.ErrQuestionTypeMismatch,
		},
		{
			name: "an unknown question type is rejected",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindReview, nil)
				c := r.addCard(reviewCard(kanaA))
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, "kana_essay", "a")}
			},
			wantErr: exceptions.ErrQuestionTypeMismatch,
		},
	})
}

func TestSubmitAnswersLessonAndScheduling(t *testing.T) {
	hiragana := common.LearningTrackHiragana
	runSubmitCases(t, []submitCase{
		{
			name: "a lesson check answer creates the card and puts it in learning",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindLesson, &hiragana)
				id := cardID(testUserID, common.SRSItemKana, kanaA, common.CardFacetRecognition)
				return s.ID, []AnswerRequestItem{choiceAnswer(id, common.QuestionKanaMCQRomaji, "a")}
			},
			check: func(t *testing.T, r *fakeRepo, res []AnswerResult) {
				id := cardID(testUserID, common.SRSItemKana, kanaA, common.CardFacetRecognition)
				card := r.cards[id]
				if card == nil || card.State != srs.StateLearning || card.ItemID != kanaA {
					t.Fatalf("card = %+v, want a learning card for あ", card)
				}
				if !res[0].Correct || res[0].Label != "good" || !res[0].Requeue || res[0].CorrectAnswer != "a" {
					t.Errorf("result = %+v, want correct, good (MCQ cap), requeued", res[0])
				}
				if day := r.activity["2026-03-10"]; day == nil || day.NewLearned != 1 || day.ReviewsDone != 1 {
					t.Errorf("activity = %+v, want one new item and one answer", day)
				}
				if log := r.logs[0]; !log.CountsForSRS || log.StateBefore != srs.StateNew || log.StabilityBefore != nil {
					t.Errorf("log = %+v, want a counted answer on a new card", log)
				}
			},
		},
		{
			name: "a correct review answer moves the due date and logs before and after",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindReview, nil)
				c := r.addCard(reviewCard(kanaA))
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "a")}
			},
			check: func(t *testing.T, r *fakeRepo, res []AnswerResult) {
				log := r.logs[0]
				card := r.cards[log.CardID]
				if !card.Due.After(testNow.Add(24*time.Hour)) || card.Reps != 5 {
					t.Errorf("card = %+v, want due days ahead and reps 5", card)
				}
				if log.StabilityBefore == nil || *log.StabilityBefore != 10 || log.StabilityAfter == nil || log.RetrievabilityBefore == nil {
					t.Errorf("log = %+v, want before and after values", log)
				}
				if res[0].NextDue == nil || !res[0].NextDue.Equal(card.Due) || res[0].Requeue {
					t.Errorf("result = %+v, want next due %v and no requeue", res[0], card.Due)
				}
				if res[0].UsualMs != int(common.USUAL_KANA_MCQ/time.Millisecond) {
					t.Errorf("usual = %d, want the type default", res[0].UsualMs)
				}
			},
		},
		{
			name: "only the first graded answer of the day counts for a review card",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindReview, nil)
				c := r.addCard(reviewCard(kanaA))
				return s.ID, []AnswerRequestItem{
					choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "a"),
					choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "i"),
				}
			},
			check: func(t *testing.T, r *fakeRepo, res []AnswerResult) {
				if !r.logs[0].CountsForSRS || r.logs[1].CountsForSRS {
					t.Errorf("counts = %v, %v; want only the first", r.logs[0].CountsForSRS, r.logs[1].CountsForSRS)
				}
				if res[1].Correct || res[1].Label != "missed" || r.cards[r.logs[0].CardID].Lapses != 0 {
					t.Errorf("second result = %+v, want missed without a lapse", res[1])
				}
			},
		},
		{
			name: "a wrong pick counts the confusion and sends the card to relearning",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindReview, nil)
				card := reviewCard(kanaA)
				card.Lapses = common.LEECH_LAPSES - 1
				c := r.addCard(card)
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "o")}
			},
			check: func(t *testing.T, r *fakeRepo, res []AnswerResult) {
				card := r.cards[r.logs[0].CardID]
				if card.State != srs.StateRelearning || !card.IsLeech {
					t.Errorf("card = %+v, want relearning and a leech", card)
				}
				if len(r.confusions) != 1 || r.confusions[0].TargetItemID != kanaA || r.confusions[0].ChosenItemID != kanaO {
					t.Errorf("confusions = %+v, want あ mistaken for お", r.confusions)
				}
				if res[0].Label != "missed" || res[0].Reason != "wrong" || !res[0].Requeue || res[0].CorrectAnswer != "a" {
					t.Errorf("result = %+v, want missed, wrong, requeued", res[0])
				}
				if r.activity["2026-03-10"].AgainCount != 1 {
					t.Errorf("again count = %d, want 1", r.activity["2026-03-10"].AgainCount)
				}
			},
		},
	})
}

func TestSubmitAnswersIdempotencyAndGoal(t *testing.T) {
	stored := uuid.New()
	runSubmitCases(t, []submitCase{
		{
			name: "a resent answer returns the stored result without a new log",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindReview, nil)
				c := r.addCard(reviewCard(kanaA))
				due := testNow.Add(72 * time.Hour)
				r.logs = append(r.logs, ReviewLog{
					UserID: testUserID, CardID: c.ID, SessionID: s.ID, ClientEventID: stored,
					QuestionType: common.QuestionKanaMCQRomaji, AnsweredAt: testNow.Add(-time.Minute), IsCorrect: true,
					ActiveMs: 1200, Rating: ptr(srs.RatingGood), GradeReason: common.GradeReasonMCQCap,
					CountsForSRS: true, ScheduledDays: 3, DueAfter: &due,
				})
				a := choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "o")
				a.ClientEventID = stored
				return s.ID, []AnswerRequestItem{a}
			},
			check: func(t *testing.T, r *fakeRepo, res []AnswerResult) {
				if len(r.logs) != 1 {
					t.Errorf("logs = %d, want the stored one only", len(r.logs))
				}
				if !res[0].Correct || res[0].Label != "good" || res[0].ActiveMs != 1200 || res[0].Requeue {
					t.Errorf("result = %+v, want the stored good answer", res[0])
				}
				if len(r.activity) != 0 {
					t.Errorf("a resend changed the daily activity: %+v", r.activity)
				}
			},
		},
		{
			name: "reaching the daily goal marks the day",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				r.activity["2026-03-10"] = &DailyActivity{ReviewsDone: 19}
				r.addCard(reviewCard(kanaI))
				s := r.addSession(common.SessionKindReview, nil)
				c := r.addCard(reviewCard(kanaA))
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "a")}
			},
			check: func(t *testing.T, r *fakeRepo, _ []AnswerResult) {
				if !r.activity["2026-03-10"].GoalMet {
					t.Error("goal not met at 20 answers")
				}
			},
		},
		{
			name: "clearing everything due with enough answers marks the day",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				r.activity["2026-03-10"] = &DailyActivity{ReviewsDone: 4}
				s := r.addSession(common.SessionKindReview, nil)
				c := r.addCard(reviewCard(kanaA))
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "a")}
			},
			check: func(t *testing.T, r *fakeRepo, _ []AnswerResult) {
				if !r.activity["2026-03-10"].GoalMet {
					t.Error("goal not met after clearing all due cards with 5 answers")
				}
			},
		},
		{
			name: "cards still due keep the goal open below the daily goal",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				r.activity["2026-03-10"] = &DailyActivity{ReviewsDone: 4}
				r.addCard(reviewCard(kanaI))
				s := r.addSession(common.SessionKindReview, nil)
				c := r.addCard(reviewCard(kanaA))
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "a")}
			},
			check: func(t *testing.T, r *fakeRepo, _ []AnswerResult) {
				if r.activity["2026-03-10"].GoalMet {
					t.Error("goal met while a card is still due")
				}
			},
		},
	})
}
