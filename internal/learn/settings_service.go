package learn

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

type LearningSettingsService struct {
	repo   SettingsRepo
	now    func() time.Time
	logger *zap.Logger
}

func NewLearningSettingsService(repo SettingsRepo, log *zap.Logger) *LearningSettingsService {
	return &LearningSettingsService{repo: repo, now: time.Now, logger: log}
}

func (s *LearningSettingsService) Get(ctx context.Context, userID uuid.UUID) (LearningSettingsResponse, error) {
	settings, err := s.repo.EnsureSettings(ctx, userID)
	if err != nil {
		return LearningSettingsResponse{}, fmt.Errorf("get learning settings: %w", err)
	}
	return toSettingsResponse(&settings), nil
}

// Update changes only the fields present in the request. The ranges repeat
// the table's CHECK constraints so a bad value is a 422, not a 500.
func (s *LearningSettingsService) Update(ctx context.Context, userID uuid.UUID, req *UpdateLearningSettingsRequest) (LearningSettingsResponse, error) {
	settings, err := s.repo.EnsureSettings(ctx, userID)
	if err != nil {
		return LearningSettingsResponse{}, fmt.Errorf("update learning settings: %w", err)
	}
	if err := applySettings(&settings, req); err != nil {
		return LearningSettingsResponse{}, err
	}
	settings.UpdatedAt = s.now().UTC()
	if err := s.repo.UpdateSettings(ctx, &settings); err != nil {
		return LearningSettingsResponse{}, fmt.Errorf("update learning settings: %w", err)
	}
	return toSettingsResponse(&settings), nil
}

type intSetting struct {
	in     *int
	out    *int
	limits common.IntRange
}

func applySettings(st *LearningSettings, req *UpdateLearningSettingsRequest) error {
	if err := applyIntSettings(st, req); err != nil {
		return err
	}
	if req.DesiredRetention != nil {
		r := roundTo(*req.DesiredRetention, common.DESIRED_RETENTION_DECIMALS)
		if r < common.DESIRED_RETENTION_RANGE.Min || r > common.DESIRED_RETENTION_RANGE.Max {
			return exceptions.ErrLearningSettingsInvalid
		}
		st.DesiredRetention = r
	}
	if req.AnswerMode != nil {
		mode, err := parseAnswerMode(*req.AnswerMode)
		if err != nil {
			return err
		}
		st.AnswerMode = mode
	}
	if req.ShowRetention != nil {
		st.ShowRetention = *req.ShowRetention
	}
	return nil
}

func applyIntSettings(st *LearningSettings, req *UpdateLearningSettingsRequest) error {
	ints := []intSetting{
		{req.NewKanaPerDay, &st.NewKanaPerDay, common.NEW_KANA_PER_DAY_RANGE},
		{req.NewKanjiPerDay, &st.NewKanjiPerDay, common.NEW_KANJI_PER_DAY_RANGE},
		{req.MaxReviewsPerDay, &st.MaxReviewsPerDay, common.MAX_REVIEWS_PER_DAY_RANGE},
		{req.SessionSize, &st.SessionSize, common.SESSION_SIZE_RANGE},
		{req.LessonSize, &st.LessonSize, common.LESSON_SIZE_RANGE},
		{req.DailyGoalAnswers, &st.DailyGoalAnswers, common.DAILY_GOAL_ANSWERS_RANGE},
		{req.DayStartsAtHour, &st.DayStartsAtHour, common.DAY_STARTS_AT_HOUR_RANGE},
	}
	for _, f := range ints {
		if f.in == nil {
			continue
		}
		if *f.in < f.limits.Min || *f.in > f.limits.Max {
			return exceptions.ErrLearningSettingsInvalid
		}
		*f.out = *f.in
	}
	return nil
}

func parseAnswerMode(raw string) (common.AnswerMode, error) {
	switch mode := common.AnswerMode(raw); mode {
	case common.AnswerModeChoice, common.AnswerModeTyped, common.AnswerModeMixed:
		return mode, nil
	default:
		return "", exceptions.ErrLearningSettingsInvalid
	}
}

func roundTo(v float64, decimals int) float64 {
	scale := math.Pow10(decimals)
	return math.Round(v*scale) / scale
}

func toSettingsResponse(s *LearningSettings) LearningSettingsResponse {
	return LearningSettingsResponse{
		DesiredRetention: s.DesiredRetention,
		NewKanaPerDay:    s.NewKanaPerDay,
		NewKanjiPerDay:   s.NewKanjiPerDay,
		MaxReviewsPerDay: s.MaxReviewsPerDay,
		SessionSize:      s.SessionSize,
		LessonSize:       s.LessonSize,
		DailyGoalAnswers: s.DailyGoalAnswers,
		DayStartsAtHour:  s.DayStartsAtHour,
		AnswerMode:       string(s.AnswerMode),
		ShowRetention:    s.ShowRetention,
	}
}
