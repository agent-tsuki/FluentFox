package learn

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

func TestLearningSettingsServiceGet(t *testing.T) {
	tests := []struct {
		name    string
		repoErr error
		wantErr bool
	}{
		{"returns the stored or default settings", nil, false},
		{"passes a repository error on", errBoom, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.err = tc.repoErr
			got, err := NewLearningSettingsService(repo, zap.NewNop()).Get(context.Background(), testUserID)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, want error %v", err, tc.wantErr)
			}
			if !tc.wantErr && (got.LessonSize != 5 || got.AnswerMode != "mixed" || got.DesiredRetention != 0.9) {
				t.Errorf("settings = %+v, want the defaults", got)
			}
		})
	}
}

func TestLearningSettingsServiceUpdate(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateLearningSettingsRequest
		wantErr error
		check   func(t *testing.T, s *LearningSettings)
	}{
		{
			name: "changes only the fields sent",
			req:  UpdateLearningSettingsRequest{LessonSize: ptr(10), AnswerMode: ptr("typed")},
			check: func(t *testing.T, s *LearningSettings) {
				if s.LessonSize != 10 || s.AnswerMode != common.AnswerModeTyped || s.SessionSize != 20 {
					t.Errorf("saved %+v", s)
				}
			},
		},
		{
			name: "zero new kana per day is allowed",
			req:  UpdateLearningSettingsRequest{NewKanaPerDay: ptr(0)},
			check: func(t *testing.T, s *LearningSettings) {
				if s.NewKanaPerDay != 0 {
					t.Errorf("new kana per day = %d, want 0", s.NewKanaPerDay)
				}
			},
		},
		{
			name: "retention is rounded to two decimals",
			req:  UpdateLearningSettingsRequest{DesiredRetention: ptr(0.8549)},
			check: func(t *testing.T, s *LearningSettings) {
				if s.DesiredRetention != 0.85 {
					t.Errorf("retention = %v, want 0.85", s.DesiredRetention)
				}
			},
		},
		{
			name: "show retention can be turned off",
			req:  UpdateLearningSettingsRequest{ShowRetention: ptr(false)},
			check: func(t *testing.T, s *LearningSettings) {
				if s.ShowRetention {
					t.Error("show retention still on")
				}
			},
		},
		{name: "a lesson size above the range is rejected", req: UpdateLearningSettingsRequest{LessonSize: ptr(21)},
			wantErr: exceptions.ErrLearningSettingsInvalid},
		{name: "a review limit below the range is rejected", req: UpdateLearningSettingsRequest{MaxReviewsPerDay: ptr(9)},
			wantErr: exceptions.ErrLearningSettingsInvalid},
		{name: "a day start hour of 24 is rejected", req: UpdateLearningSettingsRequest{DayStartsAtHour: ptr(24)},
			wantErr: exceptions.ErrLearningSettingsInvalid},
		{name: "a retention above the range is rejected", req: UpdateLearningSettingsRequest{DesiredRetention: ptr(0.98)},
			wantErr: exceptions.ErrLearningSettingsInvalid},
		{name: "an unknown answer mode is rejected", req: UpdateLearningSettingsRequest{AnswerMode: ptr("voice")},
			wantErr: exceptions.ErrLearningSettingsInvalid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			svc := NewLearningSettingsService(repo, zap.NewNop())
			svc.now = func() time.Time { return testNow }
			_, err := svc.Update(context.Background(), testUserID, &tc.req)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr != nil {
				if repo.updatedSettings != nil {
					t.Error("invalid settings were saved")
				}
				return
			}
			if repo.updatedSettings == nil {
				t.Fatal("settings were not saved")
			}
			if !repo.updatedSettings.UpdatedAt.Equal(testNow) {
				t.Errorf("updated_at = %v, want %v", repo.updatedSettings.UpdatedAt, testNow)
			}
			tc.check(t, repo.updatedSettings)
		})
	}
}
