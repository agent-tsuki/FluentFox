package content

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
)

// ChapterExerciseKind maps to the chapter_exercise_kind PostgreSQL enum.
type ChapterExerciseKind string

const (
	ChapterExerciseTrueFalse ChapterExerciseKind = "true_false"
	ChapterExerciseTransform ChapterExerciseKind = "transform"
	ChapterExercisePicture   ChapterExerciseKind = "picture"
	ChapterExerciseWord      ChapterExerciseKind = "word_sentence"
	ChapterExerciseOrder     ChapterExerciseKind = "order"
	ChapterExerciseListen    ChapterExerciseKind = "listen"
	ChapterExerciseReply     ChapterExerciseKind = "reply"
	ChapterExerciseReading   ChapterExerciseKind = "reading"
)

// ChapterExercise is one practice item; Data is the JSON of its kind's item.
type ChapterExercise struct {
	ID        uuid.UUID           `gorm:"type:uuid;primaryKey"`
	ChapterID uuid.UUID           `gorm:"column:chapter_id;type:uuid;not null"`
	Position  int                 `gorm:"column:position;not null"`
	Kind      ChapterExerciseKind `gorm:"column:kind;not null"`
	Data      json.RawMessage     `gorm:"column:data;type:jsonb;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (ChapterExercise) TableName() string { return "chapter_exercise" }

// ExerciseFile is content/exercises/*.json.
type ExerciseFile struct {
	Level    common.JLPTLevel      `json:"level"`
	Chapters []ExerciseFileChapter `json:"chapters"`
}

type ExerciseFileChapter struct {
	Slug       string          `json:"slug"`
	TrueFalse  []TrueFalseItem `json:"true_false"`
	Transforms []TransformItem `json:"transforms"`
	Pictures   []PictureItem   `json:"pictures"`
	Words      []WordSentence  `json:"word_sentences"`
	Orders     []OrderItem     `json:"order"`
	Listens    []ListenItem    `json:"listen"`
	Replies    []ReplyItem     `json:"reply"`
	Readings   []ReadingItem   `json:"reading"`
}

// TrueFalseItem: read a short exchange, say whether the statement is true.
type TrueFalseItem struct {
	Lines       []string `json:"lines"       doc:"The exchange, one line per turn"`
	Statement   string   `json:"statement"   doc:"An English claim about it"`
	Answer      bool     `json:"answer"`
	Explanation string   `json:"explanation"`
}

// TransformItem: change a sentence as asked, typing the missing part.
type TransformItem struct {
	Source      string   `json:"source"      doc:"The sentence to change"`
	Task        string   `json:"task"        doc:"What to do, in English"`
	Sentence    string   `json:"sentence"    doc:"The result, with {} for the part to type"`
	Answer      string   `json:"answer"`
	AlsoAccept  []string `json:"also_accept" doc:"Other accepted answers, e.g. the kana spelling"`
	Explanation string   `json:"explanation"`
}

// WordSentence: drop a chapter word into the sentence's blank.
type WordSentence struct {
	Sentence    string `json:"sentence"    doc:"The sentence, with {} for the word"`
	Answer      string `json:"answer"      doc:"The chapter word, exactly as taught"`
	Translation string `json:"translation"`
}

// PictureItem: choose the word that matches the picture.
type PictureItem struct {
	Image       string   `json:"image"       doc:"Path of the picture in the web app"`
	Alt         string   `json:"alt"`
	Sentence    string   `json:"sentence"    doc:"The sentence, with {} for the gap"`
	Answer      string   `json:"answer"`
	Options     []string `json:"options"     doc:"Words to choose from, including the answer"`
	Translation string   `json:"translation"`
}

// OrderItem: put the words of a sentence in order.
type OrderItem struct {
	Words       []string `json:"words"       doc:"The sentence's words, in the right order"`
	Translation string   `json:"translation"`
}

// ListenItem: hear a line and choose what it means.
type ListenItem struct {
	Text        string   `json:"text"          doc:"The line, written with readings"`
	Translation string   `json:"translation"`
	Wrong       []string `json:"wrong"         doc:"Other meanings to choose from"`
	Voice       string   `json:"voice"         doc:"Cast key for the speech engine"`
	Say         string   `json:"say,omitempty" doc:"What to speak where the engine misreads the text"`
}

// ReplyItem: choose the natural reply to a line.
type ReplyItem struct {
	Prompt      string   `json:"prompt"`
	Answer      string   `json:"answer"`
	Wrong       []string `json:"wrong"`
	Explanation string   `json:"explanation" doc:"The reply in English"`
}

// ReadingItem: read a short passage and answer a question about it.
type ReadingItem struct {
	Lines    []string `json:"lines"`
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Wrong    []string `json:"wrong"`
}
