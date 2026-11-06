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
)

// plannedQuiz adds an open あ row quiz session that asked the given cards.
func plannedQuiz(r *fakeRepo, cards ...*SRSCard) *StudySession {
	track, scope := common.LearningTrackHiragana, "a"
	s := r.addSession(common.SessionKindUnitQuiz, &track)
	s.Scope = &scope
	s.QuestionPlan = QuestionPlan{}
	for _, c := range cards {
		s.QuestionPlan = append(s.QuestionPlan, PlannedQuestion{CardID: c.ID, QuestionType: common.QuestionKanaMCQRomaji})
	}
	return s
}

func TestSubmitAnswersChecksTheTestPlan(t *testing.T) {
	newA := func(r *fakeRepo) *SRSCard {
		return r.addCard(newCard(common.SRSItemKana, kanaA, common.CardFacetRecognition))
	}
	newI := func(r *fakeRepo) *SRSCard {
		return r.addCard(newCard(common.SRSItemKana, kanaI, common.CardFacetRecognition))
	}
	runSubmitCases(t, []submitCase{
		{
			name: "a card the test did not ask is refused",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				s := plannedQuiz(r, newA(r))
				return s.ID, []AnswerRequestItem{choiceAnswer(newI(r).ID, common.QuestionKanaMCQRomaji, "i")}
			},
			wantErr: exceptions.ErrTestAnswerNotAsked,
		},
		{
			name: "a card answered more often than asked is refused",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				a := newA(r)
				s := plannedQuiz(r, a, a)
				return s.ID, []AnswerRequestItem{
					choiceAnswer(a.ID, common.QuestionKanaMCQRomaji, "a"),
					choiceAnswer(a.ID, common.QuestionKanaMCQKana, "あ"),
					choiceAnswer(a.ID, common.QuestionKanaMCQRomaji, "a"),
				}
			},
			wantErr: exceptions.ErrTestAnswerNotAsked,
		},
		{
			name: "each asked question takes one answer, either way round, and a resend is not counted twice",
			setup: func(r *fakeRepo) (uuid.UUID, []AnswerRequestItem) {
				a := newA(r)
				s := plannedQuiz(r, a, a)
				first := choiceAnswer(a.ID, common.QuestionKanaMCQRomaji, "a")
				return s.ID, []AnswerRequestItem{first, first, choiceAnswer(a.ID, common.QuestionKanaMCQKana, "あ")}
			},
			check: func(t *testing.T, r *fakeRepo, _ []AnswerResult) {
				if len(r.logs) != 2 {
					t.Errorf("logs = %d, want 2", len(r.logs))
				}
			},
		},
	})
}

func TestSubmitAnswersCountsEarlierAnswersAgainstThePlan(t *testing.T) {
	repo := newFakeRepo()
	a := repo.addCard(newCard(common.SRSItemKana, kanaA, common.CardFacetRecognition))
	s := plannedQuiz(repo, a)
	repo.logs = append(repo.logs, sessionLog(s, a, nil, true, 1500, time.Minute))
	svc := NewSubmitAnswersService(repo, NewCatalog(&fakeContent{}), zap.NewNop())
	svc.now = func() time.Time { return testNow }
	_, err := svc.Submit(context.Background(), testUserID, s.ID, &SubmitAnswersRequest{
		Answers: []AnswerRequestItem{choiceAnswer(a.ID, common.QuestionKanaMCQRomaji, "a")},
	})
	if !errors.Is(err, exceptions.ErrTestAnswerNotAsked) {
		t.Fatalf("err = %v, want %v", err, exceptions.ErrTestAnswerNotAsked)
	}
}

func TestStartTestStoresPlanAndClosesUnfinishedAttempts(t *testing.T) {
	repo := newFakeRepo()
	track, other := common.LearningTrackHiragana, "ka"
	answered := plannedQuiz(repo, repo.addCard(newCard(common.SRSItemKana, kanaA, common.CardFacetRecognition)))
	repo.logs = append(repo.logs, sessionLog(answered, repo.cards[answered.QuestionPlan[0].CardID], nil, true, 1500, time.Minute))
	untouched := plannedQuiz(repo)
	elsewhere := plannedQuiz(repo)
	elsewhere.Scope = &other
	repo.logs = append(repo.logs, sessionLog(elsewhere, repo.cards[answered.QuestionPlan[0].CardID], nil, true, 1500, time.Minute))

	closer := &fakeCloser{}
	svc := NewStartTestService(repo, NewCatalog(&fakeContent{}), NewQuestionBuilder(&fixedRand{}), closer, &fixedRand{}, zap.NewNop())
	svc.now = func() time.Time { return testNow }
	got, err := svc.Start(context.Background(), testUserID, &StartTestRequest{Kind: "unit_quiz", Track: string(track), Scope: "a"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if len(closer.closed) != 1 || closer.closed[0] != answered.ID {
		t.Errorf("closed = %v, want only the answered attempt %s (not %s or %s)", closer.closed, answered.ID, untouched.ID, elsewhere.ID)
	}
	plan := repo.sessions[got.Session.ID].QuestionPlan
	if len(plan) != len(got.Questions) {
		t.Fatalf("plan = %d questions, want %d", len(plan), len(got.Questions))
	}
	for i, q := range got.Questions {
		if plan[i].CardID != q.CardID || string(plan[i].QuestionType) != q.QuestionType {
			t.Errorf("plan[%d] = %+v, want %s %s", i, plan[i], q.CardID, q.QuestionType)
		}
	}
}

func TestQuestionPlanStoresNilAsNull(t *testing.T) {
	var none QuestionPlan
	if v, err := none.Value(); v != nil || err != nil {
		t.Errorf("nil plan = %v, %v, want NULL", v, err)
	}
	plan := QuestionPlan{{CardID: uuid.New(), QuestionType: common.QuestionKanaMCQKana}}
	v, err := plan.Value()
	if err != nil {
		t.Fatal(err)
	}
	var back QuestionPlan
	if err := back.Scan(v); err != nil || len(back) != 1 || back[0] != plan[0] {
		t.Errorf("round trip = %+v, %v, want %+v", back, err, plan)
	}
}
