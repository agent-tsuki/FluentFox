package profile

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

type fakeTutorialRepo struct {
	rows    []users.UserTutorial
	listErr error
	saved   *users.UserTutorial
}

func (f *fakeTutorialRepo) ListUserTutorials(context.Context, uuid.UUID) ([]users.UserTutorial, error) {
	return f.rows, f.listErr
}

func (f *fakeTutorialRepo) UpsertUserTutorial(_ context.Context, t *users.UserTutorial) (users.UserTutorial, error) {
	f.saved = t
	return *t, nil
}

func TestTutorialService_List(t *testing.T) {
	user := uuid.New()

	t.Run("no row reads pending", func(t *testing.T) {
		svc := NewTutorialService(&fakeTutorialRepo{}, nop())
		got, err := svc.List(context.Background(), user)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Tutorials) != 1 || got.Tutorials[0].Key != common.TUTORIAL_WELCOME || got.Tutorials[0].Status != "pending" {
			t.Fatalf("want welcome pending, got %+v", got.Tutorials)
		}
		if got.Tutorials[0].UpdatedAt != nil {
			t.Fatal("an untouched tutorial has no updated_at")
		}
	})

	t.Run("stored row wins", func(t *testing.T) {
		repo := &fakeTutorialRepo{rows: []users.UserTutorial{{UserID: user, TutorialKey: common.TUTORIAL_WELCOME, Status: common.TutorialSkipped, Version: 1}}}
		got, err := NewTutorialService(repo, nop()).List(context.Background(), user)
		if err != nil {
			t.Fatal(err)
		}
		if got.Tutorials[0].Status != "skipped" {
			t.Fatalf("want skipped, got %s", got.Tutorials[0].Status)
		}
	})

	t.Run("repo error passes through", func(t *testing.T) {
		_, err := NewTutorialService(&fakeTutorialRepo{listErr: errDB}, nop()).List(context.Background(), user)
		if !errors.Is(err, errDB) {
			t.Fatalf("want db error, got %v", err)
		}
	})
}

func TestTutorialService_Update(t *testing.T) {
	user := uuid.New()
	fixed := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	step := 2

	t.Run("unknown key is refused", func(t *testing.T) {
		repo := &fakeTutorialRepo{}
		_, err := NewTutorialService(repo, nop()).Update(context.Background(), user, "nope", &UpdateTutorialRequest{Status: "completed", Version: 1})
		if !errors.Is(err, exceptions.ErrTutorialNotFound) {
			t.Fatalf("want ErrTutorialNotFound, got %v", err)
		}
		if repo.saved != nil {
			t.Fatal("nothing should be written for an unknown key")
		}
	})

	t.Run("completed stamps completed_at", func(t *testing.T) {
		repo := &fakeTutorialRepo{}
		svc := NewTutorialService(repo, nop())
		svc.now = func() time.Time { return fixed }
		got, err := svc.Update(context.Background(), user, common.TUTORIAL_WELCOME, &UpdateTutorialRequest{Status: "completed", Version: 1, LastStep: &step})
		if err != nil {
			t.Fatal(err)
		}
		if got.CompletedAt == nil || !got.CompletedAt.Equal(fixed) || *got.LastStep != step {
			t.Fatalf("unexpected response %+v", got)
		}
	})

	t.Run("skipped keeps completed_at empty", func(t *testing.T) {
		repo := &fakeTutorialRepo{}
		_, err := NewTutorialService(repo, nop()).Update(context.Background(), user, common.TUTORIAL_WELCOME, &UpdateTutorialRequest{Status: "skipped", Version: 1, LastStep: &step})
		if err != nil {
			t.Fatal(err)
		}
		if repo.saved.CompletedAt != nil || repo.saved.Status != common.TutorialSkipped {
			t.Fatalf("unexpected write %+v", repo.saved)
		}
	})
}
