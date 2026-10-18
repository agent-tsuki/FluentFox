package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestRefreshCookieIsHostOnlyAndUnreadableToJavaScript(t *testing.T) {
	cookie := refreshCookie("opaque-refresh-value", 7*24*60*60)

	if cookie.Name != refreshCookieName {
		t.Fatalf("cookie name = %q, want %q", cookie.Name, refreshCookieName)
	}
	if !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie flags = Secure:%t HttpOnly:%t SameSite:%v, want Secure HttpOnly Lax", cookie.Secure, cookie.HttpOnly, cookie.SameSite)
	}
	if cookie.Path != "/" || cookie.Domain != "" {
		t.Fatalf("cookie scope = path %q domain %q, want path / and no domain", cookie.Path, cookie.Domain)
	}
	if !strings.HasPrefix(cookie.Name, "__Host-") {
		t.Fatalf("cookie name %q must use the __Host- prefix", cookie.Name)
	}
}

func TestLoginResponseNeverSerializesRefreshToken(t *testing.T) {
	const rawRefresh = "opaque-refresh-value"
	body, err := json.Marshal(LoginResponse{AccessToken: "access", RefreshToken: rawRefresh})
	if err != nil {
		t.Fatalf("marshal login response: %v", err)
	}
	if strings.Contains(string(body), rawRefresh) || strings.Contains(string(body), "refresh_token") {
		t.Fatalf("login JSON exposes refresh credential: %s", body)
	}
}
