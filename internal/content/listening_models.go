package content

import (
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/database"
)

type ListeningPassageKind string

const (
	ListeningDialogue  ListeningPassageKind = "dialogue"
	ListeningMonologue ListeningPassageKind = "monologue"
)

// dialogueSpeakers is how many people talk in a dialogue: one per voice.
const dialogueSpeakers = 2

type ChapterPassage struct {
	ID        uuid.UUID            `gorm:"type:uuid;primaryKey"`
	ChapterID uuid.UUID            `gorm:"column:chapter_id;type:uuid;not null"`
	Position  int                  `gorm:"column:position;not null"`
	Kind      ListeningPassageKind `gorm:"column:kind;type:listening_passage_kind;not null"`
	Title     string               `gorm:"column:title;not null"`
	Speakers  database.TextArray   `gorm:"column:speakers;type:text[];not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (ChapterPassage) TableName() string { return "chapter_passage" }

type ChapterPassageLine struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	PassageID   uuid.UUID `gorm:"column:passage_id;type:uuid;not null"`
	Position    int       `gorm:"column:position;not null"`
	Speaker     *int      `gorm:"column:speaker"`
	Text        string    `gorm:"column:text;not null"`
	Translation string    `gorm:"column:translation;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (ChapterPassageLine) TableName() string { return "chapter_passage_line" }

// The structs below mirror content/listening/*.json (written by hand).

type ListeningFile struct {
	Level    common.JLPTLevel       `json:"level"`
	Chapters []ListeningFileChapter `json:"chapters"`
}

type ListeningFileChapter struct {
	Slug     string                 `json:"slug"`
	Passages []ListeningFilePassage `json:"passages"`
}

type ListeningFilePassage struct {
	Kind     ListeningPassageKind `json:"kind"`
	Title    string               `json:"title"`
	Speakers []string             `json:"speakers"`
	Lines    []ListeningFileLine  `json:"lines"`
}

type ListeningFileLine struct {
	Speaker     *int   `json:"speaker"`
	Text        string `json:"text"`
	Translation string `json:"translation"`
}
