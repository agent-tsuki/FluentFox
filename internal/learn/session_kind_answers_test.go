package learn

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/srs"
)

func dueIn(item uuid.UUID, in time.Duration) SRSCard {
	c := reviewCard(item)
	c.Due = testNow.Add(in)
	return c
}

func TestSubmitAnswersPracticeAndTests(t *testing.T) {
	hiragana := common.LearningTrackHiragana
	runSubmitCases(t, []submitCase{
		{
			name: "a right practice answer leaves the schedule alone",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindPractice, &hiragana)
				c := r.addCard(dueIn(kanaA, 10*24*time.Hour))
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "a")}
			},
			check: func(t *testing.T, r *fakeRepo, res []AnswerResult) {
				log := r.logs[0]
				if log.Rating != nil || log.CountsForSRS || log.GradeReason != common.GradeReasonAnalyticsOnly {
					t.Errorf("log = %+v, want an unrated analytics-only answer", log)
				}
				card := r.cards[log.CardID]
				if !card.Due.Equal(testNow.Add(10*24*time.Hour)) || card.Reps != 4 {
					t.Errorf("card = %+v, want it unchanged", card)
				}
				if res[0].Label != "good" || res[0].Reason != "analytics_only" || res[0].Requeue || res[0].NextDue == nil {
					t.Errorf("result = %+v, want good, analytics only, with the unchanged due", res[0])
				}
				if day := r.activity["2026-03-10"]; day == nil || day.ReviewsDone != 1 || day.CorrectCount != 1 {
					t.Errorf("activity = %+v, want the answer counted", day)
				}
			},
		},
		{
			name: "a practice miss pulls the due date to the safety net and counts the confusion",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindPractice, &hiragana)
				c := r.addCard(dueIn(kanaA, 10*24*time.Hour))
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "o")}
			},
			check: func(t *testing.T, r *fakeRepo, res []AnswerResult) {
				card := r.cards[r.logs[0].CardID]
				want := testNow.Add(common.SAFETY_NET_DUE)
				if !card.Due.Equal(want) || card.Lapses != 0 || card.State != srs.StateReview {
					t.Errorf("card = %+v, want due %v and no lapse", card, want)
				}
				if res[0].Label != "missed" || res[0].NextDue == nil || !res[0].NextDue.Equal(want) {
					t.Errorf("result = %+v, want missed with next due %v", res[0], want)
				}
				if len(r.confusions) != 1 || r.confusions[0].ChosenItemID != kanaO {
					t.Errorf("confusions = %+v, want あ mistaken for お", r.confusions)
				}
				if r.activity["2026-03-10"].AgainCount != 0 {
					t.Error("an unrated miss counted as Again")
				}
			},
		},
		{
			name: "a miss never pushes a card due sooner later",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindUnitQuiz, &hiragana)
				c := r.addCard(dueIn(kanaA, -time.Hour))
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, common.QuestionKanaMCQRomaji, "o")}
			},
			check: func(t *testing.T, r *fakeRepo, _ []AnswerResult) {
				if card := r.cards[r.logs[0].CardID]; !card.Due.Equal(testNow.Add(-time.Hour)) {
					t.Errorf("due = %v, want it left at %v", card.Due, testNow.Add(-time.Hour))
				}
			},
		},
		{
			name: "a speed-round miss in a review session also gets the safety net",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindReview, nil)
				c := r.addCard(dueIn(kanaA, 5*24*time.Hour))
				return s.ID, []AnswerRequestItem{choiceAnswer(c.ID, common.QuestionKanaSpeed, "o")}
			},
			check: func(t *testing.T, r *fakeRepo, _ []AnswerResult) {
				if card := r.cards[r.logs[0].CardID]; !card.Due.Equal(testNow.Add(common.SAFETY_NET_DUE)) {
					t.Errorf("due = %v, want the safety net", card.Due)
				}
			},
		},
		{
			name: "a test answer on an unlearned item creates an unscheduled new card",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindSkipTest, &hiragana)
				id := cardID(testUserID, common.SRSItemKana, kanaI, common.CardFacetRecognition)
				return s.ID, []AnswerRequestItem{choiceAnswer(id, common.QuestionKanaMCQRomaji, "i")}
			},
			check: func(t *testing.T, r *fakeRepo, res []AnswerResult) {
				card := r.cards[cardID(testUserID, common.SRSItemKana, kanaI, common.CardFacetRecognition)]
				if card == nil || card.State != srs.StateNew {
					t.Fatalf("card = %+v, want a new card", card)
				}
				if res[0].NextDue != nil || res[0].Requeue || !res[0].Correct {
					t.Errorf("result = %+v, want correct with no schedule", res[0])
				}
				if r.activity["2026-03-10"].NewLearned != 0 {
					t.Error("a test answer counted a new item as learned")
				}
			},
		},
		{
			name: "a lesson schedules a card a test created and introduces it now",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				id := cardID(testUserID, common.SRSItemKana, kanaI, common.CardFacetRecognition)
				r.addCard(SRSCard{
					ID: id, ItemType: common.SRSItemKana, ItemID: kanaI, Facet: common.CardFacetRecognition,
					State: srs.StateNew, Due: testNow.Add(-48 * time.Hour), IntroducedAt: testNow.Add(-48 * time.Hour),
				})
				s := r.addSession(common.SessionKindLesson, &hiragana)
				return s.ID, []AnswerRequestItem{choiceAnswer(id, common.QuestionKanaMCQRomaji, "i")}
			},
			check: func(t *testing.T, r *fakeRepo, _ []AnswerResult) {
				card := r.cards[r.logs[0].CardID]
				if card.State != srs.StateLearning || card.IntroducedAt.Before(testNow.Add(-2*time.Hour)) {
					t.Errorf("card = %+v, want learning and introduced in this session", card)
				}
				if r.activity["2026-03-10"].NewLearned != 1 {
					t.Error("the lesson did not count the item as learned")
				}
			},
		},
	})
}

func TestSubmitAnswersPracticeRejects(t *testing.T) {
	hiragana := common.LearningTrackHiragana
	runSubmitCases(t, []submitCase{
		{
			name: "a practice cannot create a card for another track's item",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := r.addSession(common.SessionKindPractice, &hiragana)
				id := cardID(testUserID, common.SRSItemKana, kanaKA, common.CardFacetRecognition)
				return s.ID, []AnswerRequestItem{choiceAnswer(id, common.QuestionKanaMCQRomaji, "a")}
			},
			wantErr: exceptions.ErrSRSCardNotFound,
		},
	})
}
