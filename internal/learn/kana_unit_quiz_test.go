package learn

import (
	"math/rand/v2"
	"testing"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
)

// seededRand shuffles for real, so the plan's ordering rules are tested on
// many random orders.
type seededRand struct{ r *rand.Rand }

func (s seededRand) IntN(n int) int                     { return s.r.IntN(n) }
func (s seededRand) Shuffle(n int, swap func(i, j int)) { s.r.Shuffle(n, swap) }

func kanaItems(n int) []*Item {
	out := make([]*Item, n)
	for i := range out {
		out[i] = &Item{ID: uuid.New(), Type: common.SRSItemKana}
	}
	return out
}

func TestKanaQuizPlan(t *testing.T) {
	tests := []struct {
		name      string
		kana      int
		wantTotal int
		// wantEach is how often every kana is asked; 0 when not every kana fits.
		wantEach int
	}{
		{name: "a five-kana row asks each kana four times", kana: 5, wantTotal: 20, wantEach: 4},
		{name: "a three-kana row asks each kana four times", kana: 3, wantTotal: 12, wantEach: 4},
		{name: "twelve kana fill twenty questions", kana: 12, wantTotal: 20},
		{name: "twenty-five kana ask twenty different ones", kana: 25, wantTotal: 20},
		{name: "a unit of two kana still alternates", kana: 2, wantTotal: 8, wantEach: 4},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for seed := range uint64(200) {
				items := kanaItems(tc.kana)
				plan := kanaQuizPlan(items, seededRand{rand.New(rand.NewPCG(seed, 7))})
				if len(plan) != tc.wantTotal {
					t.Fatalf("seed %d: %d questions, want %d", seed, len(plan), tc.wantTotal)
				}
				asked := map[*Item]map[common.QuestionType]int{}
				ways := map[common.QuestionType]int{}
				for i, a := range plan {
					if i > 0 && plan[i-1].item == a.item {
						t.Fatalf("seed %d: the same kana is asked twice in a row at %d", seed, i)
					}
					if asked[a.item] == nil {
						asked[a.item] = map[common.QuestionType]int{}
					}
					asked[a.item][a.qt]++
					ways[a.qt]++
				}
				if d := ways[common.QuestionKanaMCQRomaji] - ways[common.QuestionKanaMCQKana]; d < -1 || d > 1 {
					t.Fatalf("seed %d: directions %v, want an even split", seed, ways)
				}
				if want := min(tc.kana, tc.wantTotal); len(asked) != want {
					t.Fatalf("seed %d: %d different kana asked, want %d", seed, len(asked), want)
				}
				for _, by := range asked {
					if tc.wantEach > 0 && (by[common.QuestionKanaMCQRomaji] != tc.wantEach/2 || by[common.QuestionKanaMCQKana] != tc.wantEach/2) {
						t.Fatalf("seed %d: a kana was asked %v, want %d each way", seed, by, tc.wantEach/2)
					}
				}
			}
		})
	}
}

func TestKanaQuizQuestionsAreMultipleChoiceBothWays(t *testing.T) {
	repo := newFakeRepo()
	svc := NewStartTestService(repo, NewCatalog(&fakeContent{}), NewQuestionBuilder(&fixedRand{}), &fixedRand{}, nil)
	got, err := svc.Start(t.Context(), testUserID, &StartTestRequest{Kind: "unit_quiz", Track: "hiragana", Scope: "a"})
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	ways := map[uuid.UUID]map[string]int{}
	for _, q := range got.Questions {
		if ways[q.ItemID] == nil {
			ways[q.ItemID] = map[string]int{}
		}
		ways[q.ItemID][q.QuestionType]++
		if len(q.Options) == 0 {
			t.Errorf("%s question for %s has no options", q.QuestionType, testChars[q.ItemID])
		}
	}
	for item, by := range ways {
		if by[string(common.QuestionKanaMCQRomaji)] != 2 || by[string(common.QuestionKanaMCQKana)] != 2 {
			t.Errorf("%s asked %v, want twice each way", testChars[item], by)
		}
	}
}
