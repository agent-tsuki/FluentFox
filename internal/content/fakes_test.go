package content

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
)

type fakeKanaReader struct {
	units      []KanaUnit
	kana       []Kana
	lookAlikes []KanaLookAlikeRow
	err        error

	gotScript common.KanaScript
}

func (f *fakeKanaReader) ListKanaUnits(_ context.Context, script common.KanaScript) ([]KanaUnit, error) {
	f.gotScript = script
	return f.units, f.err
}

func (f *fakeKanaReader) ListKana(context.Context, common.KanaScript) ([]Kana, error) {
	return f.kana, nil
}

func (f *fakeKanaReader) ListKanaLookAlikes(context.Context, common.KanaScript) ([]KanaLookAlikeRow, error) {
	return f.lookAlikes, nil
}

type fakeKanjiReader struct {
	groups     []KanjiGroup
	units      []KanjiUnit
	kanji      []Kanji
	kun        []KanjiKunReading
	parts      []KanjiPartRow
	vocab      []KanjiVocab
	lookAlikes []KanjiLookAlikeRow
	listErr    error

	calls []string
}

func (f *fakeKanjiReader) ListKanjiGroups(context.Context, common.JLPTLevel) ([]KanjiGroup, error) {
	return f.groups, f.listErr
}

func (f *fakeKanjiReader) ListKanjiUnits(context.Context, common.JLPTLevel) ([]KanjiUnit, error) {
	return f.units, f.listErr
}

func (f *fakeKanjiReader) ListKanji(context.Context, common.JLPTLevel) ([]Kanji, error) {
	return f.kanji, f.listErr
}

func (f *fakeKanjiReader) GetKanjiByChar(_ context.Context, char string) (Kanji, error) {
	for _, k := range f.kanji {
		if k.Char == char {
			return k, nil
		}
	}
	return Kanji{}, gorm.ErrRecordNotFound
}

func (f *fakeKanjiReader) GetKanjiUnit(_ context.Context, id uuid.UUID) (KanjiUnit, error) {
	for _, u := range f.units {
		if u.ID == id {
			return u, nil
		}
	}
	return KanjiUnit{}, gorm.ErrRecordNotFound
}

func (f *fakeKanjiReader) ListKanjiKunReadings(context.Context, []uuid.UUID) ([]KanjiKunReading, error) {
	f.calls = append(f.calls, "kun")
	return f.kun, nil
}

func (f *fakeKanjiReader) ListKanjiParts(context.Context, []uuid.UUID) ([]KanjiPartRow, error) {
	f.calls = append(f.calls, "parts")
	return f.parts, nil
}

func (f *fakeKanjiReader) ListKanjiVocab(context.Context, []uuid.UUID) ([]KanjiVocab, error) {
	f.calls = append(f.calls, "vocab")
	return f.vocab, nil
}

func (f *fakeKanjiReader) ListKanjiLookAlikes(context.Context, []uuid.UUID) ([]KanjiLookAlikeRow, error) {
	f.calls = append(f.calls, "lookAlikes")
	return f.lookAlikes, nil
}

// fakeWriter records upserts in memory. Transaction passes a nil *gorm.DB,
// which the fake never touches, and returns fn's error like a rollback would.
type fakeWriter struct {
	ids map[string]uuid.UUID

	kanaUnits       []KanaUnit
	kana            []Kana
	kanaLookAlikes  []KanaLookAlike
	kanjiGroups     []KanjiGroup
	kanjiUnits      []KanjiUnit
	kanji           []Kanji
	kun             []KanjiKunReading
	vocab           []KanjiVocab
	partLinks       []KanjiPartLink
	kanjiLookAlikes []KanjiLookAlike

	existingKanji map[string]uuid.UUID
}

func newFakeWriter() *fakeWriter {
	return &fakeWriter{ids: map[string]uuid.UUID{}, existingKanji: map[string]uuid.UUID{}}
}

func (f *fakeWriter) id(key string) uuid.UUID {
	if id, ok := f.ids[key]; ok {
		return id
	}
	id := uuid.New()
	f.ids[key] = id
	return id
}

func (f *fakeWriter) Transaction(_ context.Context, fn func(*gorm.DB) error) error {
	return fn(nil)
}

func (f *fakeWriter) UpsertKanaUnit(_ context.Context, _ *gorm.DB, u *KanaUnit) (uuid.UUID, error) {
	f.kanaUnits = append(f.kanaUnits, *u)
	return f.id("kana_unit:" + u.Slug), nil
}

func (f *fakeWriter) UpsertKana(_ context.Context, _ *gorm.DB, k *Kana) (uuid.UUID, error) {
	f.kana = append(f.kana, *k)
	return f.id("kana:" + k.Slug), nil
}

func (f *fakeWriter) UpsertKanaLookAlike(_ context.Context, _ *gorm.DB, la *KanaLookAlike) error {
	f.kanaLookAlikes = append(f.kanaLookAlikes, *la)
	return nil
}

func (f *fakeWriter) UpsertKanjiGroup(_ context.Context, _ *gorm.DB, g *KanjiGroup) (uuid.UUID, error) {
	f.kanjiGroups = append(f.kanjiGroups, *g)
	return f.id("kanji_group:" + g.Slug), nil
}

func (f *fakeWriter) UpsertKanjiUnit(_ context.Context, _ *gorm.DB, u *KanjiUnit) (uuid.UUID, error) {
	f.kanjiUnits = append(f.kanjiUnits, *u)
	return f.id("kanji_unit:" + u.Slug), nil
}

func (f *fakeWriter) UpsertKanjiPart(_ context.Context, _ *gorm.DB, p *KanjiPart) (uuid.UUID, error) {
	return f.id("part:" + p.Char), nil
}

func (f *fakeWriter) UpsertKanji(_ context.Context, _ *gorm.DB, k *Kanji) (uuid.UUID, error) {
	f.kanji = append(f.kanji, *k)
	id := f.id("kanji:" + k.Char)
	f.existingKanji[k.Char] = id
	return id, nil
}

func (f *fakeWriter) UpsertKanjiKunReading(_ context.Context, _ *gorm.DB, r *KanjiKunReading) error {
	f.kun = append(f.kun, *r)
	return nil
}

func (f *fakeWriter) UpsertKanjiVocab(_ context.Context, _ *gorm.DB, v *KanjiVocab) error {
	f.vocab = append(f.vocab, *v)
	return nil
}

func (f *fakeWriter) UpsertKanjiPartLink(_ context.Context, _ *gorm.DB, l *KanjiPartLink) error {
	f.partLinks = append(f.partLinks, *l)
	return nil
}

func (f *fakeWriter) FindKanjiIDsByChars(_ context.Context, _ *gorm.DB, chars []string) (map[string]uuid.UUID, error) {
	out := map[string]uuid.UUID{}
	for _, c := range chars {
		if id, ok := f.existingKanji[c]; ok {
			out[c] = id
		}
	}
	return out, nil
}

func (f *fakeWriter) UpsertKanjiLookAlike(_ context.Context, _ *gorm.DB, la *KanjiLookAlike) error {
	f.kanjiLookAlikes = append(f.kanjiLookAlikes, *la)
	return nil
}
