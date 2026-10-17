package progress

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/learn"
)

type DashboardService struct {
	repo   DashboardRepo
	pools  PoolLoader
	now    func() time.Time
	logger *zap.Logger
}

func NewDashboardService(repo DashboardRepo, pools PoolLoader, log *zap.Logger) *DashboardService {
	return &DashboardService{repo: repo, pools: pools, now: time.Now, logger: log}
}

func (s *DashboardService) Get(ctx context.Context, userID uuid.UUID) (DashboardData, error) {
	clock, err := loadClock(ctx, s.repo, userID, s.now())
	if err != nil {
		return DashboardData{}, fmt.Errorf("dashboard: %w", err)
	}
	x, stats, err := loadIndexAndStats(ctx, s.repo, s.pools, userID, time.Time{})
	if err != nil {
		return DashboardData{}, fmt.Errorf("dashboard: %w", err)
	}
	goalDates, err := s.repo.ListGoalMetDates(ctx, userID)
	if err != nil {
		return DashboardData{}, fmt.Errorf("dashboard: %w", err)
	}
	streak := computeStreaks(goalDates, clock.todayDate())
	calendar, err := s.streakCalendar(ctx, &clock, streak)
	if err != nil {
		return DashboardData{}, fmt.Errorf("dashboard: %w", err)
	}
	dashStats, err := s.stats(ctx, userID, x, streak)
	if err != nil {
		return DashboardData{}, fmt.Errorf("dashboard: %w", err)
	}
	sessions, err := s.repo.ListRecentSessions(ctx, userID, common.DASHBOARD_STUDY_LOG_SIZE)
	if err != nil {
		return DashboardData{}, fmt.Errorf("dashboard: %w", err)
	}
	return DashboardData{
		Stats:          dashStats,
		SkillBalance:   skillBalance(stats),
		Milestones:     milestones(&milestoneInputs{Index: x, Streaks: streak}),
		StudyLog:       studyLog(&clock, sessions),
		StreakCalendar: calendar,
	}, nil
}

// Every answer on an introduced card counts as a card reviewed.
func (s *DashboardService) stats(ctx context.Context, userID uuid.UUID, x *progressIndex, streak streaks) (DashboardStats, error) {
	activeMs, err := s.repo.SumActiveMs(ctx, userID)
	if err != nil {
		return DashboardStats{}, fmt.Errorf("dashboard stats: %w", err)
	}
	reviewed, err := s.repo.CountReviewAnswersSince(ctx, userID, time.Time{})
	if err != nil {
		return DashboardStats{}, fmt.Errorf("dashboard stats: %w", err)
	}
	target, err := s.repo.GetTargetLevel(ctx, userID)
	if err != nil {
		return DashboardStats{}, fmt.Errorf("dashboard stats: %w", err)
	}
	if target == "" {
		target = string(common.DEFAULT_TARGET_LEVEL)
	}
	mastered, total := 0, 0
	for _, pool := range x.pools {
		n, all := x.trackProgress(pool.Track, anyUnit, isMastered)
		mastered, total = mastered+n, total+all
	}
	hours := (time.Duration(activeMs) * time.Millisecond).Hours()
	return DashboardStats{
		StudyHours:    math.Round(hours*common.PERCENT_ROUNDING) / common.PERCENT_ROUNDING,
		CardsReviewed: reviewed,
		CurrentStreak: streak.Current,
		JLPTProgress:  percentOf(mastered, total),
		TargetLevel:   target,
	}, nil
}

// streakCalendar covers the last CALENDAR_DEFAULT_DAYS local days.
func (s *DashboardService) streakCalendar(ctx context.Context, clock *studyClock, streak streaks) (StreakCalendarData, error) {
	to := clock.todayDate()
	from := to.AddDate(0, 0, 1-common.CALENDAR_DEFAULT_DAYS)
	rows, err := s.repo.ListDailyActivity(ctx, clock.Settings.UserID, formatLocalDate(from), formatLocalDate(to))
	if err != nil {
		return StreakCalendarData{}, fmt.Errorf("streak calendar: %w", err)
	}
	days := calendarDays(rows, from, to)
	out := StreakCalendarData{
		Activities: make([]DayActivity, 0, len(days)), CurrentStreak: streak.Current, LongestStreak: streak.Longest,
	}
	for i := range days {
		out.Activities = append(out.Activities, DayActivity{Date: days[i].Date, Minutes: days[i].Minutes})
		if days[i].Answers > 0 {
			out.TotalActiveDays++
		}
		if i >= len(days)-common.DASHBOARD_WEEK_DAYS {
			out.ThisWeekMinutes += days[i].Minutes
		}
	}
	return out, nil
}

// skillBalance is accuracy per skill, 0–100, over all answers.
func skillBalance(rows []learn.CardAnswerStats) SkillBalance {
	bySkill := map[common.Skill]*answerTotals{}
	for i := range rows {
		skill, ok := common.QUESTION_TYPE_SKILL[rows[i].QuestionType]
		if !ok {
			continue
		}
		if bySkill[skill] == nil {
			bySkill[skill] = &answerTotals{}
		}
		bySkill[skill].add(&rows[i], nil)
	}
	pct := func(skills ...common.Skill) float64 {
		var t answerTotals
		for _, sk := range skills {
			if bySkill[sk] != nil {
				t.merge(bySkill[sk])
			}
		}
		return percentOf(t.Correct, t.Answers)
	}
	return SkillBalance{
		Kanji:        pct(common.SkillKanjiMeaning, common.SkillKanjiReading),
		Reading:      pct(common.SkillKanaReading),
		Listening:    pct(common.SkillListening),
		KanaReading:  pct(common.SkillKanaReading),
		KanjiMeaning: pct(common.SkillKanjiMeaning),
		KanjiReading: pct(common.SkillKanjiReading),
	}
}

func milestones(in *milestoneInputs) []Milestone {
	out := make([]Milestone, 0, len(milestoneDefs))
	for _, d := range milestoneDefs {
		value, total := d.Progress(in)
		value = min(value, total)
		out = append(out, Milestone{
			ID: d.ID, Icon: d.Icon, Title: d.Title, ColorClass: d.ColorClass,
			Subtitle:      fmt.Sprintf("%d / %d %s", value, total, d.Unit),
			ProgressValue: value, ProgressMax: total,
		})
	}
	return out
}

func studyLog(clock *studyClock, sessions []learn.StudySession) []StudyLogEntry {
	out := make([]StudyLogEntry, 0, len(sessions))
	for i := range sessions {
		ss := &sessions[i]
		style := studyLogStyles[ss.Kind]
		tracks := allTracksTitle
		if ss.Track != nil {
			tracks = trackTitles[*ss.Track]
		}
		status := studyStatusUnfinish
		if ss.CompletedAt != nil {
			status = studyStatusDone
		}
		correct := percentOf(ss.CorrectCount, ss.AnswerCount)
		out = append(out, StudyLogEntry{
			ID: ss.ID.String(), Icon: style.Icon, IconColorClass: style.IconColor,
			Title:    fmt.Sprintf("%s · %s", tracks, style.Title),
			Subtitle: fmt.Sprintf("%d answers · %.0f%% correct", ss.AnswerCount, correct),
			Date:     clock.dayOf(ss.StartedAt).Start.Format(common.STUDY_LOG_DATE_LAYOUT),
			Duration: fmt.Sprintf("%d min", minutesOf(ss.ActiveMs)),
			Status:   status,
		})
	}
	return out
}
