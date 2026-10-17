package progress

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
	"github.com/fluentfox/api/pkg/srs"
)

func dashboardFixture() *fakeRepo {
	repo := newFakeRepo()
	repo.card(common.SRSItemKana, hiraA, common.CardFacetRecognition, srs.StateReview, 30)
	repo.card(common.SRSItemKana, hiraI, common.CardFacetRecognition, srs.StateReview, 22)
	repo.card(common.SRSItemKana, kataA, common.CardFacetRecognition, srs.StateLearning, 0.1)
	repo.card(common.SRSItemKanji, kanji1, common.CardFacetMeaning, srs.StateReview, 30)
	repo.card(common.SRSItemKanji, kanji1, common.CardFacetReading, srs.StateReview, 30)
	card := uuid.New()
	repo.stats = []learn.CardAnswerStats{
		stat(card, common.QuestionKanaMCQRomaji, 10, 8, 0, 0, 0, 0),
		stat(card, common.QuestionKanaListen, 4, 2, 0, 0, 0, 0),
		stat(card, common.QuestionKanjiMeaningMCQ, 10, 9, 0, 0, 0, 0),
		stat(card, common.QuestionKanjiReadingTyped, 10, 6, 0, 0, 0, 0),
		stat(card, common.QuestionKanaTrace, 5, 0, 0, 0, 0, 0),
	}
	for d := 1; d <= 7; d++ {
		repo.goalDates = append(repo.goalDates, date("2026-03-10").AddDate(0, 0, d-8))
	}
	repo.activity = []learn.DailyActivity{
		{LocalDate: date("2025-01-01"), ReviewsDone: 50, ActiveMs: 3_600_000},
		{LocalDate: date("2026-03-07"), ReviewsDone: 40, ActiveMs: 1_800_000},
		{LocalDate: date("2026-03-10"), ReviewsDone: 10, ActiveMs: 600_000},
	}
	repo.activeMs = 5_400_000
	repo.reviewCount = 120
	review, track := common.SessionKindReview, common.LearningTrackHiragana
	done := testNow.Add(-time.Hour)
	repo.sessions = []learn.StudySession{
		{ID: uuid.New(), Kind: review, Track: &track, StartedAt: testNow.Add(-2 * time.Hour), CompletedAt: &done,
			AnswerCount: 12, CorrectCount: 9, ActiveMs: 300_000},
		{ID: uuid.New(), Kind: common.SessionKindPractice, StartedAt: testNow.Add(-26 * time.Hour), AnswerCount: 3, CorrectCount: 3, ActiveMs: 50_000},
	}
	return repo
}

func getDashboard(t *testing.T, repo *fakeRepo) DashboardData {
	t.Helper()
	svc := NewDashboardService(repo, &fakePools{pools: testPools()}, zap.NewNop())
	svc.now = func() time.Time { return testNow }
	got, err := svc.Get(context.Background(), testUserID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	return got
}

func TestDashboardStats(t *testing.T) {
	tests := []struct {
		name   string
		target string
		want   DashboardStats
	}{
		{"target level defaults to N5", "", DashboardStats{StudyHours: 1.5, CardsReviewed: 120, CurrentStreak: 7, JLPTProgress: 50, TargetLevel: "N5"}},
		{"target level from the profile", "N4", DashboardStats{StudyHours: 1.5, CardsReviewed: 120, CurrentStreak: 7, JLPTProgress: 50, TargetLevel: "N4"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := dashboardFixture()
			repo.targetLevel = tt.target
			if got := getDashboard(t, repo).Stats; got != tt.want {
				t.Errorf("stats = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDashboardSkillBalance(t *testing.T) {
	want := SkillBalance{Kanji: 75, Listening: 50, Reading: 80, KanaReading: 80, KanjiMeaning: 90, KanjiReading: 60}
	if got := getDashboard(t, dashboardFixture()).SkillBalance; got != want {
		t.Errorf("skill balance = %+v, want %+v", got, want)
	}
}

func TestDashboardMilestones(t *testing.T) {
	got := getDashboard(t, dashboardFixture()).Milestones
	want := []struct {
		id         string
		value, max int
	}{
		{"hiragana-basic-learned", 2, 2},
		{"hiragana-mastered", 2, 3},
		{"katakana-learned", 1, 1},
		{"kanji-first-20", 1, 20},
		{"kanji-n5-complete", 1, 2},
		{"streak-7", 7, 7},
		{"streak-30", 7, 30},
	}
	if len(got) != len(want) {
		t.Fatalf("milestones = %d, want %d", len(got), len(want))
	}
	for i, w := range want {
		t.Run(w.id, func(t *testing.T) {
			if got[i].ID != w.id || got[i].ProgressValue != w.value || got[i].ProgressMax != w.max {
				t.Errorf("milestone = %+v, want %s %d/%d", got[i], w.id, w.value, w.max)
			}
		})
	}
}

func TestDashboardCalendarAndStudyLog(t *testing.T) {
	got := getDashboard(t, dashboardFixture())
	cal := got.StreakCalendar
	if len(cal.Activities) != common.CALENDAR_DEFAULT_DAYS || cal.Activities[len(cal.Activities)-1].Date != "2026-03-10" {
		t.Fatalf("activities = %d ending %s", len(cal.Activities), cal.Activities[len(cal.Activities)-1].Date)
	}
	if cal.ThisWeekMinutes != 40 || cal.TotalActiveDays != 2 || cal.CurrentStreak != 7 || cal.LongestStreak != 7 {
		t.Errorf("calendar = week %d, active %d, streak %d/%d", cal.ThisWeekMinutes, cal.TotalActiveDays, cal.CurrentStreak, cal.LongestStreak)
	}
	tests := []struct {
		name string
		got  StudyLogEntry
		want StudyLogEntry
	}{
		{"completed review", got.StudyLog[0], StudyLogEntry{
			Title: "Hiragana · Review", Subtitle: "12 answers · 75% correct", Date: "Mar 10, 2026", Duration: "5 min", Status: "completed",
		}},
		{"unfinished practice over all tracks", got.StudyLog[1], StudyLogEntry{
			Title: "All tracks · Practice", Subtitle: "3 answers · 100% correct", Date: "Mar 9, 2026", Duration: "1 min", Status: "in-progress",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, w := tt.got, tt.want
			if g.Title != w.Title || g.Subtitle != w.Subtitle || g.Date != w.Date || g.Duration != w.Duration || g.Status != w.Status || g.XP != nil {
				t.Errorf("entry = %+v, want %+v", g, w)
			}
		})
	}
}
