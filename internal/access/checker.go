package access

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/middleware"
)

// Checker answers permission questions from the database on every call, so a
// revoked grant or admin flag takes effect without waiting for a new token.
type Checker struct {
	repo   CheckerRepo
	logger *zap.Logger
}

func NewChecker(repo CheckerRepo, log *zap.Logger) *Checker {
	return &Checker{repo: repo, logger: log}
}

func (c *Checker) IsSuperAdmin(ctx context.Context, userID uuid.UUID) (bool, error) {
	admin, err := c.repo.IsSuperAdmin(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("check super admin: %w", err)
	}
	return admin, nil
}

// Has is true when the user is a super-admin, holds perm for every resource,
// holds it for res, or res is a course and the user holds it for the course's channel.
func (c *Checker) Has(ctx context.Context, userID uuid.UUID, perm common.Permission, res common.ResourceRef) (bool, error) {
	if userID == uuid.Nil {
		return false, nil
	}
	admin, err := c.IsSuperAdmin(ctx, userID)
	if err != nil || admin {
		return admin, err
	}
	resources, err := c.coveringResources(ctx, res)
	if err != nil {
		return false, err
	}
	ok, err := c.repo.HasGrant(ctx, userID, perm, resources)
	if err != nil {
		return false, fmt.Errorf("check permission: %w", err)
	}
	return ok, nil
}

// Require reads the user from the request's token claims.
func (c *Checker) Require(ctx context.Context, perm common.Permission, res common.ResourceRef) error {
	ok, err := c.Has(ctx, middleware.ContextUserID(ctx), perm, res)
	if err != nil {
		return err
	}
	if !ok {
		return exceptions.ErrForbidden
	}
	return nil
}

// Every grant that would allow access to res: the all-resources grant, res
// itself and, for a course, its channel.
func (c *Checker) coveringResources(ctx context.Context, res common.ResourceRef) ([]common.ResourceRef, error) {
	resources := []common.ResourceRef{{Type: common.RESOURCE_TYPE_ALL}}
	if res.Type == common.RESOURCE_TYPE_ALL {
		return resources, nil
	}
	resources = append(resources, res)
	if res.Type != common.RESOURCE_TYPE_COURSE {
		return resources, nil
	}
	courseID, err := uuid.Parse(res.ID)
	if err != nil {
		return resources, nil //nolint:nilerr // a malformed id has no channel; the direct grants still apply
	}
	channelID, found, err := c.repo.FindCourseChannelID(ctx, courseID)
	if err != nil {
		return nil, fmt.Errorf("check permission: %w", err)
	}
	if found {
		resources = append(resources, common.ResourceRef{Type: common.RESOURCE_TYPE_CHANNEL, ID: channelID})
	}
	return resources, nil
}
