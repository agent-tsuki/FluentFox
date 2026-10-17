package content

import "github.com/fluentfox/api/internal/common"

// The structs below mirror content/kana/*.json and content/kanji/*.json,
// which scripts/content/convert-frontend.mjs writes.

type KanaFile struct {
	Script common.KanaScript `json:"script"`
	Groups []KanaFileGroup   `json:"groups"`
	Units  []KanaFileUnit    `json:"units"`
	Kana   []KanaFileEntry   `json:"kana"`
}

// Kana groups have no table; the file lists them for review and unit order only.
type KanaFileGroup struct {
	Slug     common.KanaGroup `json:"slug"`
	Title    string           `json:"title"`
	Summary  string           `json:"summary"`
	Position int              `json:"position"`
}

type KanaFileUnit struct {
	Slug        string           `json:"slug"`
	Group       common.KanaGroup `json:"group"`
	Title       string           `json:"title"`
	Summary     string           `json:"summary"`
	Position    int              `json:"position"`
	YoutubeID   *string          `json:"youtube_id"`
	VideoCredit *string          `json:"video_credit"`
}

type KanaFileEntry struct {
	Slug        string              `json:"slug"`
	Character   string              `json:"character"`
	Romaji      string              `json:"romaji"`
	AltRomaji   []string            `json:"alt_romaji"`
	Group       common.KanaGroup    `json:"group"`
	Unit        string              `json:"unit"`
	Position    int                 `json:"position"`
	StrokeCount *int                `json:"stroke_count"`
	LookAlikes  []KanaFileLookAlike `json:"look_alikes"`
}

type KanaFileLookAlike struct {
	Slug string  `json:"slug"`
	Note *string `json:"note"`
}

type KanjiFile struct {
	Level   common.JLPTLevel `json:"level"`
	Groups  []KanjiFileGroup `json:"groups"`
	Units   []KanjiFileUnit  `json:"units"`
	Parts   []KanjiFilePart  `json:"parts"`
	Entries []KanjiFileEntry `json:"entries"`
}

type KanjiFileGroup struct {
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Position int    `json:"position"`
}

type KanjiFileUnit struct {
	Slug     string `json:"slug"`
	Group    string `json:"group"`
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	Position int    `json:"position"`
}

type KanjiFilePart struct {
	Char string `json:"char"`
	Name string `json:"name"`
}

type KanjiFileEntry struct {
	Char       string               `json:"char"`
	Unit       string               `json:"unit"`
	Position   int                  `json:"position"`
	Meanings   []string             `json:"meanings"`
	Synonyms   []string             `json:"synonyms"`
	Onyomi     []string             `json:"onyomi"`
	Kun        []KanjiFileKun       `json:"kun"`
	KeyReading KanjiFileKeyReading  `json:"key_reading"`
	ReadingTip *string              `json:"reading_tip"`
	Mnemonic   *KanjiFileMnemonic   `json:"mnemonic"`
	Strokes    int                  `json:"strokes"`
	Grade      *int                 `json:"grade"`
	Freq       *int                 `json:"freq"`
	Parts      []string             `json:"parts"`
	Vocab      []KanjiFileVocab     `json:"vocab"`
	LookAlikes []KanjiFileLookAlike `json:"look_alikes"`
}

type KanjiFileKun struct {
	Kana      string           `json:"kana"`
	Okurigana *string          `json:"okurigana"`
	Affix     *common.KunAffix `json:"affix"`
}

type KanjiFileKeyReading struct {
	Type common.ReadingType `json:"type"`
	Kana string             `json:"kana"`
}

type KanjiFileMnemonic struct {
	Text  string `json:"text"`
	Draft bool   `json:"draft"`
}

type KanjiFileVocab struct {
	Word    string `json:"word"`
	Reading string `json:"reading"`
	Meaning string `json:"meaning"`
	IsKey   bool   `json:"is_key"`
}

type KanjiFileLookAlike struct {
	Char string  `json:"char"`
	Note *string `json:"note"`
}
