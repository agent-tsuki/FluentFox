package content

import (
	"slices"

	"github.com/google/uuid"
)

func buildKanjiGroups(groups []KanjiGroup, units []KanjiUnit) []KanjiGroupDTO {
	unitIDs := make(map[uuid.UUID][]string, len(groups))
	for i := range units {
		unitIDs[units[i].GroupID] = append(unitIDs[units[i].GroupID], units[i].Slug)
	}
	out := make([]KanjiGroupDTO, 0, len(groups))
	for i := range groups {
		g := &groups[i]
		ids := unitIDs[g.ID]
		if ids == nil {
			ids = []string{}
		}
		out = append(out, KanjiGroupDTO{ID: g.Slug, Title: g.Title, Summary: g.Summary, UnitIDs: ids})
	}
	return out
}

// newParts is derived rather than stored: the parts a unit's kanji use that no
// earlier unit of the level used. kanji must be in course order.
func buildKanjiUnits(units []KanjiUnit, kanji []Kanji, children kanjiChildren) []KanjiUnitDTO {
	byUnit := make(map[uuid.UUID][]*Kanji, len(units))
	for i := range kanji {
		byUnit[kanji[i].UnitID] = append(byUnit[kanji[i].UnitID], &kanji[i])
	}
	seen := map[string]bool{}
	out := make([]KanjiUnitDTO, 0, len(units))
	for i := range units {
		u := &units[i]
		chars := []string{}
		var newParts []string
		for _, k := range byUnit[u.ID] {
			chars = append(chars, k.Char)
			for _, p := range children.parts[k.ID] {
				if !seen[p.Char] {
					seen[p.Char] = true
					newParts = append(newParts, p.Char)
				}
			}
		}
		out = append(out, KanjiUnitDTO{ID: u.Slug, Title: u.Title, Summary: u.Summary, Kanji: chars, NewParts: newParts})
	}
	return out
}

func buildKanjiIndexRow(k *Kanji, unitSlug string, children kanjiChildren) KanjiIndexRow {
	meaning := ""
	if len(k.Meanings) > 0 {
		meaning = k.Meanings[0]
	}
	parts := []string{}
	for _, p := range children.parts[k.ID] {
		parts = append(parts, p.Char)
	}
	return KanjiIndexRow{
		Char:       k.Char,
		Level:      levelID(k.Level),
		UnitID:     unitSlug,
		Meaning:    meaning,
		Meanings:   slices.Concat([]string{}, k.Meanings, []string(k.Synonyms)),
		KeyReading: k.KeyReading,
		Readings:   kanjiReadings(k, children.kun[k.ID]),
		Parts:      parts,
		Strokes:    k.Strokes,
		Freq:       k.Freq,
	}
}

// On'yomi first, then each distinct kun'yomi stem.
func kanjiReadings(k *Kanji, kun []KanjiKunReading) []string {
	readings := slices.Concat([]string{}, k.Onyomi)
	for i := range kun {
		if !slices.Contains(readings, kun[i].Kana) {
			readings = append(readings, kun[i].Kana)
		}
	}
	return readings
}

func buildKanjiEntry(k *Kanji, unitSlug string, children kanjiChildren) KanjiEntry {
	entry := KanjiEntry{
		Char:       k.Char,
		JLPT:       levelID(k.Level),
		UnitID:     unitSlug,
		Meanings:   slices.Concat([]string{}, k.Meanings),
		On:         slices.Concat([]string{}, k.Onyomi),
		Kun:        []KanjiKunReadingDTO{},
		KeyReading: KanjiKeyReading{Type: string(k.KeyReadingType), Kana: k.KeyReading},
		ReadingTip: k.ReadingTip,
		Parts:      []KanjiPartDTO{},
		Vocab:      []KanjiVocabDTO{},
		Grade:      k.Grade,
		Freq:       k.Freq,
		Strokes:    k.Strokes,
		Synonyms:   []string(k.Synonyms),
	}
	if k.Mnemonic != nil {
		entry.Mnemonic = &KanjiMnemonic{Text: *k.Mnemonic, Draft: k.MnemonicDraft}
	}
	for i := range children.kun[k.ID] {
		r := &children.kun[k.ID][i]
		var affix *string
		if r.Affix != nil {
			a := string(*r.Affix)
			affix = &a
		}
		entry.Kun = append(entry.Kun, KanjiKunReadingDTO{Kana: r.Kana, Okurigana: r.Okurigana, Affix: affix})
	}
	for _, p := range children.parts[k.ID] {
		entry.Parts = append(entry.Parts, KanjiPartDTO{Char: p.Char, Name: p.Name})
	}
	for i := range children.vocab[k.ID] {
		v := &children.vocab[k.ID][i]
		entry.Vocab = append(entry.Vocab, KanjiVocabDTO{Word: v.Word, Reading: v.Reading, Meaning: v.Meaning, IsKey: v.IsKey})
	}
	for _, l := range children.lookAlikes[k.ID] {
		entry.LookAlikes = append(entry.LookAlikes, KanjiLookAlikeDTO{Char: l.OtherChar, Note: l.Note})
	}
	return entry
}
