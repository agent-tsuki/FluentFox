package profile

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/users"
	"github.com/fluentfox/api/pkg/exceptions"
)

type SessionListService struct {
	repo   SessionRepo
	logger *zap.Logger
}

func NewSessionListService(repo SessionRepo, log *zap.Logger) *SessionListService {
	return &SessionListService{repo: repo, logger: log}
}

func (s *SessionListService) List(ctx context.Context, userID, currentSessionID uuid.UUID) ([]SessionResponse, error) {
	rows, err := s.repo.ListActiveSessions(ctx, userID, time.Now().UTC())
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	out := make([]SessionResponse, 0, len(rows))
	for i := range rows {
		out = append(out, toSessionResponse(&rows[i], currentSessionID))
	}
	return out, nil
}

type SessionRevokeService struct {
	repo   SessionRepo
	logger *zap.Logger
}

func NewSessionRevokeService(repo SessionRepo, log *zap.Logger) *SessionRevokeService {
	return &SessionRevokeService{repo: repo, logger: log}
}

// The device stays signed in until its short-lived access token expires;
// revoking stops it from getting a new one.
func (s *SessionRevokeService) Revoke(ctx context.Context, userID, currentSessionID, sessionID uuid.UUID) error {
	if sessionID == currentSessionID {
		return exceptions.ErrCannotRevokeCurrentSession
	}
	ok, err := s.repo.RevokeUserSession(ctx, userID, sessionID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if !ok {
		return exceptions.ErrSessionNotFound
	}
	s.logger.Info("session revoked", zap.Stringer("user_id", userID), zap.Stringer("session_id", sessionID))
	return nil
}

// A token issued before sessions existed has no sid, so "others" is unknown;
// the client refreshes (getting a sid) and retries.
func (s *SessionRevokeService) RevokeOthers(ctx context.Context, userID, currentSessionID uuid.UUID) (int64, error) {
	if currentSessionID == uuid.Nil {
		return 0, exceptions.ErrTokenInvalid
	}
	count, err := s.repo.RevokeOtherSessions(ctx, userID, currentSessionID)
	if err != nil {
		return 0, fmt.Errorf("revoke other sessions: %w", err)
	}
	s.logger.Info("other sessions revoked", zap.Stringer("user_id", userID), zap.Int64("count", count))
	return count, nil
}

type SessionLocationService struct {
	repo   SessionRepo
	logger *zap.Logger
}

func NewSessionLocationService(repo SessionRepo, log *zap.Logger) *SessionLocationService {
	return &SessionLocationService{repo: repo, logger: log}
}

// UpdateCurrent stores the browser's GPS position for this device. The IP
// place name is kept as a label unless the login came through a VPN, where
// it is likely wrong and would contradict the GPS pin.
func (s *SessionLocationService) UpdateCurrent(ctx context.Context, userID, currentSessionID uuid.UUID, req UpdateLocationRequest) (SessionResponse, error) {
	if currentSessionID == uuid.Nil {
		return SessionResponse{}, exceptions.ErrTokenInvalid
	}
	current, err := s.findSession(ctx, userID, currentSessionID)
	if err != nil {
		return SessionResponse{}, err
	}

	lat, lng := users.RoundCoordinate(req.Latitude), users.RoundCoordinate(req.Longitude)
	accuracy := req.AccuracyM
	loc := users.SessionLocation{Latitude: &lat, Longitude: &lng, Source: users.LocationSourceGPS, AccuracyM: &accuracy}
	if current.IsVPN == nil || !*current.IsVPN {
		loc.CountryCode, loc.Region, loc.City = deref(current.CountryCode), deref(current.Region), deref(current.City)
	}

	ok, err := s.repo.UpdateSessionLocation(ctx, userID, currentSessionID, &loc)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("update session location: %w", err)
	}
	if !ok {
		return SessionResponse{}, exceptions.ErrSessionNotFound
	}

	gps := users.LocationSourceGPS
	current.Latitude, current.Longitude, current.LocationSource, current.LocationAccuracyM = &lat, &lng, &gps, &accuracy
	if current.IsVPN != nil && *current.IsVPN {
		current.CountryCode, current.Region, current.City = nil, nil, nil
	}
	return toSessionResponse(&current, currentSessionID), nil
}

func (s *SessionLocationService) findSession(ctx context.Context, userID, sessionID uuid.UUID) (users.RefreshToken, error) {
	rows, err := s.repo.ListActiveSessions(ctx, userID, time.Now().UTC())
	if err != nil {
		return users.RefreshToken{}, fmt.Errorf("find session: %w", err)
	}
	for i := range rows {
		if rows[i].ID == sessionID {
			return rows[i], nil
		}
	}
	return users.RefreshToken{}, exceptions.ErrSessionNotFound
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
