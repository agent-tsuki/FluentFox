package auth

import (
	"net/url"
	"strings"
	"testing"
)

func TestXOAuthStateKeyDoesNotExposeRequestToken(t *testing.T) {
	requestToken := "temporary-request-token"
	key := xOAuthStateKey(requestToken)
	if strings.Contains(key, requestToken) {
		t.Fatalf("state key %q exposes request token", key)
	}
	if !strings.HasPrefix(key, "oauth:x:state:") {
		t.Fatalf("state key %q has unexpected prefix", key)
	}
}

func TestFrontendXLinkURLStaysOnSettings(t *testing.T) {
	service := &XOAuthService{frontendURL: "https://www.foxsensei.com"}
	parsed, err := url.Parse(service.frontendXLinkURL(""))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/profile" || parsed.Query().Get("tab") != "settings" || parsed.Query().Get("x") != "linked" {
		t.Fatalf("link URL = %q, want Profile settings success URL", parsed.String())
	}

	callback, err := url.Parse(service.frontendErrorURL("oauth_x_failed"))
	if err != nil {
		t.Fatal(err)
	}
	if callback.Path != "/auth/callback" || callback.Fragment != "error=oauth_x_failed&provider=x" {
		t.Fatalf("error callback = %q, want X callback fragment", callback.String())
	}
}
