package learn

import (
	"time"

	"github.com/google/uuid"
)

// Field names are snake_case: they are the frontend's src/types/learn.ts.

type LearningSettingsResponse struct {
	DesiredRetention float64 `json:"desired_retention"   doc:"Target recall chance, 0.70 to 0.97"`
	NewKanaPerDay    int     `json:"new_kana_per_day"    doc:"New kana per day, 0 to 100"`
	NewKanjiPerDay   int     `json:"new_kanji_per_day"   doc:"New kanji per day, 0 to 50"`
	MaxReviewsPerDay int     `json:"max_reviews_per_day" doc:"Review answers per day, 10 to 1000"`
	SessionSize      int     `json:"session_size"        doc:"Cards per review session, 5 to 100"`
	LessonSize       int     `json:"lesson_size"         doc:"New items per lesson, 1 to 20"`
	DailyGoalAnswers int     `json:"daily_goal_answers"  doc:"Answers that meet the daily goal, 5 to 200"`
	DayStartsAtHour  int     `json:"day_starts_at_hour"  doc:"Local hour the study day starts, 0 to 23"`
	AnswerMode       string  `json:"answer_mode"         doc:"choice, typed or mixed"`
	ShowRetention    bool    `json:"show_retention"      doc:"Show recall vs target on the dashboard"`
}

// Only the fields present are changed.
type UpdateLearningSettingsRequest struct {
	DesiredRetention *float64 `json:"desired_retention,omitempty"   minimum:"0.7" maximum:"0.97"`
	NewKanaPerDay    *int     `json:"new_kana_per_day,omitempty"    minimum:"0"   maximum:"100"`
	NewKanjiPerDay   *int     `json:"new_kanji_per_day,omitempty"   minimum:"0"   maximum:"50"`
	MaxReviewsPerDay *int     `json:"max_reviews_per_day,omitempty" minimum:"10"  maximum:"1000"`
	SessionSize      *int     `json:"session_size,omitempty"        minimum:"5"   maximum:"100"`
	LessonSize       *int     `json:"lesson_size,omitempty"         minimum:"1"   maximum:"20"`
	DailyGoalAnswers *int     `json:"daily_goal_answers,omitempty"  minimum:"5"   maximum:"200"`
	DayStartsAtHour  *int     `json:"day_starts_at_hour,omitempty"  minimum:"0"   maximum:"23"`
	AnswerMode       *string  `json:"answer_mode,omitempty"         enum:"choice,typed,mixed"`
	ShowRetention    *bool    `json:"show_retention,omitempty"`
}

type SummaryInput struct {
	Track string `query:"track" required:"false" enum:"hiragana,katakana,kanji_n5" doc:"Limit to one track; all tracks when omitted"`
}

type LearnSummary struct {
	DueNow       int            `json:"due_now"        doc:"Cards due now"`
	DueToday     int            `json:"due_today"      doc:"Cards due before the study day ends, overdue included"`
	NewLeftToday int            `json:"new_left_today" doc:"New items that may still be started today"`
	Learned      int            `json:"learned"        doc:"Items with at least one card past the new state"`
	NextUnit     *LearnNextUnit `json:"next_unit"      doc:"First unit with items not yet introduced, null when all are"`
	Goal         LearnGoal      `json:"goal"`
}

type LearnNextUnit struct {
	Type  string `json:"type"  doc:"kana or kanji"`
	ID    string `json:"id"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

type LearnGoal struct {
	AnswersToday int  `json:"answers_today"`
	GoalAnswers  int  `json:"goal_answers"`
	Met          bool `json:"met"`
}

type StartLessonRequest struct {
	Track string `json:"track" enum:"hiragana,katakana,kanji_n5" doc:"Track to learn new items from"`
}

type StartReviewRequest struct {
	Track string `json:"track,omitempty" enum:"hiragana,katakana,kanji_n5" doc:"Limit to one track; all tracks when omitted"`
}

type StudySessionDTO struct {
	ID          uuid.UUID  `json:"id"`
	Kind        string     `json:"kind"`
	Track       *string    `json:"track"`
	Scope       *string    `json:"scope"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// Question carries its answer so the client can show feedback at once; the
// server checks the answer again when it is submitted.
type Question struct {
	CardID          uuid.UUID `json:"card_id"`
	ItemType        string    `json:"item_type"`
	ItemID          uuid.UUID `json:"item_id"`
	Facet           string    `json:"facet"`
	QuestionType    string    `json:"question_type"`
	Prompt          string    `json:"prompt"                     doc:"Character, romaji, meaning or reading to show (or speak)"`
	Options         []string  `json:"options,omitempty"          doc:"Choices, absent for typed questions"`
	CorrectAnswer   string    `json:"correct_answer"`
	AcceptedAnswers []string  `json:"accepted_answers,omitempty" doc:"Other answers also accepted (alt romaji, synonyms, other readings)"`
	CardState       int       `json:"card_state"                 doc:"0 new, 1 learning, 2 review, 3 relearning"`
	Char            string    `json:"char"                       doc:"The kana or kanji the card is about"`
	IsLeech         bool      `json:"is_leech"`
}

type LessonItem struct {
	ItemType   string    `json:"item_type"`
	ItemID     uuid.UUID `json:"item_id"`
	Char       string    `json:"char"`
	Romaji     string    `json:"romaji,omitempty"      doc:"Kana only"`
	Meanings   []string  `json:"meanings,omitempty"    doc:"Kanji only"`
	KeyReading string    `json:"key_reading,omitempty" doc:"Kanji only"`
}

type SessionStart struct {
	Session   StudySessionDTO `json:"session"`
	Questions []Question      `json:"questions"`
	Items     []LessonItem    `json:"items,omitempty" doc:"Lessons only: the new items in path order"`
}

type SessionIDInput struct {
	ID uuid.UUID `path:"id" format:"uuid" doc:"Study session id"`
}

type SubmitAnswersInput struct {
	ID   uuid.UUID `path:"id" format:"uuid" doc:"Study session id"`
	Body SubmitAnswersRequest
}

type SubmitAnswersRequest struct {
	Answers []AnswerRequestItem `json:"answers" minItems:"1" maxItems:"50"`
}

type AnswerRequestItem struct {
	ClientEventID  uuid.UUID `json:"client_event_id"          doc:"Client-made id; a resend with the same id returns the stored result"`
	CardID         uuid.UUID `json:"card_id"`
	QuestionType   string    `json:"question_type"            doc:"One of the question_type values; checked against the card facet"`
	ShownAt        time.Time `json:"shown_at"`
	DurationMs     int       `json:"duration_ms"              minimum:"0" maximum:"86400000"`
	IdleMs         int       `json:"idle_ms"                  minimum:"0" maximum:"86400000"`
	FirstInputMs   *int      `json:"first_input_ms"           minimum:"0" maximum:"86400000" required:"false"`
	AnswerText     *string   `json:"answer_text,omitempty"    maxLength:"64"`
	ChosenOption   *string   `json:"chosen_option,omitempty"  maxLength:"64"`
	Options        []string  `json:"options,omitempty"        maxItems:"12"`
	Attempts       int       `json:"attempts"                 minimum:"1" maximum:"100"`
	HintsUsed      int       `json:"hints_used"               minimum:"0" maximum:"100"`
	AnswerRevealed bool      `json:"answer_revealed"`
	AudioReplays   int       `json:"audio_replays"            minimum:"0" maximum:"100"`
	AudioMs        *int      `json:"audio_ms,omitempty"       minimum:"0" maximum:"86400000"`
	StrokeErrors   *int      `json:"stroke_errors,omitempty"  minimum:"0" maximum:"1000"`
	StrokeCount    *int      `json:"stroke_count,omitempty"   minimum:"0" maximum:"1000"`
	GuideLevel     *string   `json:"guide_level,omitempty"    enum:"full,faded,none"`
}

type AnswerResult struct {
	ClientEventID uuid.UUID  `json:"client_event_id"`
	Correct       bool       `json:"correct"`
	CorrectAnswer string     `json:"correct_answer"`
	Label         string     `json:"label"    doc:"strong, good, shaky or missed"`
	Reason        string     `json:"reason"`
	ActiveMs      int        `json:"active_ms"`
	UsualMs       int        `json:"usual_ms" doc:"The user's usual time for this question type"`
	NextDue       *time.Time `json:"next_due"`
	Requeue       bool       `json:"requeue"  doc:"Card is in a learning step: ask it again at the end of the session"`
}

type SubmitAnswersResponse struct {
	Results []AnswerResult `json:"results"`
}

type GradeCounts struct {
	Strong int `json:"strong"`
	Good   int `json:"good"`
	Shaky  int `json:"shaky"`
	Missed int `json:"missed"`
}

type SessionCardResult struct {
	CardID   uuid.UUID  `json:"card_id"`
	ItemType string     `json:"item_type"`
	ItemID   uuid.UUID  `json:"item_id"`
	Facet    string     `json:"facet"`
	Char     string     `json:"char"`
	Label    string     `json:"label"`
	Reason   string     `json:"reason"`
	ActiveMs int        `json:"active_ms"`
	UsualMs  int        `json:"usual_ms"`
	NextDue  *time.Time `json:"next_due"`
}

type SessionResult struct {
	Session      StudySessionDTO     `json:"session"`
	AnswerCount  int                 `json:"answer_count"`
	CorrectCount int                 `json:"correct_count"`
	ActiveMs     int64               `json:"active_ms"`
	Counts       GradeCounts         `json:"counts" doc:"Cards per label, from each card's first graded answer"`
	Cards        []SessionCardResult `json:"cards"`
}
