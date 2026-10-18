package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/fluentfox/api/pkg/humautil"
	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/validator"
	"go.uber.org/zap"
)

type Handler struct {
	authService   *AuthService
	verifyService *TokenVerificationService
	resendService *ResendVerificationService
	loginService  *LoginService
	changeEmail   *ChangeEmailService
	passwordReset *PasswordResetService
	logger        *zap.Logger
	validate      *validator.Validator
	refreshMaxAge int
}

func NewHandler(
	authService *AuthService,
	verifyService *TokenVerificationService,
	resendService *ResendVerificationService,
	loginService *LoginService,
	changeEmail *ChangeEmailService,
	passwordReset *PasswordResetService,
	log *zap.Logger,
	v *validator.Validator,
	refreshExpiryDays int,
) *Handler {
	return &Handler{
		authService:   authService,
		verifyService: verifyService,
		resendService: resendService,
		loginService:  loginService,
		changeEmail:   changeEmail,
		passwordReset: passwordReset,
		logger:        log,
		validate:      v,
		refreshMaxAge: int((time.Duration(refreshExpiryDays) * 24 * time.Hour).Seconds()),
	}
}

const refreshCookieName = "__Host-ff_refresh"

func refreshCookie(raw string, maxAge int) http.Cookie {
	return http.Cookie{
		Name:     refreshCookieName,
		Value:    raw,
		Path:     "/",
		MaxAge:   maxAge,
		Expires:  time.Now().UTC().Add(time.Duration(maxAge) * time.Second),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func expiredRefreshCookie() http.Cookie {
	return http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(1, 0).UTC(),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func sessionOK[T any](data T, rawRefresh string, maxAge int) *SessionOutput[T] {
	return &SessionOutput[T]{
		SetCookie:    []http.Cookie{refreshCookie(rawRefresh, maxAge)},
		CacheControl: "no-store",
		Body:         humautil.APIResponse[T]{Status: "success", Data: data},
	}
}

func (h *Handler) ForgotPassword(ctx context.Context, input *humautil.Input[ForgotPasswordRequest]) (*humautil.Output[humautil.APIResponse[humautil.MessageBody]], error) {
	if err := h.passwordReset.RequestCode(ctx, input.Body); err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(humautil.MessageBody{Message: "If an account exists for that email, we sent a password reset code."}), nil
}

func (h *Handler) VerifyPasswordResetCode(ctx context.Context, input *humautil.Input[VerifyPasswordResetCodeRequest]) (*humautil.Output[humautil.APIResponse[humautil.MessageBody]], error) {
	if err := h.passwordReset.VerifyCode(ctx, input.Body); err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(humautil.MessageBody{Message: "We sent a password reset link to your email."}), nil
}

func (h *Handler) ResetPassword(ctx context.Context, input *humautil.Input[ResetPasswordRequest]) (*humautil.Output[humautil.APIResponse[humautil.MessageBody]], error) {
	if err := h.passwordReset.ResetPassword(ctx, input.Body); err != nil {
		return nil, humautil.MapErr(err, middleware.LoggerFromContext(ctx, h.logger))
	}
	return humautil.OK(humautil.MessageBody{Message: "password reset successfully"}), nil
}

type AuthVerifyInput struct {
	Token string `query:"token" doc:"Email verification token sent to the user's inbox"`
}

// AuthRegister handles POST /auth/register.
func (h *Handler) AuthRegister(ctx context.Context, input *humautil.Input[RegisterRequest]) (*SessionOutput[RegisterResponse], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	session, err := h.authService.registerUser(ctx, &input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, log)
	}

	return sessionOK(RegisterResponse{
		Message:       "registration successful, check your email to verify",
		LoginResponse: session,
	}, session.RefreshToken, h.refreshMaxAge), nil
}

// AuthVerify handles POST /auth/verify?token=<token>.
func (h *Handler) AuthVerify(ctx context.Context, input *AuthVerifyInput) (*humautil.Output[humautil.APIResponse[humautil.MessageBody]], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	if err := h.verifyService.VerifyUserToken(ctx, input.Token); err != nil {
		return nil, humautil.MapErr(err, log)
	}

	return humautil.OK(humautil.MessageBody{Message: "email verified successfully"}), nil
}

// Login handles POST /auth/login.
func (h *Handler) Login(ctx context.Context, input *humautil.Input[LoginRequest]) (*SessionOutput[LoginResponse], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	resp, err := h.loginService.Login(ctx, input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, log)
	}
	// No session yet, so no refresh cookie: the client must send the emailed code.
	if resp.LoginChallenge != nil {
		return &SessionOutput[LoginResponse]{
			CacheControl: "no-store",
			Body:         humautil.APIResponse[LoginResponse]{Status: "success", Data: resp},
		}, nil
	}

	return sessionOK(resp, resp.RefreshToken, h.refreshMaxAge), nil
}

// VerifyLoginCode handles POST /auth/login/verify-code.
func (h *Handler) VerifyLoginCode(ctx context.Context, input *humautil.Input[VerifyLoginCodeRequest]) (*SessionOutput[LoginResponse], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	resp, err := h.loginService.VerifyLoginCode(ctx, input.Body)
	if err != nil {
		return nil, humautil.MapErr(err, log)
	}

	return sessionOK(resp, resp.RefreshToken, h.refreshMaxAge), nil
}

// Refresh handles POST /auth/refresh.
func (h *Handler) Refresh(ctx context.Context, input *RefreshCookieInput) (*SessionOutput[LoginResponse], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	resp, err := h.loginService.RefreshToken(ctx, input.RefreshToken)
	if err != nil {
		return nil, humautil.MapErr(err, log)
	}

	return sessionOK(resp, resp.RefreshToken, h.refreshMaxAge), nil
}

// ResendVerification handles POST /auth/verification/resend.
func (h *Handler) ResendVerification(ctx context.Context, _ *EmptyInput) (*humautil.Output[humautil.APIResponse[humautil.MessageBody]], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	if err := h.resendService.ResendForUser(ctx, middleware.ContextUserID(ctx)); err != nil {
		return nil, humautil.MapErr(err, log)
	}

	return humautil.OK(humautil.MessageBody{Message: "verification email sent, please check your inbox"}), nil
}

// ChangeEmail handles POST /auth/change-email.
func (h *Handler) ChangeEmail(
	ctx context.Context, input *humautil.Input[ChangeEmailRequest],
) (*humautil.Output[humautil.APIResponse[humautil.MessageBody]], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	if err := h.changeEmail.ChangeEmail(ctx, middleware.ContextUserID(ctx), input.Body); err != nil {
		return nil, humautil.MapErr(err, log)
	}

	return humautil.OK(humautil.MessageBody{Message: "email updated, check your new inbox to verify"}), nil
}

// Logout handles POST /auth/logout.
func (h *Handler) Logout(ctx context.Context, input *RefreshCookieInput) (*SessionOutput[humautil.MessageBody], error) {
	log := middleware.LoggerFromContext(ctx, h.logger)

	if err := h.loginService.Logout(ctx, input.RefreshToken); err != nil {
		return nil, humautil.MapErr(err, log)
	}

	return &SessionOutput[humautil.MessageBody]{
		SetCookie:    []http.Cookie{expiredRefreshCookie()},
		CacheControl: "no-store",
		Body: humautil.APIResponse[humautil.MessageBody]{
			Status: "success",
			Data:   humautil.MessageBody{Message: "logged out successfully"},
		},
	}, nil
}
