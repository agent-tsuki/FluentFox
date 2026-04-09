package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"

	"golang.org/x/crypto/argon2"

	"github.com/fluentfox/api/internal/common"
)


type Argon2Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

func (argon *Argon2Config) defaultConfig() Argon2Config {
	return Argon2Config{
		Memory:      64 * 1024, // 64 MB
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
	}
}

func (argon *Argon2Config) generateSalt(length uint32) ([]byte, error) {
	salt := make([]byte, length)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, err
	}
	return salt, nil
}

func (argon *Argon2Config) hashedString(token string) (string, error) {
	// If Argon config give else we will use default config
	config := argon
	if argon == nil{
		defaultConfig := argon.defaultConfig()
		config = &defaultConfig

	}

	// Generating random salt
	salt, err := argon.generateSalt(config.SaltLength)
	if err != nil {
		return "", err
	}

	// Hashing the token
	hash := argon2.IDKey(
		[]byte(token),
		salt,
		config.Iterations,
		config.Memory,
		config.Parallelism,
		config.KeyLength,
	)

	// Encode to base64 as we are getting hash bytes
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Forming for hashed token  
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		config.Memory,
		config.Iterations,
		config.Parallelism,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

// Generate username with first and last name
func generateString(length int) (string, error) {
	result := make([]byte, length)
	for i := range result {
        n, err := rand.Int(rand.Reader, big.NewInt(int64(len(common.CHARSET))))
        if err != nil {
            return "", err
        }
        result[i] = common.CHARSET[n.Int64()]
    }
	return string(result), nil
}