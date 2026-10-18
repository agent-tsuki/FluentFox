package humautil

import (
	"errors"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/fluentfox/api/pkg/exceptions"
	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/token"
)

const BEARER_SCHEME = "bearer"

const (
	METADATA_PUBLIC           = "public"
	METADATA_ALLOW_UNVERIFIED = "allowUnverified"
)

const bearerPrefix = "Bearer "

func Public() map[string]any {
	return map[string]any{METADATA_PUBLIC: true}
}

func AllowUnverified() map[string]any {
	return map[string]any{METADATA_ALLOW_UNVERIFIED: true}
}

func BearerSecurity() []map[string][]string {
	return []map[string][]string{{BEARER_SCHEME: {}}}
}

func RegisterBearerScheme(api huma.API) {
	components := api.OpenAPI().Components
	if components.SecuritySchemes == nil {
		components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	components.SecuritySchemes[BEARER_SCHEME] = &huma.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	}
}

// Every operation is protected unless marked Public(), so a route that forgets
// the flag fails closed. Must be added before any route is registered.
func AuthMiddleware(api huma.API, maker *token.Maker, allowedOrigins ...string) func(ctx huma.Context, next func(huma.Context)) {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}

	return func(ctx huma.Context, next func(huma.Context)) {
		op := ctx.Operation()
		if isCookieSessionOperation(op) && !isTrustedCookieRequest(ctx, allowed) {
			writeAppError(api, ctx, exceptions.ErrForbidden)
			return
		}
		if op != nil && hasFlag(op, METADATA_PUBLIC) {
			next(ctx)
			return
		}

		raw, ok := bearerToken(ctx.Header("Authorization"))
		if op == nil || !ok {
			writeAppError(api, ctx, exceptions.ErrTokenInvalid)
			return
		}
		claims, err := maker.ValidateAccessToken(raw)
		if err != nil {
			// Distinct codes let the client refresh on expiry but log in again otherwise.
			if errors.Is(err, token.ErrTokenExpired) {
				writeAppError(api, ctx, exceptions.ErrTokenExpired)
			} else {
				writeAppError(api, ctx, exceptions.ErrTokenInvalid)
			}
			return
		}

		// The claim can lag behind the database for up to one access-token
		// lifetime; clients call /auth/refresh after verifying to clear it.
		if !claims.EmailVerified && !hasFlag(op, METADATA_ALLOW_UNVERIFIED) {
			writeAppError(api, ctx, exceptions.ErrEmailNotVerified)
			return
		}

		next(huma.WithContext(ctx, middleware.ContextWithClaims(ctx.Context(), claims)))
	}
}

// Refresh and logout are authenticated by an HttpOnly cookie rather than a
// bearer token. A required custom header forces CORS preflight, and the Origin
// must exactly match configured first-party origins. This stops cross-site form
// posts and credentialed cross-site fetches from using the cookie as CSRF.
func isCookieSessionOperation(op *huma.Operation) bool {
	return op != nil && op.Method == "POST" && (op.Path == "/auth/refresh" || op.Path == "/auth/logout")
}

func isTrustedCookieRequest(ctx huma.Context, allowed map[string]struct{}) bool {
	origin := ctx.Header("Origin")
	if _, ok := allowed[origin]; !ok {
		return false
	}
	return strings.TrimSpace(ctx.Header("X-CSRF-Token")) != ""
}

func writeAppError(api huma.API, ctx huma.Context, appErr *exceptions.AppError) {
	writeAPIError(api, ctx, newAPIError(appErr.Status, appErr.Code, appErr.Message))
}

func hasFlag(op *huma.Operation, key string) bool {
	flag, ok := op.Metadata[key].(bool)
	return ok && flag
}

func bearerToken(header string) (string, bool) {
	if !strings.HasPrefix(header, bearerPrefix) {
		return "", false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
	return raw, raw != ""
}

// writeAPIError writes our error envelope directly. huma.WriteErr would rebuild
// the error from the status alone and lose codes like EMAIL_NOT_VERIFIED.
func writeAPIError(api huma.API, ctx huma.Context, apiErr *APIError) {
	ctx.SetHeader("Content-Type", "application/json")
	ctx.SetStatus(apiErr.GetStatus())
	if err := api.Marshal(ctx.BodyWriter(), "application/json", apiErr); err != nil {
		// Headers are already sent, so the status code is all the client gets.
		return
	}
}
