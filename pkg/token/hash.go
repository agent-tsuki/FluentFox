// Package token — hash helper for opaque tokens.
// Raw refresh tokens, email verification tokens, and password reset tokens
// must NEVER be stored in the database. Always hash them first with HashToken.
package token

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken returns a hex-encoded SHA-256 digest of the input token.
// The same token always produces the same hash, making lookups possible
// without ever persisting the raw value.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
