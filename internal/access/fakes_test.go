package access

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
)

// fakeCheckerRepo stores grants in memory and answers HasGrant the way the
// SQL does: any of the given resources matching a stored grant.
type fakeCheckerRepo struct {
	admins         map[uuid.UUID]bool
	grants         map[uuid.UUID][]Grant
	courseChannels map[uuid.UUID]string
	err            error
}

func (f *fakeCheckerRepo) IsSuperAdmin(_ context.Context, userID uuid.UUID) (bool, error) {
	return f.admins[userID], f.err
}

func (f *fakeCheckerRepo) HasGrant(_ context.Context, userID uuid.UUID, perm common.Permission, resources []common.ResourceRef) (bool, error) {
	for _, g := range f.grants[userID] {
		for _, res := range resources {
			if g.Permission == perm && g.Resource() == res {
				return true, nil
			}
		}
	}
	return false, nil
}

func (f *fakeCheckerRepo) FindCourseChannelID(_ context.Context, courseID uuid.UUID) (string, bool, error) {
	id, ok := f.courseChannels[courseID]
	return id, ok, nil
}

type fakeAcceptRepo struct {
	invite    *StaffInvite
	created   []UserPermission
	accepted  bool
	createErr error
}

func (f *fakeAcceptRepo) Transaction(_ context.Context, fn func(*gorm.DB) error) error {
	created, accepted := f.created, f.accepted
	if err := fn(nil); err != nil {
		f.created, f.accepted = created, accepted
		return err
	}
	return nil
}

func (f *fakeAcceptRepo) LockInviteByHash(_ context.Context, _ *gorm.DB, tokenHash []byte) (StaffInvite, error) {
	if f.invite == nil || string(f.invite.TokenHash) != string(tokenHash) {
		return StaffInvite{}, mapInviteErr(gorm.ErrRecordNotFound)
	}
	return *f.invite, nil
}

func (f *fakeAcceptRepo) CreatePermissions(_ context.Context, _ *gorm.DB, perms []UserPermission) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, perms...)
	return nil
}

func (f *fakeAcceptRepo) MarkInviteAccepted(_ context.Context, _ *gorm.DB, _, _ uuid.UUID, _ time.Time) error {
	f.accepted = true
	return nil
}

// fakeUserLookup embeds the interface so an unexpected call panics.
type fakeUserLookup struct {
	UserLookup
	byID users.User
}

func (f *fakeUserLookup) GetUserByID(_ context.Context, _ uuid.UUID) (users.User, error) {
	return f.byID, nil
}
