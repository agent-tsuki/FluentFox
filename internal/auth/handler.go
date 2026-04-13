package auth

import (
	"context"

	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/validator"
	"go.uber.org/zap"
)

type Handler struct {
	authService   *AuthService
	verifyService *TokenVerificationService
	loginService *LoginService
	logger        *zap.Logger
	validate      *validator.Validator
}

func NewHandler(authService *AuthService, verifyService *TokenVerificationService, loginService *LoginService, log *zap.Logger, v *validator.Validator) *Handler {
	return &Handler{
		authService:   authService,
		verifyService: verifyService,
		loginService: loginService,
		logger:        log,
		validate:      v,
	}
}

type AuthVerifyInput struct {
	Token string `query:"token" doc:"Email verification token sent to the user's inbox"`
}

// AuthRegister handles POST /auth/register.
func (h *Handler) AuthRegister(ctx context.Context, input *humautil.Input[RegisterRequest]) (*humautil.Output[humautil.MessageBody], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	if err := h.authService.registerUser(ctx, input.Body); err != nil {
		return nil, humautil.MapErr(err, log)
	}

	return &humautil.Output[humautil.MessageBody]{
		Body: humautil.MessageBody{Message: "registration successful, check your email to verify"},
	}, nil
}

// AuthVerify handles POST /auth/verify?token=<token>.
func (h *Handler) AuthVerify(ctx context.Context, input *AuthVerifyInput) (*humautil.Output[humautil.MessageBody], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	if err := h.verifyService.VerifyUserToken(ctx, input.Token); err != nil {
		return nil, humautil.MapErr(err, log)
	}

	return &humautil.Output[humautil.MessageBody]{
		Body: humautil.MessageBody{Message: "email verified successfully"},
	}, nil
}


// Login handles POST /auth/login.
func (h *Handler) Login(ctx context.Context, input *humautil.Input[LoginRequest]) (*humautil.Output[LoginResponse], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	resp, err := h.loginService.Login(ctx, input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, log)
	}

	return &humautil.Output[LoginResponse]{Body: resp}, nil
}

// Refresh handles POST /auth/refresh.
func (h *Handler) Refresh(ctx context.Context, input *humautil.Input[RefreshTokenRequest]) (*humautil.Output[LoginResponse], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	resp, err := h.loginService.RefreshToken(ctx, input.Body.RefreshToken)
	if err != nil {
		return nil, humautil.MapErr(err, log)
	}

	return &humautil.Output[LoginResponse]{Body: resp}, nil
}

// Logout handles POST /auth/logout.
func (h *Handler) Logout(ctx context.Context, input *humautil.Input[RefreshTokenRequest]) (*humautil.Output[humautil.MessageBody], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	if err := h.loginService.Logout(ctx, input.Body.RefreshToken); err != nil {
		return nil, humautil.MapErr(err, log)
	}

	return &humautil.Output[humautil.MessageBody]{
		Body: humautil.MessageBody{Message: "logged out successfully"},
	}, nil
}

