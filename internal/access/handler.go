package access

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
)

type PermissionHandler struct {
	mine   *MyPermissionsService
	list   *ListPermissionsService
	grant  *GrantPermissionService
	revoke *RevokePermissionService
	logger *zap.Logger
}

func NewPermissionHandler(
	mine *MyPermissionsService, list *ListPermissionsService, grant *GrantPermissionService, revoke *RevokePermissionService, log *zap.Logger,
) *PermissionHandler {
	return &PermissionHandler{mine: mine, list: list, grant: grant, revoke: revoke, logger: log}
}

type InviteServices struct {
	Create  *CreateInviteService
	List    *ListInvitesService
	Resend  *ResendInviteService
	Revoke  *RevokeInviteService
	Preview *PreviewInviteService
	Accept  *AcceptInviteService
}

type InviteHandler struct {
	svc    InviteServices
	logger *zap.Logger
}

func NewInviteHandler(svc InviteServices, log *zap.Logger) *InviteHandler {
	return &InviteHandler{svc: svc, logger: log}
}

type (
	myPermissionsOutput = humautil.Output[humautil.APIResponse[MyPermissionsResponse]]
	permissionsOutput   = humautil.Output[humautil.APIResponse[[]PermissionResponse]]
	permissionOutput    = humautil.Output[humautil.APIResponse[PermissionResponse]]
	inviteOutput        = humautil.Output[humautil.APIResponse[InviteResponse]]
	invitesOutput       = humautil.Output[humautil.APIResponse[[]InviteResponse]]
	previewOutput       = humautil.Output[humautil.APIResponse[InvitePreviewResponse]]
	acceptOutput        = humautil.Output[humautil.APIResponse[AcceptInviteResponse]]
	messageOutput       = humautil.Output[humautil.APIResponse[humautil.MessageBody]]
)

type EmptyInput struct{}

type ListPermissionsInput struct {
	UserEmail string `query:"user_email" maxLength:"254" doc:"Only this user's grants; empty lists all"`
}

type IDInput struct {
	ID uuid.UUID `path:"id"`
}

type TokenInput struct {
	Token string `path:"token" maxLength:"128" doc:"Token from the invite link"`
}

func (h *PermissionHandler) GetMine(ctx context.Context, _ *EmptyInput) (*myPermissionsOutput, error) {
	resp, err := h.mine.Get(ctx, middleware.ContextUserID(ctx))
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *PermissionHandler) List(ctx context.Context, input *ListPermissionsInput) (*permissionsOutput, error) {
	resp, err := h.list.List(ctx, middleware.ContextUserID(ctx), input.UserEmail)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *PermissionHandler) Grant(ctx context.Context, input *humautil.Input[GrantPermissionRequest]) (*permissionOutput, error) {
	resp, err := h.grant.Grant(ctx, middleware.ContextUserID(ctx), &input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *PermissionHandler) Revoke(ctx context.Context, input *IDInput) (*messageOutput, error) {
	if err := h.revoke.Revoke(ctx, middleware.ContextUserID(ctx), input.ID); err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(humautil.MessageBody{Message: "permission revoked"}), nil
}

func (h *InviteHandler) Create(ctx context.Context, input *humautil.Input[CreateInviteRequest]) (*inviteOutput, error) {
	resp, err := h.svc.Create.Create(ctx, middleware.ContextUserID(ctx), &input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *InviteHandler) List(ctx context.Context, _ *EmptyInput) (*invitesOutput, error) {
	resp, err := h.svc.List.List(ctx, middleware.ContextUserID(ctx))
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *InviteHandler) Resend(ctx context.Context, input *IDInput) (*inviteOutput, error) {
	resp, err := h.svc.Resend.Resend(ctx, middleware.ContextUserID(ctx), input.ID)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *InviteHandler) Revoke(ctx context.Context, input *IDInput) (*messageOutput, error) {
	if err := h.svc.Revoke.Revoke(ctx, middleware.ContextUserID(ctx), input.ID); err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(humautil.MessageBody{Message: "invite revoked"}), nil
}

func (h *InviteHandler) Preview(ctx context.Context, input *TokenInput) (*previewOutput, error) {
	resp, err := h.svc.Preview.Preview(ctx, input.Token)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}

func (h *InviteHandler) Accept(ctx context.Context, input *TokenInput) (*acceptOutput, error) {
	resp, err := h.svc.Accept.Accept(ctx, middleware.ContextUserID(ctx), input.Token)
	if err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(resp), nil
}
