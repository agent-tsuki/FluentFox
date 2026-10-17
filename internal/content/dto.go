package content

// Payload field names are camelCase on purpose: they are the frontend's
// kana course types and src/types/kanji.ts, which the UI already consumes.
// Only strokes_base_url is new.

type KanaScriptInput struct {
	Script string `path:"script" example:"hiragana" doc:"Kana script: hiragana or katakana"`
}

type KanjiLevelInput struct {
	Level string `path:"level" example:"n5" doc:"Lower-case JLPT level id; only n5 in v1"`
}

type KanjiCharInput struct {
	Char string `path:"char" example:"日" doc:"The kanji character"`
}

type KanaCourseResponse struct {
	Script         string         `json:"script"           doc:"hiragana or katakana"`
	StrokesBaseURL string         `json:"strokes_base_url" doc:"Base URL of the stroke JSON files"`
	Groups         []KanaGroupDTO `json:"groups"           doc:"Unit groups in path order"`
	Units          []KanaUnitDTO  `json:"units"            doc:"Units in path order"`
	Kana           []KanaChar     `json:"kana"             doc:"Every kana of the script in path order"`
}

type KanaGroupDTO struct {
	ID      string   `json:"id"      doc:"basic, dakuten, combo or foreign"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	UnitIDs []string `json:"unitIds" doc:"Unit ids in path order"`
}

type KanaUnitDTO struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Summary     string   `json:"summary"`
	KanaIDs     []string `json:"kanaIds"               doc:"Kana ids taught in this unit, in order"`
	YoutubeID   *string  `json:"youtubeId,omitempty"`
	VideoCredit *string  `json:"videoCredit,omitempty"`
}

// KanaChar is the frontend KanaChar plus the fields the course needs.
type KanaChar struct {
	ID          string             `json:"id"                    doc:"Chart id, the same in both scripts (e.g. shi)"`
	Character   string             `json:"character"`
	Romaji      string             `json:"romaji"`
	Type        string             `json:"type"                  doc:"basic, dakuten, compound or foreign"`
	IsCompound  bool               `json:"isCompound,omitempty"  doc:"Two-part kana such as きゃ"`
	AltRomaji   []string           `json:"altRomaji"             doc:"Other accepted spellings (si, tu, hu, zi ...)"`
	GroupID     string             `json:"groupId"`
	UnitID      string             `json:"unitId"`
	Position    int                `json:"position"`
	StrokeCount *int               `json:"strokeCount,omitempty"`
	LookAlikes  []KanaLookAlikeDTO `json:"lookAlikes"            doc:"Easily confused kana of the same script"`
}

type KanaLookAlikeDTO struct {
	ID   string  `json:"id"`
	Char string  `json:"char"`
	Note *string `json:"note,omitempty" doc:"How to tell them apart"`
}

type KanjiLevelIndex struct {
	Level          string          `json:"level"            doc:"Lower-case level id, e.g. n5"`
	StrokesBaseURL string          `json:"strokes_base_url" doc:"Base URL of the stroke JSON files"`
	Units          []KanjiUnitDTO  `json:"units"`
	Groups         []KanjiGroupDTO `json:"groups"`
	Rows           []KanjiIndexRow `json:"rows"`
}

type KanjiUnitDTO struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Summary  string   `json:"summary"`
	Kanji    []string `json:"kanji"              doc:"Kanji taught in this unit, in order"`
	NewParts []string `json:"newParts,omitempty" doc:"Parts this unit uses for the first time in the level"`
}

type KanjiGroupDTO struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	UnitIDs []string `json:"unitIds"`
}

type KanjiIndexRow struct {
	Char       string   `json:"char"`
	Level      string   `json:"level"`
	UnitID     string   `json:"unitId"`
	Meaning    string   `json:"meaning"`
	Meanings   []string `json:"meanings"       doc:"Meanings, then synonyms"`
	KeyReading string   `json:"keyReading"`
	Readings   []string `json:"readings"       doc:"On'yomi then kun'yomi stems"`
	Parts      []string `json:"parts"`
	Strokes    int      `json:"strokes"`
	Freq       *int     `json:"freq,omitempty"`
}

type KanjiLevelDetail struct {
	Level          string       `json:"level"`
	StrokesBaseURL string       `json:"strokes_base_url" doc:"Base URL of the stroke JSON files"`
	Entries        []KanjiEntry `json:"entries"`
}

type KanjiEntryResponse struct {
	KanjiEntry
	StrokesBaseURL string `json:"strokes_base_url" doc:"Base URL of the stroke JSON files"`
}

type KanjiEntry struct {
	Char       string               `json:"char"`
	JLPT       string               `json:"jlpt"`
	UnitID     string               `json:"unitId"`
	Meanings   []string             `json:"meanings"`
	On         []string             `json:"on"`
	Kun        []KanjiKunReadingDTO `json:"kun"`
	KeyReading KanjiKeyReading      `json:"keyReading"`
	ReadingTip *string              `json:"readingTip,omitempty"`
	Parts      []KanjiPartDTO       `json:"parts"`
	Vocab      []KanjiVocabDTO      `json:"vocab"`
	Grade      *int                 `json:"grade,omitempty"`
	Freq       *int                 `json:"freq,omitempty"`
	Strokes    int                  `json:"strokes"`
	Mnemonic   *KanjiMnemonic       `json:"mnemonic,omitempty"`
	Synonyms   []string             `json:"synonyms,omitempty"`
	LookAlikes []KanjiLookAlikeDTO  `json:"lookAlikes,omitempty"`
}

type KanjiKunReadingDTO struct {
	Kana      string  `json:"kana"`
	Okurigana *string `json:"okurigana,omitempty"`
	Affix     *string `json:"affix,omitempty"`
}

type KanjiKeyReading struct {
	Type string `json:"type" doc:"on or kun"`
	Kana string `json:"kana"`
}

type KanjiPartDTO struct {
	Char string `json:"char"`
	Name string `json:"name"`
}

type KanjiVocabDTO struct {
	Word    string `json:"word"`
	Reading string `json:"reading"`
	Meaning string `json:"meaning"`
	IsKey   bool   `json:"isKey,omitempty"`
}

type KanjiMnemonic struct {
	Text  string `json:"text"`
	Draft bool   `json:"draft"`
}

type KanjiLookAlikeDTO struct {
	Char string  `json:"char"`
	Note *string `json:"note,omitempty"`
}
