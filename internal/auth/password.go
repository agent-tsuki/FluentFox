package auth

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/fluentfox/api/internal/common"
)

var errMalformedPasswordHash = errors.New("malformed argon2id password hash")

// Each argon2 hash allocates 64 MB, so unbounded parallel logins could exhaust
// the server's memory. Shared by every service that hashes passwords.
var argonSlots = make(chan struct{}, common.ARGON2_MAX_CONCURRENT_HASHES)

// Unknown emails are checked against this hash so they take as long as real
// ones; otherwise response time would reveal which emails have accounts.
var dummyPasswordHash = func() string {
	argon := Argon2Config{}
	cfg := argon.defaultConfig()
	return cfg.hashedString("", make([]byte, cfg.SaltLength))
}()

type argonParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func acquireArgonSlot(ctx context.Context) (func(), error) {
	select {
	case argonSlots <- struct{}{}:
		return func() { <-argonSlots }, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("wait for password hashing slot: %w", ctx.Err())
	}
}

// Uses the parameters stored inside the hash, not today's defaults, so tuning
// the defaults later never locks out existing users.
func verifyPassword(ctx context.Context, plain, encoded string) (bool, error) {
	params, salt, key, err := decodePasswordHash(encoded)
	if err != nil {
		return false, err
	}

	release, err := acquireArgonSlot(ctx)
	if err != nil {
		return false, err
	}
	defer release()

	keyLen := uint32(len(key)) //nolint:gosec // argon2 keys are a few dozen bytes
	candidate := argon2.IDKey([]byte(plain), salt, params.iterations, params.memory, params.parallelism, keyLen)
	return subtle.ConstantTimeCompare(candidate, key) == 1, nil
}

// Format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<key>.
func decodePasswordHash(encoded string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != common.ARGON2_HASH_PARTS || parts[1] != "argon2id" {
		return argonParams{}, nil, nil, errMalformedPasswordHash
	}

	var p argonParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism); err != nil {
		return argonParams{}, nil, nil, fmt.Errorf("%w: parameters: %w", errMalformedPasswordHash, err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argonParams{}, nil, nil, fmt.Errorf("%w: salt: %w", errMalformedPasswordHash, err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argonParams{}, nil, nil, fmt.Errorf("%w: key: %w", errMalformedPasswordHash, err)
	}
	return p, salt, key, nil
}
