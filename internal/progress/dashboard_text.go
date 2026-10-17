package progress

import (
	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
)

// Display text and Material icon names for the dashboard, which the
// frontend DashboardData type expects from the server.

type milestoneDef struct {
	ID         string
	Icon       string
	Title      string
	Unit       string
	ColorClass string
	Progress   func(in *milestoneInputs) (value, total int)
}

type milestoneInputs struct {
	Index   *progressIndex
	Streaks streaks
}

const (
	colorPrimary   = "text-primary"
	colorSecondary = "text-secondary"
	colorTertiary  = "text-tertiary"
	iconTranslate  = "translate"
	iconFire       = "local_fire_department"
	unitLearned    = "learned"
	unitMastered   = "mastered"
	unitDays       = "days"
)

var milestoneDefs = []milestoneDef{
	{ID: "hiragana-basic-learned", Icon: iconTranslate, Title: "Basic hiragana", Unit: unitLearned, ColorClass: colorPrimary,
		Progress: func(in *milestoneInputs) (int, int) {
			return in.Index.trackProgress(common.LearningTrackHiragana, isBasicKana, isLearned)
		}},
	{ID: "hiragana-mastered", Icon: iconTranslate, Title: "Hiragana mastered", Unit: unitMastered, ColorClass: colorPrimary,
		Progress: func(in *milestoneInputs) (int, int) {
			return in.Index.trackProgress(common.LearningTrackHiragana, anyUnit, isMastered)
		}},
	{ID: "katakana-learned", Icon: iconTranslate, Title: "All katakana", Unit: unitLearned, ColorClass: colorSecondary,
		Progress: func(in *milestoneInputs) (int, int) {
			return in.Index.trackProgress(common.LearningTrackKatakana, anyUnit, isLearned)
		}},
	{ID: "kanji-first-20", Icon: "draw", Title: "First 20 kanji", Unit: unitLearned, ColorClass: colorSecondary,
		Progress: func(in *milestoneInputs) (int, int) {
			learned, _ := in.Index.trackProgress(common.LearningTrackKanjiN5, anyUnit, isLearned)
			return learned, common.MILESTONE_FIRST_KANJI
		}},
	{ID: "kanji-n5-complete", Icon: "workspace_premium", Title: "N5 kanji complete", Unit: unitMastered, ColorClass: colorSecondary,
		Progress: func(in *milestoneInputs) (int, int) {
			return in.Index.trackProgress(common.LearningTrackKanjiN5, anyUnit, isMastered)
		}},
	{ID: "streak-7", Icon: iconFire, Title: "7-day streak", Unit: unitDays, ColorClass: colorTertiary,
		Progress: func(in *milestoneInputs) (int, int) { return in.Streaks.Longest, common.MILESTONE_SHORT_STREAK }},
	{ID: "streak-30", Icon: iconFire, Title: "30-day streak", Unit: unitDays, ColorClass: colorTertiary,
		Progress: func(in *milestoneInputs) (int, int) { return in.Streaks.Longest, common.MILESTONE_LONG_STREAK }},
}

func isBasicKana(u learn.Unit) bool { //nolint:gocritic // Pool.ItemsWhere passes units by value
	return u.Group == string(common.KanaGroupBasic)
}

type studyLogStyle struct {
	Title     string
	Icon      string
	IconColor string
}

var studyLogStyles = map[common.SessionKind]studyLogStyle{
	common.SessionKindLesson:    {Title: "Lesson", Icon: "school", IconColor: "text-primary bg-primary/5"},
	common.SessionKindReview:    {Title: "Review", Icon: "replay", IconColor: "text-secondary bg-secondary/5"},
	common.SessionKindPractice:  {Title: "Practice", Icon: "fitness_center", IconColor: "text-tertiary bg-tertiary/5"},
	common.SessionKindUnitQuiz:  {Title: "Unit quiz", Icon: "quiz", IconColor: "text-primary bg-primary/5"},
	common.SessionKindSkipTest:  {Title: "Skip test", Icon: "quiz", IconColor: "text-primary bg-primary/5"},
	common.SessionKindFinalTest: {Title: "Final test", Icon: "military_tech", IconColor: "text-primary bg-primary/5"},
}

var trackTitles = map[common.LearningTrack]string{
	common.LearningTrackHiragana: "Hiragana",
	common.LearningTrackKatakana: "Katakana",
	common.LearningTrackKanjiN5:  "N5 kanji",
}

const (
	allTracksTitle      = "All tracks"
	studyStatusDone     = "completed"
	studyStatusUnfinish = "in-progress"
)
