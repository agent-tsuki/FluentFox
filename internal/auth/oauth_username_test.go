package auth

import "testing"

// Guards the slice in oauthUsername: a suffix shorter than its length panics
// on every new Google or X sign-up.
func TestOAuthUsernameSuffixIsLongEnough(t *testing.T) {
	for range 100 {
		raw, err := oauthRandom(oauthUsernameSuffixBytes)
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) < oauthUsernameSuffixLen {
			t.Fatalf("suffix %q has %d characters, need %d", raw, len(raw), oauthUsernameSuffixLen)
		}
	}
}
