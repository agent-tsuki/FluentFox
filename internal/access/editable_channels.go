package access

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/middleware"
)

// GrantReader is satisfied by *Repository.
type GrantReader interface {
	ListUserGrants(ctx context.Context, userID uuid.UUID) ([]UserPermission, error)
}

// EditableChannels tells the classes admin which channels the signed-in user
// may work on; classes.GrantLister is the consumer interface.
type EditableChannels struct {
	checker *Checker
	grants  GrantReader
}

func NewEditableChannels(checker *Checker, grants GrantReader) *EditableChannels {
	return &EditableChannels{checker: checker, grants: grants}
}

// all is true for super-admins and for any classes grant on every resource;
// ids are the channels the user holds a classes grant on. Course grants do
// not make a channel editable.
func (e *EditableChannels) EditableChannelIDs(ctx context.Context) (all bool, ids []string, err error) {
	userID := middleware.ContextUserID(ctx)
	if userID == uuid.Nil {
		return false, nil, nil
	}
	admin, err := e.checker.IsSuperAdmin(ctx, userID)
	if err != nil || admin {
		return admin, nil, err
	}
	grants, err := e.grants.ListUserGrants(ctx, userID)
	if err != nil {
		return false, nil, fmt.Errorf("list editable channels: %w", err)
	}
	for _, g := range grants {
		if !strings.HasPrefix(string(g.Permission), classesPermissionPrefix) {
			continue
		}
		switch g.ResourceType {
		case common.RESOURCE_TYPE_ALL:
			return true, nil, nil
		case common.RESOURCE_TYPE_CHANNEL:
			if !slices.Contains(ids, g.ResourceID) {
				ids = append(ids, g.ResourceID)
			}
		}
	}
	return false, ids, nil
}

const classesPermissionPrefix = "classes."
