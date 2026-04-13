package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)


type Argon2Config struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

type HashManager struct {

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

func (argon *Argon2Config) hashStringWithSalt(token string) (string, error){
	// If Argon config give else we will use default config
	config := argon
	if argon == nil{
		defaultConfig := argon.defaultConfig()
		config = &defaultConfig

	}
	salt, err := argon.generateSalt(config.SaltLength)
	if err != nil {
		return "", err
	}

	hashedData := argon.hashedString(token, salt)

	return hashedData, err
}


func (argon *Argon2Config) hashedString(token string, salt []byte) string {
	// If Argon config give else we will use default config
	config := argon
	if argon == nil{
		defaultConfig := argon.defaultConfig()
		config = &defaultConfig

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

	return encodedHash
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

func GenerateHashedToken(length int) (string, error) {
	randomBytes := make([]byte, length)

	_, err := rand.Read(randomBytes)
	if err != nil{
		return "", err
	}

	// Encoding random byte to base64
	randomString := base64.URLEncoding.EncodeToString(randomBytes)
	
	return randomString, nil
}

func hashVerificationToken(token string) string {
    h := sha256.Sum256([]byte(token))
    return hex.EncodeToString(h[:])
}

func GetHashedSalt(token string) (string, error) {
	// split string '$'
	splitData := strings.Split(token, "$")
	
	if len(splitData) < 5{
		return "", exceptions.Wrap(http.StatusBadRequest, "BAD REQUEST", "Error while logging", nil)
	}

	return  splitData[4], nil
}

func DecodeHashSalt(saltStr string) ([]byte, error) {
	saltBytes, err := base64.RawStdEncoding.DecodeString(saltStr)
	if err != nil {
		// If it's not RawStd, try base64.StdEncoding (with padding)
		saltBytes, err = base64.StdEncoding.DecodeString(saltStr)
	}
	return  saltBytes, nil
}
