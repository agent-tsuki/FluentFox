package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
	"gorm.io/gorm"
)

var errOAuthProviderAlreadyLinked = exceptions.New(http.StatusConflict, "OAUTH_PROVIDER_ALREADY_LINKED", "this FoxSensei account is already connected to a different account from this provider")

// linkOAuthIdentityToEmailOwner signs a provider identity into the account that
// already owns the same email. Callers must only pass an email the provider has
// verified, because owning that inbox is the proof that joins the two logins.
// found is false when no account uses the email.
func linkOAuthIdentityToEmailOwner(
	ctx context.Context,
	repo *users.Repository,
	login *LoginService,
	provider, subject, email string,
) (user users.User, found bool, err error) {
	user, err = repo.GetUserForEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return users.User{}, false, nil
	}
	if err != nil {
		return users.User{}, false, fmt.Errorf("check OAuth email owner: %w", err)
	}
	if err := login.ValidateUserDetail(&user); err != nil {
		return users.User{}, true, err
	}
	if _, err := repo.GetOAuthIdentityForUser(ctx, user.ID, provider); err == nil {
		return users.User{}, true, errOAuthProviderAlreadyLinked
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return users.User{}, true, err
	}

	disabledHash, err := disabledPasswordHash()
	if err != nil {
		return users.User{}, true, err
	}
	now := time.Now().UTC()
	err = repo.Transaction(ctx, func(tx *gorm.DB) error {
		if !user.IsEmailVerified {
			// An unverified signup may have been created by someone who does not
			// own the inbox. Drop its password and sessions so only the verified
			// provider login can reach the account (pre-hijacking defence).
			if err := repo.UpdatePasswordHash(ctx, tx, user.ID, disabledHash, now); err != nil {
				return err
			}
			if err := repo.RevokeRefreshTokens(ctx, tx, user.ID); err != nil {
				return err
			}
			if err := repo.ExpireActiveVerifications(ctx, tx, user.ID, now); err != nil {
				return err
			}
			if err := repo.UpdateUserForVerification(ctx, tx, user.ID); err != nil {
				return err
			}
		}
		return repo.CreateOAuthIdentity(ctx, tx, user.ID, provider, subject, &email, true, now)
	})
	if err != nil {
		// A simultaneous callback may have created the same link first.
		identity, lookupErr := repo.GetOAuthIdentity(ctx, provider, subject)
		if lookupErr != nil || identity.UserID != user.ID {
			return users.User{}, true, fmt.Errorf("link OAuth identity by email: %w", err)
		}
	}
	user.IsEmailVerified = true
	return user, true, nil
}

// googleEmailIsAuthoritative follows Google's guidance: email_verified alone is
// only trustworthy for Gmail addresses and Workspace domains (hd claim). Other
// addresses on a Google account may not be controlled by its owner.
func googleEmailIsAuthoritative(email, hostedDomain string) bool {
	domain := email[strings.LastIndex(email, "@")+1:]
	if domain == "gmail.com" || domain == "googlemail.com" {
		return true
	}
	return hostedDomain != "" && strings.EqualFold(domain, hostedDomain)
}

// A social-only account still has a password-hash column today. Store a
// cryptographically random Argon2 hash so password sign-in always fails
// safely; no user knows or can derive this value.
func disabledPasswordHash() (string, error) {
	disabledPassword, err := oauthRandom(32)
	if err != nil {
		return "", fmt.Errorf("generate social-only password: %w", err)
	}
	argon := (&Argon2Config{}).defaultConfig()
	disabledHash, err := argon.hashStringWithSalt(disabledPassword)
	if err != nil {
		return "", fmt.Errorf("hash social-only password: %w", err)
	}
	return disabledHash, nil
}
