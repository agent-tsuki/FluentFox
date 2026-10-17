package progress

import (
	"time"

	"github.com/google/uuid"
)

// Field names are snake_case: they are the frontend's src/types/progress.ts.
// The dashboard alone is camelCase, as the frontend DashboardData type is.

type ItemsPerLevel struct {
	New      int `json:"new"`
	Learning int `json:"learning" doc:"In a learning step, or stability under 1 day"`
	Familiar int `json:"familiar" doc:"Stability 1 to 7 days"`
	Strong   int `json:"strong"   doc:"Stability 7 to 21 days"`
	Mastered int `json:"mastered" doc:"Stability 21 days or more; a kanji needs both cards"`
}

type TrackOverview struct {
	Track         string        `json:"track"`
	TotalItems    int           `json:"total_items"`
	ItemsPerLevel ItemsPerLevel `json:"items_per_level"`
	MasteredPct   float64       `json:"mastered_pct"  doc:"0 to 100"`
	Retention30d  *float64      `json:"retention_30d" doc:"Share of reviews recalled in the last 30 days, 0 to 1; null without reviews"`
	AvgAnswerMs   *int          `json:"avg_answer_ms" doc:"Mean time of correct answers in the last 30 days; null without any"`
	Items         []ItemLevel   `json:"items"         doc:"Every item of the track with its level, in path order"`
}

type ItemLevel struct {
	ItemID uuid.UUID `json:"item_id"`
	Char   string    `json:"char"`
	Level  string    `json:"level"`
}

type TodayStats struct {
	Answers      int   `json:"answers"`
	Correct      int   `json:"correct"`
	ActiveMs     int64 `json:"active_ms"`
	NewLearned   int   `json:"new_learned"`
	NewKanaLeft  int   `json:"new_kana_left"  doc:"New kana that may still be started today (hiragana and katakana share it)"`
	NewKanjiLeft int   `json:"new_kanji_left"`
	ReviewsLeft  int   `json:"reviews_left"   doc:"Review answers left under max_reviews_per_day"`
	GoalAnswers  int   `json:"goal_answers"`
	GoalMet      bool  `json:"goal_met"`
	DueNow       int   `json:"due_now"`
}

type StreakInfo struct {
	Current int `json:"current" doc:"Days in a row with the goal met; an unfinished today does not break it"`
	Longest int `json:"longest"`
}

type RetentionInfo struct {
	Actual *float64 `json:"actual" doc:"Share of reviews recalled in the last 30 days; null without reviews"`
	Target float64  `json:"target" doc:"desired_retention"`
}

type ProgressOverview struct {
	Tracks    []TrackOverview `json:"tracks"`
	Today     TodayStats      `json:"today"`
	Streak    StreakInfo      `json:"streak"`
	Retention *RetentionInfo  `json:"retention,omitempty" doc:"Absent when the user hid retention (show_retention false)"`
}

type ForecastInput struct {
	Days int `query:"days" default:"14" minimum:"1" maximum:"60" doc:"Days to forecast, today included"`
}

type ForecastDay struct {
	Date    string         `json:"date"     doc:"Local study date, YYYY-MM-DD; overdue reviews count on the first day"`
	ByTrack map[string]int `json:"by_track" doc:"Reviews due per track; tracks with none are left out"`
	Total   int            `json:"total"`
}

type ProgressForecast struct {
	Days []ForecastDay `json:"days"`
}

type CalendarInput struct {
	From string `query:"from" required:"false" doc:"First local date, YYYY-MM-DD; 364 days before to when omitted"`
	To   string `query:"to"   required:"false" doc:"Last local date, YYYY-MM-DD; today when omitted"`
}

type CalendarDay struct {
	Date     string   `json:"date"`
	Answers  int      `json:"answers"`
	Minutes  int      `json:"minutes"`
	Accuracy *float64 `json:"accuracy" doc:"0 to 1; null on days without answers"`
	GoalMet  bool     `json:"goal_met"`
}

type ProgressCalendar struct {
	Days          []CalendarDay `json:"days"           doc:"Every date of the range, oldest first"`
	CurrentStreak int           `json:"current_streak"`
	LongestStreak int           `json:"longest_streak" doc:"Over all time, not only the range"`
	ActiveDays    int           `json:"active_days"    doc:"Days of the range with answers"`
}

type TrackInput struct {
	Track string `query:"track" required:"false" enum:"hiragana,katakana,kanji_n5" doc:"Limit to one track; all tracks when omitted"`
}

type WeakItemsInput struct {
	Track string `query:"track" required:"false" enum:"hiragana,katakana,kanji_n5" doc:"Limit to one track; all tracks when omitted"`
	Limit int    `query:"limit" default:"20" minimum:"1" maximum:"100"`
}

type PracticeSuggestion struct {
	Track string `json:"track"`
	Mode  string `json:"mode"           doc:"Practice mode for POST /learn/practice"`
	Unit  string `json:"unit,omitempty" doc:"Unit slug for mode unit"`
	Pair  string `json:"pair,omitempty" doc:"Two characters for mode lookalike, such as ぬ:め"`
}

type WeakItem struct {
	CardID         uuid.UUID          `json:"card_id"`
	ItemType       string             `json:"item_type"`
	ItemID         uuid.UUID          `json:"item_id"`
	Facet          string             `json:"facet"`
	Track          string             `json:"track"`
	Char           string             `json:"char"`
	Level          string             `json:"level"`
	Score          float64            `json:"score"           doc:"Weakness score, 0 to 1 (higher is weaker)"`
	Recall         float64            `json:"recall"          doc:"Recall chance now, 0 to 1"`
	LapseRate      float64            `json:"lapse_rate"`
	TimeRatio      float64            `json:"time_ratio"      doc:"Answer time over the user's usual; 0 without timed answers"`
	ConfusionShare float64            `json:"confusion_share" doc:"Share of the item's answers that picked its top mix-up"`
	Reasons        []string           `json:"reasons"`
	Suggestion     PracticeSuggestion `json:"suggestion"`
}

type WeakItemsResponse struct {
	Items []WeakItem `json:"items"`
}

type WeakGroup struct {
	Track      string             `json:"track"`
	Kind       string             `json:"kind"              doc:"unit, or kana_group for dakuten, combo or foreign"`
	UnitID     *uuid.UUID         `json:"unit_id,omitempty" doc:"Units only"`
	Slug       string             `json:"slug"              doc:"Unit slug or kana group key"`
	Title      string             `json:"title"`
	Reviews    int                `json:"reviews"`
	LapseRate  float64            `json:"lapse_rate"`
	TimeRatio  float64            `json:"time_ratio"`
	Reason     string             `json:"reason"`
	Suggestion PracticeSuggestion `json:"suggestion"`
}

type WeakGroupsResponse struct {
	Groups []WeakGroup `json:"groups"`
}

type ConfusionsInput struct {
	Track string `query:"track" required:"false" enum:"hiragana,katakana,kanji_n5" doc:"Limit to one track; all tracks when omitted"`
	Limit int    `query:"limit" default:"10" minimum:"1" maximum:"50"`
}

type ConfusionSide struct {
	ItemID uuid.UUID `json:"item_id"`
	Char   string    `json:"char"`
}

type ConfusionPair struct {
	ItemType string        `json:"item_type"`
	Track    string        `json:"track"`
	Target   ConfusionSide `json:"target" doc:"The item asked"`
	Chosen   ConfusionSide `json:"chosen" doc:"The item picked instead"`
	Times    int           `json:"times"`
	LastAt   time.Time     `json:"last_at"`
}

type ConfusionsResponse struct {
	Pairs []ConfusionPair `json:"pairs"`
}

type ItemHistoryInput struct {
	Type  string `path:"type"   enum:"kana,kanji"`
	ID    string `path:"id"     maxLength:"64" doc:"Kana slug or character, kanji character, or item id"`
	Track string `query:"track" required:"false" enum:"hiragana,katakana,kanji_n5" doc:"Needed for a kana slug that both scripts have"`
}

type ItemCardProgress struct {
	CardID       uuid.UUID  `json:"card_id"`
	Facet        string     `json:"facet"`
	State        int        `json:"state"  doc:"0 new, 1 learning, 2 review, 3 relearning"`
	Level        string     `json:"level"`
	Recall       *float64   `json:"recall" doc:"Recall chance now, 0 to 1; null for a new card"`
	Stability    float64    `json:"stability"`
	Difficulty   float64    `json:"difficulty"`
	Reps         int        `json:"reps"`
	Lapses       int        `json:"lapses"`
	NextDue      *time.Time `json:"next_due"`
	LastReviewAt *time.Time `json:"last_review_at"`
	IsLeech      bool       `json:"is_leech"`
}

type ItemAnswer struct {
	CardID       uuid.UUID `json:"card_id"`
	QuestionType string    `json:"question_type"`
	AnsweredAt   time.Time `json:"answered_at"`
	Correct      bool      `json:"correct"`
	Label        *string   `json:"label"  doc:"null for analytics-only answers"`
	Reason       string    `json:"reason"`
	ActiveMs     int       `json:"active_ms"`
}

type ItemStats struct {
	Answers     int      `json:"answers"`
	Correct     int      `json:"correct"`
	Accuracy    *float64 `json:"accuracy"`
	Reviews     int      `json:"reviews"`
	Retention   *float64 `json:"retention"     doc:"Share of reviews recalled; null without reviews"`
	AvgAnswerMs *int     `json:"avg_answer_ms" doc:"Mean time of correct answers"`
}

type ItemHistory struct {
	ItemType string             `json:"item_type"`
	ItemID   uuid.UUID          `json:"item_id"`
	Track    string             `json:"track"`
	Char     string             `json:"char"`
	Level    string             `json:"level"`
	Cards    []ItemCardProgress `json:"cards"`
	Stats    ItemStats          `json:"stats"`
	Answers  []ItemAnswer       `json:"answers" doc:"Last 20 answers, newest first"`
}

type DashboardStats struct {
	StudyHours    float64 `json:"studyHours"`
	CardsReviewed int     `json:"cardsReviewed" doc:"Answers on cards already introduced"`
	GlobalRank    *string `json:"globalRank"    doc:"Always null until there is a leaderboard"`
	CurrentStreak int     `json:"currentStreak"`
	JLPTProgress  float64 `json:"jlptProgress"  doc:"Share of N5 items (kana included) mastered, 0 to 100"`
	TargetLevel   string  `json:"targetLevel"`
}

// SkillBalance keeps the frontend keys. Kanji, reading and listening hold
// accuracy (0–100) from the question types; grammar, vocabulary and speaking
// have no questions yet and stay 0.
type SkillBalance struct {
	Kanji        float64 `json:"kanji"        doc:"Kanji meaning and reading answers"`
	Grammar      float64 `json:"grammar"`
	Listening    float64 `json:"listening"`
	Vocabulary   float64 `json:"vocabulary"`
	Reading      float64 `json:"reading"      doc:"Kana reading answers"`
	Speaking     float64 `json:"speaking"`
	KanaReading  float64 `json:"kanaReading"`
	KanjiMeaning float64 `json:"kanjiMeaning"`
	KanjiReading float64 `json:"kanjiReading"`
}

type Milestone struct {
	ID            string `json:"id"`
	Icon          string `json:"icon"`
	Title         string `json:"title"`
	Subtitle      string `json:"subtitle"`
	ProgressValue int    `json:"progressValue"`
	ProgressMax   int    `json:"progressMax"`
	ColorClass    string `json:"colorClass"`
}

type StudyLogEntry struct {
	ID             string `json:"id"`
	Icon           string `json:"icon"`
	IconColorClass string `json:"iconColorClass"`
	Title          string `json:"title"`
	Subtitle       string `json:"subtitle"`
	Date           string `json:"date"     doc:"Local date such as Apr 3, 2026"`
	Duration       string `json:"duration" doc:"Such as 12 min"`
	Status         string `json:"status"   doc:"completed or in-progress"`
	XP             *int   `json:"xp"       doc:"Always null; there is no XP yet"`
}

type DayActivity struct {
	Date    string `json:"date"`
	Minutes int    `json:"minutes"`
}

type StreakCalendarData struct {
	Activities      []DayActivity `json:"activities"      doc:"The last 365 local days, oldest first"`
	CurrentStreak   int           `json:"currentStreak"`
	LongestStreak   int           `json:"longestStreak"`
	TotalActiveDays int           `json:"totalActiveDays"`
	ThisWeekMinutes int           `json:"thisWeekMinutes" doc:"The last 7 local days, today included"`
}

type DashboardData struct {
	Stats          DashboardStats     `json:"stats"`
	SkillBalance   SkillBalance       `json:"skillBalance"`
	Milestones     []Milestone        `json:"milestones"`
	StudyLog       []StudyLogEntry    `json:"studyLog"`
	StreakCalendar StreakCalendarData `json:"streakCalendar"`
}
