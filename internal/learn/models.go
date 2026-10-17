package learn

import (
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/srs"
)

type LearningSettings struct {
	UserID           uuid.UUID             `gorm:"column:user_id;type:uuid;primaryKey"`
	DesiredRetention float64               `gorm:"column:desired_retention;not null"`
	NewKanaPerDay    int                   `gorm:"column:new_kana_per_day;not null"`
	NewKanjiPerDay   int                   `gorm:"column:new_kanji_per_day;not null"`
	MaxReviewsPerDay int                   `gorm:"column:max_reviews_per_day;not null"`
	SessionSize      int                   `gorm:"column:session_size;not null"`
	LessonSize       int                   `gorm:"column:lesson_size;not null"`
	DailyGoalAnswers int                   `gorm:"column:daily_goal_answers;not null"`
	DayStartsAtHour  int                   `gorm:"column:day_starts_at_hour;not null"`
	AnswerMode       common.AnswerMode     `gorm:"column:answer_mode;type:answer_mode;not null"`
	ShowRetention    bool                  `gorm:"column:show_retention;not null"`
	FSRSWeights      database.Float64Array `gorm:"column:fsrs_weights;type:double precision[]"`

	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (LearningSettings) TableName() string { return "learning_settings" }

type SRSCard struct {
	ID            uuid.UUID          `gorm:"type:uuid;primaryKey"`
	UserID        uuid.UUID          `gorm:"column:user_id;type:uuid;not null"`
	ItemType      common.SRSItemType `gorm:"column:item_type;type:srs_item_type;not null"`
	ItemID        uuid.UUID          `gorm:"column:item_id;type:uuid;not null"`
	Facet         common.CardFacet   `gorm:"column:facet;type:card_facet;not null"`
	State         srs.State          `gorm:"column:state;not null"`
	Due           time.Time          `gorm:"column:due;not null"`
	Stability     float64            `gorm:"column:stability;not null"`
	Difficulty    float64            `gorm:"column:difficulty;not null"`
	Reps          int                `gorm:"column:reps;not null"`
	Lapses        int                `gorm:"column:lapses;not null"`
	LearningStep  int                `gorm:"column:learning_step;not null"`
	ScheduledDays int                `gorm:"column:scheduled_days;not null"`
	LastReviewAt  *time.Time         `gorm:"column:last_review_at"`
	IsLeech       bool               `gorm:"column:is_leech;not null"`
	IntroducedAt  time.Time          `gorm:"column:introduced_at;not null"`

	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (SRSCard) TableName() string { return "srs_card" }

type StudySession struct {
	ID           uuid.UUID             `gorm:"type:uuid;primaryKey"`
	UserID       uuid.UUID             `gorm:"column:user_id;type:uuid;not null"`
	Kind         common.SessionKind    `gorm:"column:kind;type:session_kind;not null"`
	Track        *common.LearningTrack `gorm:"column:track;type:learning_track"`
	Scope        *string               `gorm:"column:scope"`
	StartedAt    time.Time             `gorm:"column:started_at;not null"`
	CompletedAt  *time.Time            `gorm:"column:completed_at"`
	AnswerCount  int                   `gorm:"column:answer_count;not null"`
	CorrectCount int                   `gorm:"column:correct_count;not null"`
	ActiveMs     int64                 `gorm:"column:active_ms;not null"`
}

func (StudySession) TableName() string { return "study_session" }

type ReviewLog struct {
	ID            uuid.UUID           `gorm:"type:uuid;primaryKey"`
	UserID        uuid.UUID           `gorm:"column:user_id;type:uuid;not null"`
	CardID        uuid.UUID           `gorm:"column:card_id;type:uuid;not null"`
	SessionID     uuid.UUID           `gorm:"column:session_id;type:uuid;not null"`
	ClientEventID uuid.UUID           `gorm:"column:client_event_id;type:uuid;not null"`
	QuestionType  common.QuestionType `gorm:"column:question_type;type:question_type;not null"`
	AnsweredAt    time.Time           `gorm:"column:answered_at;not null"`

	ShownAt        time.Time           `gorm:"column:shown_at;not null"`
	DurationMs     int                 `gorm:"column:duration_ms;not null"`
	IdleMs         int                 `gorm:"column:idle_ms;not null"`
	FirstInputMs   *int                `gorm:"column:first_input_ms"`
	Attempts       int                 `gorm:"column:attempts;not null"`
	AnswerText     *string             `gorm:"column:answer_text"`
	ChosenOption   *string             `gorm:"column:chosen_option"`
	Options        *database.TextArray `gorm:"column:options;type:text[]"`
	HintsUsed      int                 `gorm:"column:hints_used;not null"`
	AnswerRevealed bool                `gorm:"column:answer_revealed;not null"`
	AudioReplays   int                 `gorm:"column:audio_replays;not null"`
	AudioMs        *int                `gorm:"column:audio_ms"`
	StrokeErrors   *int                `gorm:"column:stroke_errors"`
	StrokeCount    *int                `gorm:"column:stroke_count"`
	GuideLevel     *common.GuideLevel  `gorm:"column:guide_level;type:guide_level"`

	IsCorrect            bool               `gorm:"column:is_correct;not null"`
	TypoDistance         *int               `gorm:"column:typo_distance"`
	ActiveMs             int                `gorm:"column:active_ms;not null"`
	Rating               *srs.Rating        `gorm:"column:rating"`
	GradeReason          common.GradeReason `gorm:"column:grade_reason;type:grade_reason;not null"`
	CountsForSRS         bool               `gorm:"column:counts_for_srs;not null"`
	StateBefore          srs.State          `gorm:"column:state_before;not null"`
	RetrievabilityBefore *float64           `gorm:"column:retrievability_before"`
	StabilityBefore      *float64           `gorm:"column:stability_before"`
	StabilityAfter       *float64           `gorm:"column:stability_after"`
	DifficultyBefore     *float64           `gorm:"column:difficulty_before"`
	DifficultyAfter      *float64           `gorm:"column:difficulty_after"`
	ElapsedDays          int                `gorm:"column:elapsed_days;not null"`
	ScheduledDays        int                `gorm:"column:scheduled_days;not null"`
	DueAfter             *time.Time         `gorm:"column:due_after"`
}

func (ReviewLog) TableName() string { return "review_log" }

type UserConfusion struct {
	UserID       uuid.UUID          `gorm:"column:user_id;type:uuid;primaryKey"`
	ItemType     common.SRSItemType `gorm:"column:item_type;type:srs_item_type;primaryKey"`
	TargetItemID uuid.UUID          `gorm:"column:target_item_id;type:uuid;primaryKey"`
	ChosenItemID uuid.UUID          `gorm:"column:chosen_item_id;type:uuid;primaryKey"`
	Times        int                `gorm:"column:times;not null"`
	LastAt       time.Time          `gorm:"column:last_at;not null"`
}

func (UserConfusion) TableName() string { return "user_confusion" }

// LocalDate is the user's study day (see dayWindow), read back as midnight UTC.
type DailyActivity struct {
	UserID        uuid.UUID `gorm:"column:user_id;type:uuid;primaryKey"`
	LocalDate     time.Time `gorm:"column:local_date;type:date;primaryKey"`
	ReviewsDone   int       `gorm:"column:reviews_done;not null"`
	NewLearned    int       `gorm:"column:new_learned;not null"`
	CorrectCount  int       `gorm:"column:correct_count;not null"`
	AgainCount    int       `gorm:"column:again_count;not null"`
	ActiveMs      int64     `gorm:"column:active_ms;not null"`
	DueAtDayStart int       `gorm:"column:due_at_day_start;not null"`
	GoalMet       bool      `gorm:"column:goal_met;not null"`
}

func (DailyActivity) TableName() string { return "daily_activity" }

// CardKey is one existing card of an item, enough to tell which facets of
// the item have been introduced.
type CardKey struct {
	ItemID uuid.UUID        `gorm:"column:item_id"`
	Facet  common.CardFacet `gorm:"column:facet"`
}

// ItemChar is a join result: the character shown for a card's item.
type ItemChar struct {
	ID   uuid.UUID `gorm:"column:id"`
	Char string    `gorm:"column:char"`
}

// BaselineSample is one past correct answer used to build a Baseline.
type BaselineSample struct {
	ActiveMs     int     `gorm:"column:active_ms"`
	FirstInputMs *int    `gorm:"column:first_input_ms"`
	AnswerText   *string `gorm:"column:answer_text"`
}
