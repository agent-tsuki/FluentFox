package humautil

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"

	"github.com/fluentfox/api/pkg/middleware"
	"github.com/fluentfox/api/pkg/token"
)

type pingOutput struct {
	Body struct {
		UserID string `json:"user_id"`
	}
}

func TestAuthMiddleware(t *testing.T) {
	InitHumaErrors()
	maker := token.NewMaker("access-secret", 15, 7)
	expiredMaker := token.NewMaker("access-secret", -5, 7)
	otherMaker := token.NewMaker("other-secret", 15, 7)
	userID := uuid.New()

	mint := func(t *testing.T, m *token.Maker, verified bool) string {
		t.Helper()
		raw, err := m.GenerateAccessToken(userID, false, verified)
		if err != nil {
			t.Fatalf("mint token: %v", err)
		}
		return raw
	}

	_, api := humatest.New(t)
	api.UseMiddleware(AuthMiddleware(api, maker))
	handler := func(ctx context.Context, _ *struct{}) (*pingOutput, error) {
		out := &pingOutput{}
		out.Body.UserID = middleware.ContextUserID(ctx).String()
		return out, nil
	}
	huma.Register(api, huma.Operation{OperationID: "public", Method: http.MethodGet, Path: "/public", Metadata: Public()}, handler)
	huma.Register(api, huma.Operation{OperationID: "protected", Method: http.MethodGet, Path: "/protected"}, handler)
	huma.Register(api, huma.Operation{OperationID: "lenient", Method: http.MethodGet, Path: "/lenient", Metadata: AllowUnverified()}, handler)

	tests := []struct {
		name       string
		path       string
		header     string
		wantStatus int
		wantCode   string
		wantUserID bool
	}{
		{name: "public route allows a request without a header", path: "/public", wantStatus: http.StatusOK},
		{name: "protected route returns 401 TOKEN_INVALID without a header", path: "/protected", wantStatus: http.StatusUnauthorized, wantCode: "TOKEN_INVALID"},
		{name: "protected route returns 401 for a non-bearer header", path: "/protected", header: "Basic abc", wantStatus: http.StatusUnauthorized, wantCode: "TOKEN_INVALID"},
		{name: "protected route returns 401 for an empty bearer token", path: "/protected", header: "Bearer ", wantStatus: http.StatusUnauthorized, wantCode: "TOKEN_INVALID"},
		{name: "protected route returns 401 for a garbage token", path: "/protected", header: "Bearer not-a-jwt", wantStatus: http.StatusUnauthorized, wantCode: "TOKEN_INVALID"},
		{name: "protected route returns 401 for a token signed with another secret", path: "/protected", header: "Bearer " + mint(t, otherMaker, true), wantStatus: http.StatusUnauthorized, wantCode: "TOKEN_INVALID"},
		{name: "protected route returns 401 TOKEN_EXPIRED for an expired token", path: "/protected", header: "Bearer " + mint(t, expiredMaker, true), wantStatus: http.StatusUnauthorized, wantCode: "TOKEN_EXPIRED"},
		{name: "protected route returns 403 EMAIL_NOT_VERIFIED for an unverified user", path: "/protected", header: "Bearer " + mint(t, maker, false), wantStatus: http.StatusForbidden, wantCode: "EMAIL_NOT_VERIFIED"},
		{name: "protected route allows a verified user", path: "/protected", header: "Bearer " + mint(t, maker, true), wantStatus: http.StatusOK, wantUserID: true},
		{name: "allow-unverified route lets an unverified user in with their id in context", path: "/lenient", header: "Bearer " + mint(t, maker, false), wantStatus: http.StatusOK, wantUserID: true},
		{name: "allow-unverified route still requires a token", path: "/lenient", wantStatus: http.StatusUnauthorized, wantCode: "TOKEN_INVALID"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var args []any
			if tc.header != "" {
				args = append(args, "Authorization: "+tc.header)
			}

			resp := api.Get(tc.path, args...)

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body %s", resp.Code, tc.wantStatus, resp.Body.String())
			}
			if tc.wantCode != "" && !strings.Contains(resp.Body.String(), `"error_code":"`+tc.wantCode+`"`) {
				t.Errorf("body %s lacks error_code %s", resp.Body.String(), tc.wantCode)
			}
			if tc.wantUserID && !strings.Contains(resp.Body.String(), userID.String()) {
				t.Errorf("body %s lacks user id %s", resp.Body.String(), userID)
			}
		})
	}
}
