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

// oauthNewUser is a first-time social sign-in with a provider-verified email.
type oauthNewUser struct {
	Provider   string
	Subject    string
	Email      string
	Username   string
	FirstName  string
	LastName   *string
	NativeLang string
	// Returned when a concurrent signup took the email before this one committed.
	EmailTakenErr error
}

func createOAuthUser(ctx context.Context, repo *users.Repository, in *oauthNewUser) (users.User, error) {
	disabledHash, err := disabledPasswordHash()
	if err != nil {
		return users.User{}, err
	}

	var created users.User
	err = repo.Transaction(ctx, func(tx *gorm.DB) error {
		var err error
		created, err = insertOAuthUser(ctx, tx, repo, in, disabledHash)
		return err
	})
	if err == nil {
		return created, nil
	}

	// Concurrent callbacks for the same provider account can race before the
	// identity exists. If the other transaction won, read its identity and use
	// that account; never fall back to email based linking.
	identity, identityErr := repo.GetOAuthIdentity(ctx, in.Provider, in.Subject)
	if identityErr == nil {
		user, err := repo.GetUserByID(ctx, identity.UserID)
		if err != nil {
			return users.User{}, fmt.Errorf("load raced OAuth user: %w", err)
		}
		return user, nil
	}
	if _, ownerErr := repo.GetUserForEmail(ctx, in.Email); ownerErr == nil {
		return users.User{}, in.EmailTakenErr
	}
	return users.User{}, fmt.Errorf("create OAuth user: %w", err)
}

// Five random bytes encode to at least the eight characters used as a suffix.
const oauthUsernameSuffixBytes = 5

// oauthUsername uses the first candidate that has letters or digits, then adds
// a random suffix so a public profile name cannot be guessed from the email.
func oauthUsername(ctx context.Context, repo *users.Repository, fallback string, candidates ...string) (string, error) {
	base := fallback
	for _, candidate := range candidates {
		if cleaned := oauthUsernameBase(candidate); cleaned != "" {
			base = cleaned
			break
		}
	}
	for range 8 {
		raw, err := oauthRandom(oauthUsernameSuffixBytes)
		if err != nil {
			return "", err
		}
		candidate := base + strings.ToLower(raw[:8])
		exists, err := repo.GetExistingUserForUsername(ctx, candidate)
		if err != nil {
			return "", fmt.Errorf("check OAuth username: %w", err)
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", errors.New("could not generate an OAuth username")
}

func insertOAuthUser(ctx context.Context, tx *gorm.DB, repo *users.Repository, in *oauthNewUser, disabledHash string) (users.User, error) {
	userID, err := repo.CreateUser(ctx, tx, in.Email, in.Username, disabledHash, nil)
	if err != nil {
		return users.User{}, fmt.Errorf("create OAuth user: %w", err)
	}
	if err := repo.CreateProfile(ctx, tx, userID, in.FirstName, in.LastName, in.NativeLang); err != nil {
		return users.User{}, fmt.Errorf("create OAuth user profile: %w", err)
	}
	if err := repo.CreateUsersSettings(ctx, tx, userID, nil, nil); err != nil {
		return users.User{}, fmt.Errorf("create OAuth user settings: %w", err)
	}
	if err := repo.UpdateUserForVerification(ctx, tx, userID); err != nil {
		return users.User{}, fmt.Errorf("verify OAuth user email: %w", err)
	}
	if err := repo.CreateOAuthIdentity(ctx, tx, userID, in.Provider, in.Subject, &in.Email, true, time.Now().UTC()); err != nil {
		return users.User{}, fmt.Errorf("create OAuth identity: %w", err)
	}
	return users.User{
		ID:              userID,
		Email:           in.Email,
		Username:        in.Username,
		PasswordHash:    disabledHash,
		IsActive:        true,
		IsEmailVerified: true,
	}, nil
}
