package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"net/url"
	"testing"
)

func TestValidOAuthReturnTo(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{name: "uses dashboard by default", raw: "", want: "/dashboard", ok: true},
		{name: "accepts internal path and query", raw: "/dashboard?tab=review", want: "/dashboard?tab=review", ok: true},
		{name: "rejects scheme relative URL", raw: "//attacker.example", ok: false},
		{name: "rejects absolute URL", raw: "https://attacker.example", ok: false},
		{name: "rejects relative URL", raw: "dashboard", ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := validOAuthReturnTo(test.raw)
			if (err == nil) != test.ok {
				t.Fatalf("validOAuthReturnTo(%q) error = %v, want success %v", test.raw, err, test.ok)
			}
			if got != test.want {
				t.Fatalf("validOAuthReturnTo(%q) = %q, want %q", test.raw, got, test.want)
			}
		})
	}
}

func TestOAuthStoreCipherRoundTrip(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	store := &oauthStore{aead: aead}

	ciphertext, err := store.seal([]byte("state and tokens stay private"))
	if err != nil {
		t.Fatal(err)
	}
	if ciphertext == "state and tokens stay private" {
		t.Fatal("OAuth store wrote plaintext")
	}
	plaintext, err := store.open(ciphertext)
	if err != nil {
		t.Fatal(err)
	}
	if string(plaintext) != "state and tokens stay private" {
		t.Fatalf("decrypted value = %q", plaintext)
	}
}

func TestDecodeOAuthKey(t *testing.T) {
	key := make([]byte, 32)
	encoded := base64.StdEncoding.EncodeToString(key)
	got, err := decodeOAuthKey(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 32 {
		t.Fatalf("key length = %d, want 32", len(got))
	}
	if _, err := decodeOAuthKey(base64.StdEncoding.EncodeToString([]byte("too short"))); err == nil {
		t.Fatal("short key was accepted")
	}
}

func TestFrontendCallbackUsesFragment(t *testing.T) {
	service := &GoogleOAuthService{frontendURL: "https://www.foxsensei.com"}
	got := service.frontendCallbackURL(url.Values{"code": {"one-time-code"}})
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/auth/callback" || parsed.RawQuery != "" {
		t.Fatalf("callback URL = %s, want no query and /auth/callback", got)
	}
	if parsed.Fragment != "code=one-time-code" {
		t.Fatalf("callback fragment = %q", parsed.Fragment)
	}
}
