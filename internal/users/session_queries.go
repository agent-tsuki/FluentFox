package users

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/fluentfox/api/internal/common"
)

// Location sources stored in refresh_tokens.location_source.
const (
	LocationSourceIP  = "ip"
	LocationSourceGPS = "gps"
)

const (
	columnLastUsedAt = "last_used_at"
	columnIsRevoked  = "is_revoked"
)

// SessionLocation is where a session was opened. Empty strings and nil
// pointers are stored as NULL ("unknown").
type SessionLocation struct {
	CountryCode string
	Region      string
	City        string
	Latitude    *float64
	Longitude   *float64
	Source      string
	AccuracyM   *int
	IsVPN       *bool
}

// RoundCoordinate keeps a map pin accurate to about 110 m without storing
// the exact spot a person logged in from.
func RoundCoordinate(v float64) float64 {
	return math.Round(v*common.LOCATION_COORD_SCALE) / common.LOCATION_COORD_SCALE
}

type NewSession struct {
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	UserAgent string
	IPPrefix  string
	Location  SessionLocation
}

func (r *Repository) CreateSession(ctx context.Context, tx *gorm.DB, s *NewSession, now time.Time) (uuid.UUID, error) {
	row := RefreshToken{
		UserID:     s.UserID,
		TokenHash:  s.TokenHash,
		ExpiresAt:  s.ExpiresAt,
		UserAgent:  nullIfEmpty(s.UserAgent),
		IPPrefix:   nullIfEmpty(s.IPPrefix),
		LastUsedAt: now,
	}
	applyLocation(&row, &s.Location)
	if err := tx.WithContext(ctx).Create(&row).Error; err != nil {
		return uuid.Nil, fmt.Errorf("create session: %w", err)
	}
	return row.ID, nil
}

// PruneSessions runs inside each login, since there is no background job:
// it deletes expired rows and rows revoked before revokedBefore, then revokes
// the least recently used active sessions so at most keepActive remain.
func (r *Repository) PruneSessions(ctx context.Context, tx *gorm.DB, userID uuid.UUID, now, revokedBefore time.Time, keepActive int) error {
	if err := tx.WithContext(ctx).
		Where("user_id = ? AND (expires_at < ? OR (is_revoked AND updated_at < ?))", userID, now, revokedBefore).
		Delete(&RefreshToken{}).Error; err != nil {
		return fmt.Errorf("prune sessions: delete old: %w", err)
	}

	oldest := tx.Model(&RefreshToken{}).Select("id").
		Where("user_id = ? AND is_revoked = false", userID).
		Order(columnLastUsedAt + " DESC").Offset(keepActive)
	if err := tx.WithContext(ctx).Model(&RefreshToken{}).
		Where("id IN (?)", oldest).
		Updates(map[string]any{columnIsRevoked: true, columnUpdatedAt: now}).Error; err != nil {
		return fmt.Errorf("prune sessions: revoke over limit: %w", err)
	}
	return nil
}

func (r *Repository) SetLastLogin(ctx context.Context, tx *gorm.DB, userID uuid.UUID, at time.Time) error {
	if err := tx.WithContext(ctx).Model(&User{}).Where("id = ?", userID).
		UpdateColumn("last_login_at", at).Error; err != nil {
		return fmt.Errorf("set last login: %w", err)
	}
	return nil
}

// RotateSession swaps the token hash only if it is still oldHash, so two
// refreshes racing with the same token cannot both succeed.
func (r *Repository) RotateSession(ctx context.Context, sessionID uuid.UUID, oldHash, newHash string, expiresAt, now time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("id = ? AND token_hash = ? AND is_revoked = false", sessionID, oldHash).
		Updates(map[string]any{"token_hash": newHash, columnExpiresAt: expiresAt, columnLastUsedAt: now, columnUpdatedAt: now})
	if result.Error != nil {
		return false, fmt.Errorf("rotate session: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (r *Repository) RevokeSessionByID(ctx context.Context, sessionID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("id = ?", sessionID).
		Update(columnIsRevoked, true).Error; err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (r *Repository) ListActiveSessions(ctx context.Context, userID uuid.UUID, now time.Time) ([]RefreshToken, error) {
	var rows []RefreshToken
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_revoked = false AND expires_at >= ?", userID, now).
		Order(columnLastUsedAt + " DESC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	return rows, nil
}

// The user_id filter means a user can only ever revoke their own sessions.
func (r *Repository) RevokeUserSession(ctx context.Context, userID, sessionID uuid.UUID) (bool, error) {
	result := r.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("id = ? AND user_id = ? AND is_revoked = false", sessionID, userID).
		Update(columnIsRevoked, true)
	if result.Error != nil {
		return false, fmt.Errorf("revoke user session: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (r *Repository) RevokeOtherSessions(ctx context.Context, userID, keepID uuid.UUID) (int64, error) {
	result := r.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("user_id = ? AND id <> ? AND is_revoked = false", userID, keepID).
		Update(columnIsRevoked, true)
	if result.Error != nil {
		return 0, fmt.Errorf("revoke other sessions: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// UpdateSessionLocation replaces the IP-based location with a better one
// (the browser's GPS). It only touches the caller's own active session, and
// keeps is_vpn: GPS says where the device is, not how it connects.
func (r *Repository) UpdateSessionLocation(ctx context.Context, userID, sessionID uuid.UUID, loc *SessionLocation) (bool, error) {
	var row RefreshToken
	applyLocation(&row, loc)
	result := r.db.WithContext(ctx).Model(&RefreshToken{}).
		Where("id = ? AND user_id = ? AND is_revoked = false", sessionID, userID).
		Updates(map[string]any{
			"country_code":        row.CountryCode,
			"region":              row.Region,
			"city":                row.City,
			"latitude":            row.Latitude,
			"longitude":           row.Longitude,
			"location_source":     row.LocationSource,
			"location_accuracy_m": row.LocationAccuracyM,
		})
	if result.Error != nil {
		return false, fmt.Errorf("update session location: %w", result.Error)
	}
	return result.RowsAffected == 1, nil
}

func applyLocation(row *RefreshToken, loc *SessionLocation) {
	row.CountryCode = nullIfEmpty(loc.CountryCode)
	row.Region = nullIfEmpty(loc.Region)
	row.City = nullIfEmpty(loc.City)
	row.Latitude = loc.Latitude
	row.Longitude = loc.Longitude
	row.LocationSource = nullIfEmpty(loc.Source)
	row.LocationAccuracyM = loc.AccuracyM
	row.IsVPN = loc.IsVPN
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
