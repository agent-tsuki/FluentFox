package access

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

// permissions.manage is only ever global: a scoped manager could still grant
// global permissions, so a scope would be meaningless.
func validateGrant(g Grant) error {
	if !slices.Contains(common.ALL_PERMISSIONS, g.Permission) {
		return exceptions.ErrPermissionInvalid
	}
	switch g.ResourceType {
	case common.RESOURCE_TYPE_ALL:
		if g.ResourceID != "" {
			return exceptions.ErrPermissionInvalid
		}
	case common.RESOURCE_TYPE_CHANNEL, common.RESOURCE_TYPE_COURSE:
		if _, err := uuid.Parse(g.ResourceID); err != nil {
			return exceptions.WithCause(exceptions.ErrPermissionInvalid, err)
		}
	default:
		return exceptions.ErrPermissionInvalid
	}
	if g.Permission == common.PERMISSION_PERMISSIONS_MANAGE && g.ResourceType != common.RESOURCE_TYPE_ALL {
		return exceptions.ErrPermissionInvalid
	}
	return nil
}

func requireManager(ctx context.Context, authz Authorizer, actorID uuid.UUID) error {
	ok, err := authz.Has(ctx, actorID, common.PERMISSION_PERMISSIONS_MANAGE, common.ResourceRef{})
	if err != nil {
		return fmt.Errorf("check manager: %w", err)
	}
	if !ok {
		return exceptions.ErrForbidden
	}
	return nil
}

// Only super-admins may hand out permissions.manage, so a manager cannot
// create more managers.
func authorizeGrants(ctx context.Context, authz Authorizer, actorID uuid.UUID, grants []Grant) error {
	if err := requireManager(ctx, authz, actorID); err != nil {
		return err
	}
	for _, g := range grants {
		if err := validateGrant(g); err != nil {
			return err
		}
	}
	if !slices.ContainsFunc(grants, isManageGrant) {
		return nil
	}
	admin, err := authz.IsSuperAdmin(ctx, actorID)
	if err != nil {
		return fmt.Errorf("check super admin: %w", err)
	}
	if !admin {
		return exceptions.ErrForbidden
	}
	return nil
}

func isManageGrant(g Grant) bool {
	return g.Permission == common.PERMISSION_PERMISSIONS_MANAGE
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// Only the hash is stored, so a leaked table cannot be turned into working invite links.
func newInviteToken() (string, []byte, error) {
	raw := make([]byte, common.STAFF_INVITE_TOKEN_BYTES)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generate invite token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	return token, hashInviteToken(token), nil
}

func hashInviteToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// "mia@gmail.com" becomes "m***@gmail.com": enough for the invitee to
// recognise the address without the public preview revealing it.
func maskEmail(email string) string {
	local, domain, found := strings.Cut(email, "@")
	if !found || local == "" {
		return "***"
	}
	first, _ := utf8.DecodeRuneInString(local)
	return string(first) + "***@" + domain
}
