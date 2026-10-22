package access

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

func TestAcceptInviteService_Accept(t *testing.T) {
	const rawToken = "invite-token"
	userID, inviterID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	grant := Grant{Permission: common.PERMISSION_CHANNEL_EDIT, ResourceType: common.RESOURCE_TYPE_CHANNEL, ResourceID: uuid.NewString()}
	verified := users.User{ID: userID, Email: "Mia@Example.com", IsEmailVerified: true}

	tests := []struct {
		name      string
		invite    func(inv *StaffInvite)
		user      users.User
		token     string
		createErr error
		wantErr   error
	}{
		{name: "matching verified email gets the grants", user: verified, token: rawToken},
		{name: "different email is rejected", user: users.User{ID: userID, Email: "other@example.com", IsEmailVerified: true},
			token: rawToken, wantErr: exceptions.ErrInviteEmailMismatch},
		{name: "unverified email is rejected even when it matches", user: users.User{ID: userID, Email: "mia@example.com"},
			token: rawToken, wantErr: exceptions.ErrInviteEmailMismatch},
		{name: "expired invite is rejected", user: verified, token: rawToken,
			invite: func(inv *StaffInvite) { inv.ExpiresAt = now.Add(-time.Minute) }, wantErr: exceptions.ErrInviteExpired},
		{name: "accepted invite cannot be used again", user: verified, token: rawToken,
			invite: func(inv *StaffInvite) { inv.AcceptedAt = &now }, wantErr: exceptions.ErrInviteUsed},
		{name: "revoked invite is rejected", user: verified, token: rawToken,
			invite: func(inv *StaffInvite) { inv.RevokedAt = &now }, wantErr: exceptions.ErrInviteRevoked},
		{name: "unknown token is not found", user: verified, token: "wrong", wantErr: exceptions.ErrInviteNotFound},
		{name: "inviter cannot accept their own invite", user: users.User{ID: inviterID, Email: "mia@example.com", IsEmailVerified: true},
			token: rawToken, wantErr: exceptions.ErrPermissionSelfGrant},
		{name: "failed grant insert leaves the invite unused", user: verified, token: rawToken,
			createErr: errors.New("insert failed")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inv := StaffInvite{
				ID: uuid.New(), Email: "mia@example.com", TokenHash: hashInviteToken(rawToken), InvitedBy: inviterID,
				Grants: Grants{grant}, ExpiresAt: now.Add(time.Hour),
			}
			if tt.invite != nil {
				tt.invite(&inv)
			}
			repo := &fakeAcceptRepo{invite: &inv, createErr: tt.createErr}
			svc := NewAcceptInviteService(repo, &fakeUserLookup{byID: tt.user}, zap.NewNop())

			resp, err := svc.Accept(context.Background(), tt.user.ID, tt.token)

			wantFail := tt.wantErr != nil || tt.createErr != nil
			if wantFail {
				if err == nil || (tt.wantErr != nil && !errors.Is(err, tt.wantErr)) {
					t.Fatalf("Accept error = %v, want %v", err, tt.wantErr)
				}
				if repo.accepted || len(repo.created) > 0 {
					t.Errorf("failed accept wrote state: accepted=%v created=%d", repo.accepted, len(repo.created))
				}
				return
			}
			if err != nil {
				t.Fatalf("Accept: %v", err)
			}
			if !repo.accepted {
				t.Error("invite was not marked accepted")
			}
			if len(repo.created) != 1 || repo.created[0].UserID != userID || repo.created[0].grant() != grant {
				t.Errorf("created grants = %+v, want %+v for the user", repo.created, grant)
			}
			if *repo.created[0].GrantedBy != inviterID {
				t.Errorf("granted_by = %v, want the inviter", *repo.created[0].GrantedBy)
			}
			if len(resp.Grants) != 1 {
				t.Errorf("response grants = %d, want 1", len(resp.Grants))
			}
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{name: "keeps first letter and domain", in: "mia@gmail.com", want: "m***@gmail.com"},
		{name: "multibyte first letter is kept whole", in: "ゆき@example.jp", want: "ゆ***@example.jp"},
		{name: "address without @ is fully hidden", in: "nobody", want: "***"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskEmail(tt.in); got != tt.want {
				t.Errorf("maskEmail(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
