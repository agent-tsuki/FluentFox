package auth

import (
	"encoding/json"
	"testing"
)

// The handoff is stored as JSON in Redis; LoginResponse hides the refresh
// token from JSON, so this guards against it being dropped on the way.
func TestOAuthHandoff_KeepsRefreshTokenThroughJSON(t *testing.T) {
	tests := []struct {
		name  string
		login LoginResponse
	}{
		{name: "refresh token survives the store round trip", login: LoginResponse{AccessToken: "access", RefreshToken: "refresh"}},
		{name: "an empty refresh token stays empty", login: LoginResponse{AccessToken: "access"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(newOAuthHandoff(&tt.login, "a@b.com", "/dashboard")) //nolint:gosec // test data, not a real token
			if err != nil {
				t.Fatal(err)
			}
			var restored oauthHandoff
			if err := json.Unmarshal(raw, &restored); err != nil {
				t.Fatal(err)
			}
			got := restored.login()
			if got.RefreshToken != tt.login.RefreshToken || got.AccessToken != tt.login.AccessToken {
				t.Errorf("restored login = %+v, want %+v", got, tt.login)
			}
		})
	}
}
