package classes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/middleware"
)

func channelRef(id uuid.UUID) common.ResourceRef {
	return common.ResourceRef{Type: common.RESOURCE_TYPE_CHANNEL, ID: id.String()}
}

func courseRef(id uuid.UUID) common.ResourceRef {
	return common.ResourceRef{Type: common.RESOURCE_TYPE_COURSE, ID: id.String()}
}

// requireAny passes when the user holds any of perms; a failure other than
// "forbidden" (such as a database error) is returned as is.
func requireAny(ctx context.Context, checker PermissionChecker, res common.ResourceRef, perms ...common.Permission) error {
	for _, perm := range perms {
		err := checker.Require(ctx, perm, res)
		if err == nil {
			return nil
		}
		if !errors.Is(err, exceptions.ErrForbidden) {
			return fmt.Errorf("check %s: %w", perm, err)
		}
	}
	return exceptions.ErrForbidden
}

var classesPermissions = []common.Permission{
	common.PERMISSION_CHANNEL_EDIT,
	common.PERMISSION_CHANNEL_MEDIA,
	common.PERMISSION_COURSE_EDIT,
	common.PERMISSION_CLASSES_PUBLISH,
	common.PERMISSION_CLASSES_SYNC,
}

func notFoundAs(err error, notFound *exceptions.AppError, action string) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return notFound
	}
	return fmt.Errorf("%s: %w", action, err)
}

// channelEdit builds the changes for one channel from its locked row.
type channelEdit func(tx *gorm.DB, ch *Channel, cs *changeSet) error

// updateChannel locks the channel, applies edit, and writes the changed
// columns and their change-log rows in one transaction. It returns the row as
// it was before the edit and as it is after.
func updateChannel(ctx context.Context, repo ChannelWriter, id uuid.UUID, edit channelEdit) (before, after Channel, err error) {
	err = repo.Transaction(ctx, func(tx *gorm.DB) error {
		locked, err := repo.LockChannel(ctx, tx, id)
		if err != nil {
			return notFoundAs(err, exceptions.ErrChannelNotFound, "lock channel")
		}
		before = locked
		cs := newChangeSet()
		if err := edit(tx, &locked, cs); err != nil {
			return err
		}
		if cs.Empty() {
			return nil
		}
		if err := repo.UpdateChannel(ctx, tx, id, cs.columns); err != nil {
			return fmt.Errorf("update channel: %w", err)
		}
		rows := cs.Rows(middleware.ContextUserID(ctx), common.RESOURCE_TYPE_CHANNEL, id, time.Now())
		if err := repo.AppendChangeLog(ctx, tx, rows); err != nil {
			return fmt.Errorf("log channel change: %w", err)
		}
		return nil
	})
	if err != nil {
		return Channel{}, Channel{}, fmt.Errorf("update channel: %w", err)
	}
	after, err = repo.GetChannel(ctx, id)
	if err != nil {
		return Channel{}, Channel{}, notFoundAs(err, exceptions.ErrChannelNotFound, "reload channel")
	}
	return before, after, nil
}

// courseEdit builds the changes for one course from its locked row.
type courseEdit func(tx *gorm.DB, c *Course, cs *changeSet) error

type courseUpdater interface {
	Transaction(ctx context.Context, fn func(*gorm.DB) error) error
	LockCourse(ctx context.Context, tx *gorm.DB, id uuid.UUID) (Course, error)
	UpdateCourse(ctx context.Context, tx *gorm.DB, id uuid.UUID, columns map[string]any) error
	AppendChangeLog(ctx context.Context, tx *gorm.DB, rows []ChangeLog) error
}

func updateCourse(ctx context.Context, repo courseUpdater, id uuid.UUID, edit courseEdit) error {
	err := repo.Transaction(ctx, func(tx *gorm.DB) error {
		locked, err := repo.LockCourse(ctx, tx, id)
		if err != nil {
			return notFoundAs(err, exceptions.ErrClassNotFound, "lock course")
		}
		cs := newChangeSet()
		if err := edit(tx, &locked, cs); err != nil {
			return err
		}
		if cs.Empty() {
			return nil
		}
		if err := repo.UpdateCourse(ctx, tx, id, cs.columns); err != nil {
			return fmt.Errorf("update course: %w", err)
		}
		rows := cs.Rows(middleware.ContextUserID(ctx), common.RESOURCE_TYPE_COURSE, id, time.Now())
		if err := repo.AppendChangeLog(ctx, tx, rows); err != nil {
			return fmt.Errorf("log course change: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("update course: %w", err)
	}
	return nil
}
