package access

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/token"
)

func TestChecker_Has(t *testing.T) {
	user := uuid.New()
	channel := uuid.NewString()
	otherChannel := uuid.NewString()
	course := uuid.New()
	orphanCourse := uuid.New()
	channelRef := common.ResourceRef{Type: common.RESOURCE_TYPE_CHANNEL, ID: channel}
	courseRef := common.ResourceRef{Type: common.RESOURCE_TYPE_COURSE, ID: course.String()}

	tests := []struct {
		name   string
		admin  bool
		grants []Grant
		perm   common.Permission
		res    common.ResourceRef
		want   bool
	}{
		{name: "super-admin has every permission without grants", admin: true,
			perm: common.PERMISSION_PERMISSIONS_MANAGE, want: true},
		{name: "no grant means no access",
			perm: common.PERMISSION_CHANNEL_EDIT, res: channelRef, want: false},
		{name: "global grant covers any channel",
			grants: []Grant{{Permission: common.PERMISSION_CHANNEL_EDIT}},
			perm:   common.PERMISSION_CHANNEL_EDIT, res: channelRef, want: true},
		{name: "channel grant covers that channel",
			grants: []Grant{{Permission: common.PERMISSION_CHANNEL_EDIT, ResourceType: common.RESOURCE_TYPE_CHANNEL, ResourceID: channel}},
			perm:   common.PERMISSION_CHANNEL_EDIT, res: channelRef, want: true},
		{name: "channel grant does not cover another channel",
			grants: []Grant{{Permission: common.PERMISSION_CHANNEL_EDIT, ResourceType: common.RESOURCE_TYPE_CHANNEL, ResourceID: otherChannel}},
			perm:   common.PERMISSION_CHANNEL_EDIT, res: channelRef, want: false},
		{name: "channel grant covers the channel's course",
			grants: []Grant{{Permission: common.PERMISSION_COURSE_EDIT, ResourceType: common.RESOURCE_TYPE_CHANNEL, ResourceID: channel}},
			perm:   common.PERMISSION_COURSE_EDIT, res: courseRef, want: true},
		{name: "channel grant does not cover a course with no channel row",
			grants: []Grant{{Permission: common.PERMISSION_COURSE_EDIT, ResourceType: common.RESOURCE_TYPE_CHANNEL, ResourceID: channel}},
			perm:   common.PERMISSION_COURSE_EDIT, res: common.ResourceRef{Type: common.RESOURCE_TYPE_COURSE, ID: orphanCourse.String()}, want: false},
		{name: "course grant covers that course",
			grants: []Grant{{Permission: common.PERMISSION_COURSE_EDIT, ResourceType: common.RESOURCE_TYPE_COURSE, ResourceID: course.String()}},
			perm:   common.PERMISSION_COURSE_EDIT, res: courseRef, want: true},
		{name: "a different permission on the resource does not count",
			grants: []Grant{{Permission: common.PERMISSION_CHANNEL_MEDIA, ResourceType: common.RESOURCE_TYPE_CHANNEL, ResourceID: channel}},
			perm:   common.PERMISSION_CHANNEL_EDIT, res: channelRef, want: false},
		{name: "scoped grant does not give the global permission",
			grants: []Grant{{Permission: common.PERMISSION_CHANNEL_EDIT, ResourceType: common.RESOURCE_TYPE_CHANNEL, ResourceID: channel}},
			perm:   common.PERMISSION_CHANNEL_EDIT, res: common.ResourceRef{}, want: false},
		{name: "malformed course id still checks direct grants",
			grants: []Grant{{Permission: common.PERMISSION_COURSE_EDIT}},
			perm:   common.PERMISSION_COURSE_EDIT, res: common.ResourceRef{Type: common.RESOURCE_TYPE_COURSE, ID: "not-a-uuid"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeCheckerRepo{
				admins:         map[uuid.UUID]bool{user: tt.admin},
				grants:         map[uuid.UUID][]Grant{user: tt.grants},
				courseChannels: map[uuid.UUID]string{course: channel},
			}
			got, err := NewChecker(repo, zap.NewNop()).Has(context.Background(), user, tt.perm, tt.res)
			if err != nil {
				t.Fatalf("Has: %v", err)
			}
			if got != tt.want {
				t.Errorf("Has = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChecker_Require(t *testing.T) {
	user := uuid.New()
	tests := []struct {
		name    string
		ctx     context.Context
		admin   bool
		repoErr error
		wantErr error
	}{
		{name: "allowed user passes", ctx: withUser(user), admin: true},
		{name: "user without grant is forbidden", ctx: withUser(user), wantErr: exceptions.ErrForbidden},
		{name: "request without a user is forbidden", ctx: context.Background(), admin: true, wantErr: exceptions.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeCheckerRepo{admins: map[uuid.UUID]bool{user: tt.admin}, err: tt.repoErr}
			err := NewChecker(repo, zap.NewNop()).Require(tt.ctx, common.PERMISSION_CHANNEL_EDIT, common.ResourceRef{})
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Require error = %v, want %v", err, tt.wantErr)
			}
		})
	}

	t.Run("database error is returned, not treated as forbidden", func(t *testing.T) {
		repo := &fakeCheckerRepo{err: errors.New("db down")}
		err := NewChecker(repo, zap.NewNop()).Require(withUser(user), common.PERMISSION_CHANNEL_EDIT, common.ResourceRef{})
		if err == nil || errors.Is(err, exceptions.ErrForbidden) {
			t.Errorf("Require error = %v, want the database error", err)
		}
	})
}

func withUser(id uuid.UUID) context.Context {
	return middleware.ContextWithClaims(context.Background(), &token.Claims{UserID: id, EmailVerified: true})
}
