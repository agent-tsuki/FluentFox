package learn

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

func TestUnitProgressMarkStep(t *testing.T) {
	earlier := testNow.Add(-48 * time.Hour)
	tests := []struct {
		name      string
		in        UnitProgressInput
		setup     func(r *fakeRepo)
		wantUnit  uuid.UUID
		wantTrack string
		wantAt    time.Time
		wantErr   error
	}{
		{
			name:     "a slug found in one track needs no track",
			in:       UnitProgressInput{Type: "kana", ID: "za", Body: UnitProgressRequest{Step: "watched"}},
			wantUnit: unitZa, wantTrack: "hiragana", wantAt: testNow,
		},
		{
			name:     "the track picks between hiragana and katakana",
			in:       UnitProgressInput{Type: "kana", ID: "a", Track: "katakana", Body: UnitProgressRequest{Step: "traced"}},
			wantUnit: unitKA, wantTrack: "katakana", wantAt: testNow,
		},
		{
			name:     "a unit id works as well as a slug",
			in:       UnitProgressInput{Type: "kanji", ID: unitK2.String(), Body: UnitProgressRequest{Step: "written"}},
			wantUnit: unitK2, wantTrack: "kanji_n5", wantAt: testNow,
		},
		{
			name: "doing a step again keeps the first time",
			in:   UnitProgressInput{Type: "kanji", ID: "u01", Body: UnitProgressRequest{Step: "practised"}},
			setup: func(r *fakeRepo) {
				r.unitProgress[unitK1] = &UnitProgress{UserID: testUserID, UnitType: common.UnitTypeKanji, UnitID: unitK1, PractisedAt: &earlier}
			},
			wantUnit: unitK1, wantTrack: "kanji_n5", wantAt: earlier,
		},
		{
			name:    "a slug in two tracks without a track is ambiguous",
			in:      UnitProgressInput{Type: "kana", ID: "a", Body: UnitProgressRequest{Step: "watched"}},
			wantErr: exceptions.ErrUnitAmbiguous,
		},
		{
			name:    "a kanji slug is not a kana unit",
			in:      UnitProgressInput{Type: "kana", ID: "u01", Body: UnitProgressRequest{Step: "watched"}},
			wantErr: exceptions.ErrUnitNotFound,
		},
		{
			name:    "an unknown step is rejected",
			in:      UnitProgressInput{Type: "kana", ID: "za", Body: UnitProgressRequest{Step: "sung"}},
			wantErr: exceptions.ErrUnitProgressStepInvalid,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			if tc.setup != nil {
				tc.setup(repo)
			}
			svc := NewUnitProgressService(repo, NewCatalog(&fakeContent{}), zap.NewNop())
			svc.now = func() time.Time { return testNow }
			got, err := svc.MarkStep(context.Background(), testUserID, &tc.in)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if got.UnitID != tc.wantUnit || got.Track != tc.wantTrack || got.UnitType != tc.in.Type {
				t.Errorf("unit = %s %s %s, want %s in %s", got.UnitType, got.UnitID, got.Track, tc.wantUnit, tc.wantTrack)
			}
			at := map[string]*time.Time{
				"watched": got.WatchedAt, "traced": got.TracedAt, "written": got.WrittenAt, "practised": got.PractisedAt,
			}[tc.in.Body.Step]
			if at == nil || !at.Equal(tc.wantAt) {
				t.Errorf("%s at = %v, want %v", tc.in.Body.Step, at, tc.wantAt)
			}
		})
	}
}

func TestUnitProgressList(t *testing.T) {
	pct := 90
	tests := []struct {
		name      string
		track     string
		wantUnits []uuid.UUID
		wantErr   error
	}{
		{name: "one track in path order", track: "kanji_n5", wantUnits: []uuid.UUID{unitK1, unitK2}},
		{name: "every track when none is given", wantUnits: []uuid.UUID{unitZa, unitA, unitKA, unitK1, unitK2}},
		{name: "an unknown track is rejected", track: "kanji_n1", wantErr: exceptions.ErrLearningTrackInvalid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.unitProgress[unitK1] = &UnitProgress{UnitType: common.UnitTypeKanji, UnitID: unitK1, QuizBestPct: &pct}
			svc := NewUnitProgressService(repo, NewCatalog(&fakeContent{}), zap.NewNop())
			got, err := svc.List(context.Background(), testUserID, tc.track)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if len(got.Units) != len(tc.wantUnits) {
				t.Fatalf("units = %+v, want %d", got.Units, len(tc.wantUnits))
			}
			for i, u := range got.Units {
				if u.UnitID != tc.wantUnits[i] {
					t.Errorf("unit %d = %s, want %s", i, u.UnitSlug, tc.wantUnits[i])
				}
				hasQuiz := u.QuizBestPct != nil && *u.QuizBestPct == pct
				if hasQuiz != (u.UnitID == unitK1) || u.WatchedAt != nil {
					t.Errorf("unit %s progress = %+v, want only u01's quiz score", u.UnitSlug, u)
				}
			}
		})
	}
}

func TestKanjiNoteService(t *testing.T) {
	saved := testNow.Add(-time.Hour)
	tests := []struct {
		name     string
		char     string
		put      *KanjiNoteRequest
		setup    func(r *fakeRepo)
		wantText string
		wantErr  error
		check    func(t *testing.T, r *fakeRepo, got *KanjiNoteResponse)
	}{
		{
			name: "no note reads as empty text", char: "日",
			check: func(t *testing.T, _ *fakeRepo, got *KanjiNoteResponse) {
				if got.UpdatedAt != nil {
					t.Errorf("updated at = %v, want null", got.UpdatedAt)
				}
			},
		},
		{
			name: "a stored note is returned", char: "日",
			setup: func(r *fakeRepo) {
				r.notes[kanjiDay] = &KanjiNote{UserID: testUserID, KanjiID: kanjiDay, Text: "sun over the horizon", UpdatedAt: saved}
			},
			wantText: "sun over the horizon",
		},
		{
			name: "saving trims the text and stamps the time", char: "一", put: &KanjiNoteRequest{Text: "  one line  "},
			wantText: "one line",
			check: func(t *testing.T, r *fakeRepo, got *KanjiNoteResponse) {
				if n := r.notes[kanjiOne]; n == nil || n.Text != "one line" || !n.UpdatedAt.Equal(testNow) {
					t.Errorf("stored = %+v, want the trimmed text at now", n)
				}
				if got.UpdatedAt == nil || !got.UpdatedAt.Equal(testNow) {
					t.Errorf("updated at = %v, want now", got.UpdatedAt)
				}
			},
		},
		{
			name: "empty text deletes the note", char: "日", put: &KanjiNoteRequest{Text: "   "},
			setup: func(r *fakeRepo) {
				r.notes[kanjiDay] = &KanjiNote{UserID: testUserID, KanjiID: kanjiDay, Text: "old", UpdatedAt: saved}
			},
			check: func(t *testing.T, r *fakeRepo, got *KanjiNoteResponse) {
				if r.notes[kanjiDay] != nil || got.UpdatedAt != nil {
					t.Errorf("note = %+v, response = %+v, want it deleted", r.notes[kanjiDay], got)
				}
			},
		},
		{
			name: "1000 characters is the limit", char: "日", put: &KanjiNoteRequest{Text: strings.Repeat("日", common.KANJI_NOTE_MAX_CHARS+1)},
			wantErr: exceptions.ErrKanjiNoteTooLong,
		},
		{name: "an unknown kanji is not found", char: "猫", wantErr: exceptions.ErrKanjiNotFound},
		{
			name: "a storage error is returned", char: "日", put: &KanjiNoteRequest{Text: "x"},
			setup:   func(r *fakeRepo) { r.err = errBoom },
			wantErr: errBoom,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			if tc.setup != nil {
				tc.setup(repo)
			}
			svc := NewKanjiNoteService(repo, zap.NewNop())
			svc.now = func() time.Time { return testNow }
			var got KanjiNoteResponse
			var err error
			if tc.put != nil {
				got, err = svc.Put(context.Background(), testUserID, tc.char, tc.put)
			} else {
				got, err = svc.Get(context.Background(), testUserID, tc.char)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				return
			}
			if got.Text != tc.wantText {
				t.Errorf("text = %q, want %q", got.Text, tc.wantText)
			}
			if tc.check != nil {
				tc.check(t, repo, &got)
			}
		})
	}
}
