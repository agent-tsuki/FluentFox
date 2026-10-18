package auth

import (
	"context"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// Matches chk_users_profile_oauth_image_url.
	oauthImageURLMaxLen = 1024
	xImageSmallSuffix   = "_normal."
	xImageLargeSuffix   = "_400x400."
	// Lower-case on purpose: chk_users_profile_oauth_image_url compares with LIKE.
	httpsPrefix = "https://"
)

type oauthImageStore interface {
	SetOAuthImage(ctx context.Context, userID uuid.UUID, imageURL string) error
}

// saveOAuthImage records the provider's picture after a social login. The
// profile shows it only while the user has not uploaded or picked an avatar.
// It is best effort: a failure must not undo a login that already succeeded.
func saveOAuthImage(ctx context.Context, store oauthImageStore, userID uuid.UUID, raw string, log *zap.Logger) {
	if !validOAuthImageURL(raw) {
		return
	}
	if err := store.SetOAuthImage(ctx, userID, raw); err != nil {
		log.Warn("save OAuth profile picture failed", zap.Stringer("user_id", userID), zap.Error(err))
	}
}

// Only https URLs are stored: the frontend puts this value straight into <img src>.
func validOAuthImageURL(raw string) bool {
	if len(raw) > oauthImageURLMaxLen || !strings.HasPrefix(raw, httpsPrefix) {
		return false
	}
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Host != ""
}

// X returns a 48px "_normal" image; the same URL with "_400x400" is the
// large version.
func xLargeImage(raw string) string {
	return strings.Replace(raw, xImageSmallSuffix, xImageLargeSuffix, 1)
}
