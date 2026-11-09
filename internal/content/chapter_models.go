package content

import (
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/database"
)

type GrammarChapter struct {
	ID       uuid.UUID        `gorm:"type:uuid;primaryKey"`
	Level    common.JLPTLevel `gorm:"column:level;type:jlpt_level;not null"`
	Slug     string           `gorm:"column:slug;not null"`
	Number   int              `gorm:"column:number;not null"`
	Title    string           `gorm:"column:title;not null"`
	Position int              `gorm:"column:position;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (GrammarChapter) TableName() string { return "grammar_chapter" }

type ChapterWord struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	ChapterID uuid.UUID `gorm:"column:chapter_id;type:uuid;not null"`
	Position  int       `gorm:"column:position;not null"`
	Word      string    `gorm:"column:word;not null"`
	Kana      string    `gorm:"column:kana;not null"`
	Romaji    string    `gorm:"column:romaji;not null"`
	Meaning   string    `gorm:"column:meaning;not null"`
	Hint      *string   `gorm:"column:hint"`
	Say       *string   `gorm:"column:say"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (ChapterWord) TableName() string { return "chapter_word" }

type GrammarSentence struct {
	ID          uuid.UUID          `gorm:"type:uuid;primaryKey"`
	ChapterID   uuid.UUID          `gorm:"column:chapter_id;type:uuid;not null"`
	Position    int                `gorm:"column:position;not null"`
	Sentence    string             `gorm:"column:sentence;not null"`
	Answer      string             `gorm:"column:answer;not null"`
	Distractors database.TextArray `gorm:"column:distractors;type:text[];not null"`
	AlsoAccept  database.TextArray `gorm:"column:also_accept;type:text[];not null"`
	Romaji      database.TextArray `gorm:"column:romaji;type:text[];not null"`
	Translation string             `gorm:"column:translation;not null"`
	Explanation string             `gorm:"column:explanation;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (GrammarSentence) TableName() string { return "grammar_sentence" }

// The structs below mirror content/chapters/*.json, which
// scripts/content/convert-frontend.mjs writes.

type ChapterFile struct {
	Level    common.JLPTLevel   `json:"level"`
	Chapters []ChapterFileEntry `json:"chapters"`
}

type ChapterFileEntry struct {
	Slug      string                `json:"slug"`
	Number    int                   `json:"number"`
	Title     string                `json:"title"`
	Position  int                   `json:"position"`
	Words     []ChapterFileWord     `json:"words"`
	Sentences []ChapterFileSentence `json:"sentences"`
}

type ChapterFileWord struct {
	Word    string  `json:"word"`
	Kana    string  `json:"kana"`
	Romaji  string  `json:"romaji"`
	Meaning string  `json:"meaning"`
	Hint    *string `json:"hint"`
	// Say is what to speak where the engine misreads the kana alone (e.g. はは).
	Say *string `json:"say,omitempty"`
}

type ChapterFileSentence struct {
	Sentence    string   `json:"sentence"`
	Answer      string   `json:"answer"`
	Distractors []string `json:"distractors"`
	AlsoAccept  []string `json:"also_accept"`
	Romaji      []string `json:"romaji"`
	Translation string   `json:"translation"`
	Explanation string   `json:"explanation"`
}
