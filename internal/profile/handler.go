package profile

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

type ProfileHandler struct {
	profile *ProfileService
	update  *UpdateProfileService
	avatars *AvatarService
	presets *AvatarResolver
	logger  *zap.Logger
}

func NewProfileHandler(
	profile *ProfileService, update *UpdateProfileService, avatars *AvatarService, presets *AvatarResolver, log *zap.Logger,
) *ProfileHandler {
	return &ProfileHandler{profile: profile, update: update, avatars: avatars, presets: presets, logger: log}
}

type SessionHandler struct {
	sessions *SessionListService
	revoke   *SessionRevokeService
	location *SessionLocationService
	logger   *zap.Logger
}

func NewSessionHandler(sessions *SessionListService, revoke *SessionRevokeService, location *SessionLocationService, log *zap.Logger) *SessionHandler {
	return &SessionHandler{sessions: sessions, revoke: revoke, location: location, logger: log}
}

// Response shapes, named so handler signatures stay readable.
type (
	profileOutput  = humautil.Output[humautil.APIResponse[ProfileResponse]]
	avatarOutput   = humautil.Output[humautil.APIResponse[AvatarResponse]]
	presetsOutput  = humautil.Output[humautil.APIResponse[[]AvatarPresetResponse]]
	sessionsOutput = humautil.Output[humautil.APIResponse[[]SessionResponse]]
	sessionOutput  = humautil.Output[humautil.APIResponse[SessionResponse]]
	messageOutput  = humautil.Output[humautil.APIResponse[humautil.MessageBody]]
	revokedOutput  = humautil.Output[humautil.APIResponse[RevokeOthersResponse]]
)

type EmptyInput struct{}

// The bytes are the image itself (Content-Type image/jpeg or image/png); its
// real type is sniffed from the bytes, not trusted from the header.
type AvatarUploadInput struct {
	RawBody []byte `contentType:"image/jpeg"`
}

type SessionIDInput struct {
	ID uuid.UUID `path:"id" doc:"Session id from GET /profile/sessions"`
}

func (h *ProfileHandler) GetProfile(ctx context.Context, _ *EmptyInput) (*profileOutput, error) {
	resp, err := h.profile.Get(ctx, middleware.ContextUserID(ctx), middleware.ContextSessionID(ctx))
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *ProfileHandler) UpdateProfile(ctx context.Context, input *humautil.Input[UpdateProfileRequest]) (*profileOutput, error) {
	resp, err := h.update.Update(ctx, middleware.ContextUserID(ctx), middleware.ContextSessionID(ctx), &input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *ProfileHandler) ListAvatarPresets(_ context.Context, _ *EmptyInput) (*presetsOutput, error) {
	return humautil.OK(h.presets.Presets()), nil
}

func (h *ProfileHandler) UploadAvatar(ctx context.Context, input *AvatarUploadInput) (*avatarOutput, error) {
	resp, err := h.avatars.Upload(ctx, middleware.ContextUserID(ctx), input.RawBody)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *ProfileHandler) RemoveAvatar(ctx context.Context, _ *EmptyInput) (*avatarOutput, error) {
	resp, err := h.avatars.Remove(ctx, middleware.ContextUserID(ctx))
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *SessionHandler) ListSessions(ctx context.Context, _ *EmptyInput) (*sessionsOutput, error) {
	resp, err := h.sessions.List(ctx, middleware.ContextUserID(ctx), middleware.ContextSessionID(ctx))
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *SessionHandler) RevokeSession(ctx context.Context, input *SessionIDInput) (*messageOutput, error) {
	err := h.revoke.Revoke(ctx, middleware.ContextUserID(ctx), middleware.ContextSessionID(ctx), input.ID)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(humautil.MessageBody{Message: "device signed out"}), nil
}

func (h *SessionHandler) RevokeOtherSessions(ctx context.Context, _ *EmptyInput) (*revokedOutput, error) {
	count, err := h.revoke.RevokeOthers(ctx, middleware.ContextUserID(ctx), middleware.ContextSessionID(ctx))
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(RevokeOthersResponse{Revoked: count}), nil
}

func (h *SessionHandler) UpdateCurrentLocation(ctx context.Context, input *humautil.Input[UpdateLocationRequest]) (*sessionOutput, error) {
	resp, err := h.location.UpdateCurrent(ctx, middleware.ContextUserID(ctx), middleware.ContextSessionID(ctx), input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}
