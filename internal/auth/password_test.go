package auth

import (
	"context"
	"errors"
	"testing"
)

func TestVerifyPassword(t *testing.T) {
	defaultCfg := Argon2Config{}
	defaultCfg = defaultCfg.defaultConfig()
	defaultHash, err := defaultCfg.hashStringWithSalt("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	custom := Argon2Config{Memory: 32 * 1024, Iterations: 2, Parallelism: 1, SaltLength: 16, KeyLength: 32}
	customHash, err := custom.hashStringWithSalt("correct-password")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		plain   string
		encoded string
		want    bool
	}{
		{"returns true for the correct password", "correct-password", defaultHash, true},
		{"returns false for a wrong password", "wrong-password", defaultHash, false},
		// Proves the params stored in the hash are used, not today's defaults.
		{"verifies a hash made with non-default params", "correct-password", customHash, true},
		{"rejects a wrong password on a non-default-params hash", "wrong-password", customHash, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := verifyPassword(context.Background(), tc.plain, tc.encoded)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDecodePasswordHash_malformed(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
	}{
		{"returns errMalformedPasswordHash for an empty string", ""},
		{"returns errMalformedPasswordHash when part count is wrong", "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA"},
		{"returns errMalformedPasswordHash when the algorithm is not argon2id", "$argon2i$v=19$m=65536,t=3,p=4$c2FsdA$a2V5"},
		{"returns errMalformedPasswordHash when params are unparsable", "$argon2id$v=19$m=abc,t=3,p=4$c2FsdA$a2V5"},
		{"returns errMalformedPasswordHash when params are missing fields", "$argon2id$v=19$m=65536,t=3$c2FsdA$a2V5"},
		{"returns errMalformedPasswordHash when salt is not base64", "$argon2id$v=19$m=65536,t=3,p=4$!!!$a2V5"},
		{"returns errMalformedPasswordHash when key is not base64", "$argon2id$v=19$m=65536,t=3,p=4$c2FsdA$!!!"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, err := decodePasswordHash(tc.encoded)
			if !errors.Is(err, errMalformedPasswordHash) {
				t.Fatalf("got %v, want errMalformedPasswordHash", err)
			}
			if _, err := verifyPassword(context.Background(), "x", tc.encoded); !errors.Is(err, errMalformedPasswordHash) {
				t.Fatalf("verifyPassword got %v, want errMalformedPasswordHash", err)
			}
		})
	}
}

func TestDummyPasswordHash_decodes(t *testing.T) {
	if _, _, _, err := decodePasswordHash(dummyPasswordHash); err != nil {
		t.Fatalf("dummy hash must decode: %v", err)
	}
}

func TestAcquireArgonSlot(t *testing.T) {
	t.Run("returns context.Canceled without blocking when all slots are taken", func(t *testing.T) {
		filled := 0
		for len(argonSlots) < cap(argonSlots) {
			argonSlots <- struct{}{}
			filled++
		}
		t.Cleanup(func() {
			for ; filled > 0; filled-- {
				<-argonSlots
			}
		})

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		done := make(chan error, 1)
		go func() {
			_, err := acquireArgonSlot(ctx)
			done <- err
		}()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v, want context.Canceled", err)
		}
	})

	t.Run("returns a release func that frees the slot", func(t *testing.T) {
		before := len(argonSlots)
		release, err := acquireArgonSlot(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(argonSlots) != before+1 {
			t.Fatalf("slot not taken")
		}
		release()
		if len(argonSlots) != before {
			t.Fatalf("slot not released")
		}
	})
}
