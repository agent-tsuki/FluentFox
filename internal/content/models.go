package content

import (
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/database"
)

type KanaUnit struct {
	ID          uuid.UUID         `gorm:"type:uuid;primaryKey"`
	Script      common.KanaScript `gorm:"column:script;type:kana_script;not null"`
	GroupKey    common.KanaGroup  `gorm:"column:group_key;type:kana_group;not null"`
	Slug        string            `gorm:"column:slug;not null"`
	Title       string            `gorm:"column:title;not null"`
	Summary     string            `gorm:"column:summary;not null"`
	Position    int               `gorm:"column:position;not null"`
	YoutubeID   *string           `gorm:"column:youtube_id"`
	VideoCredit *string           `gorm:"column:video_credit"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (KanaUnit) TableName() string { return "kana_unit" }

type Kana struct {
	ID          uuid.UUID          `gorm:"type:uuid;primaryKey"`
	Script      common.KanaScript  `gorm:"column:script;type:kana_script;not null"`
	Slug        string             `gorm:"column:slug;not null"`
	Character   string             `gorm:"column:character;not null"`
	Romaji      string             `gorm:"column:romaji;not null"`
	AltRomaji   database.TextArray `gorm:"column:alt_romaji;type:text[];not null"`
	GroupKey    common.KanaGroup   `gorm:"column:group_key;type:kana_group;not null"`
	UnitID      uuid.UUID          `gorm:"column:unit_id;type:uuid;not null"`
	Position    int                `gorm:"column:position;not null"`
	StrokeCount *int               `gorm:"column:stroke_count"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Kana) TableName() string { return "kana" }

type KanaLookAlike struct {
	KanaID      uuid.UUID `gorm:"column:kana_id;type:uuid;primaryKey"`
	OtherKanaID uuid.UUID `gorm:"column:other_kana_id;type:uuid;primaryKey"`
	Note        *string   `gorm:"column:note"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (KanaLookAlike) TableName() string { return "kana_look_alike" }

type KanjiGroup struct {
	ID       uuid.UUID        `gorm:"type:uuid;primaryKey"`
	Level    common.JLPTLevel `gorm:"column:level;type:jlpt_level;not null"`
	Slug     string           `gorm:"column:slug;not null"`
	Title    string           `gorm:"column:title;not null"`
	Summary  string           `gorm:"column:summary;not null"`
	Position int              `gorm:"column:position;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (KanjiGroup) TableName() string { return "kanji_group" }

type KanjiUnit struct {
	ID       uuid.UUID        `gorm:"type:uuid;primaryKey"`
	GroupID  uuid.UUID        `gorm:"column:group_id;type:uuid;not null"`
	Level    common.JLPTLevel `gorm:"column:level;type:jlpt_level;not null"`
	Slug     string           `gorm:"column:slug;not null"`
	Title    string           `gorm:"column:title;not null"`
	Summary  string           `gorm:"column:summary;not null"`
	Position int              `gorm:"column:position;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (KanjiUnit) TableName() string { return "kanji_unit" }

type Kanji struct {
	ID             uuid.UUID          `gorm:"type:uuid;primaryKey"`
	Char           string             `gorm:"column:char;not null"`
	Level          common.JLPTLevel   `gorm:"column:level;type:jlpt_level;not null"`
	UnitID         uuid.UUID          `gorm:"column:unit_id;type:uuid;not null"`
	Position       int                `gorm:"column:position;not null"`
	Meanings       database.TextArray `gorm:"column:meanings;type:text[];not null"`
	Synonyms       database.TextArray `gorm:"column:synonyms;type:text[];not null"`
	Onyomi         database.TextArray `gorm:"column:onyomi;type:text[];not null"`
	KeyReadingType common.ReadingType `gorm:"column:key_reading_type;type:reading_type;not null"`
	KeyReading     string             `gorm:"column:key_reading;not null"`
	ReadingTip     *string            `gorm:"column:reading_tip"`
	Mnemonic       *string            `gorm:"column:mnemonic"`
	MnemonicDraft  bool               `gorm:"column:mnemonic_draft;not null"`
	Strokes        int                `gorm:"column:strokes;not null"`
	Grade          *int               `gorm:"column:grade"`
	Freq           *int               `gorm:"column:freq"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Kanji) TableName() string { return "kanji" }

type KanjiKunReading struct {
	ID        uuid.UUID        `gorm:"type:uuid;primaryKey"`
	KanjiID   uuid.UUID        `gorm:"column:kanji_id;type:uuid;not null"`
	Kana      string           `gorm:"column:kana;not null"`
	Okurigana *string          `gorm:"column:okurigana"`
	Affix     *common.KunAffix `gorm:"column:affix;type:kun_affix"`
	Position  int              `gorm:"column:position;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (KanjiKunReading) TableName() string { return "kanji_kun_reading" }

type KanjiVocab struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	KanjiID  uuid.UUID `gorm:"column:kanji_id;type:uuid;not null"`
	Word     string    `gorm:"column:word;not null"`
	Reading  string    `gorm:"column:reading;not null"`
	Meaning  string    `gorm:"column:meaning;not null"`
	IsKey    bool      `gorm:"column:is_key;not null"`
	Position int       `gorm:"column:position;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (KanjiVocab) TableName() string { return "kanji_vocab" }

type KanjiPart struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey"`
	Char string    `gorm:"column:char;not null"`
	Name string    `gorm:"column:name;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (KanjiPart) TableName() string { return "kanji_part" }

type KanjiPartLink struct {
	KanjiID  uuid.UUID `gorm:"column:kanji_id;type:uuid;primaryKey"`
	PartID   uuid.UUID `gorm:"column:part_id;type:uuid;primaryKey"`
	Position int       `gorm:"column:position;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (KanjiPartLink) TableName() string { return "kanji_part_link" }

type KanjiLookAlike struct {
	KanjiID      uuid.UUID `gorm:"column:kanji_id;type:uuid;primaryKey"`
	OtherKanjiID uuid.UUID `gorm:"column:other_kanji_id;type:uuid;primaryKey"`
	Note         *string   `gorm:"column:note"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (KanjiLookAlike) TableName() string { return "kanji_look_alike" }

// The rows below are join results for the API, not tables.

type KanaLookAlikeRow struct {
	KanaID    uuid.UUID `gorm:"column:kana_id"`
	OtherSlug string    `gorm:"column:other_slug"`
	OtherChar string    `gorm:"column:other_char"`
	Note      *string   `gorm:"column:note"`
}

type KanjiPartRow struct {
	KanjiID uuid.UUID `gorm:"column:kanji_id"`
	Char    string    `gorm:"column:char"`
	Name    string    `gorm:"column:name"`
}

type KanjiLookAlikeRow struct {
	KanjiID   uuid.UUID `gorm:"column:kanji_id"`
	OtherChar string    `gorm:"column:other_char"`
	Note      *string   `gorm:"column:note"`
}
