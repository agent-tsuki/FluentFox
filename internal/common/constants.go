package common

import "time"

const USERNAME_RETRY int = 3
const CHARSET string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const USERNAME_SUFFIX_LEN int = 7
const AUTH_TOKEN_LEN int = 32
const AUTH_TOKEN_EXPIRE_HOUR int = 6

const (
	PASSWORD_RESET_CODE_LENGTH        int           = 8
	PASSWORD_RESET_CODE_EXPIRES_AFTER time.Duration = 10 * time.Minute
	PASSWORD_RESET_LINK_EXPIRES_AFTER time.Duration = time.Hour
	PASSWORD_RESET_CODE_COOLDOWN      time.Duration = 60 * time.Second
	PASSWORD_RESET_SEND_WINDOW        time.Duration = 24 * time.Hour
	PASSWORD_RESET_DAILY_SEND_LIMIT   int64         = 5
	PASSWORD_RESET_MAX_CODE_ATTEMPTS  int           = 5
)

// Together these must stay well under the server's 15s WriteTimeout, because
// the send runs inside the request.
const (
	EMAIL_SEND_TIMEOUT      time.Duration = 8 * time.Second
	EMAIL_MX_LOOKUP_TIMEOUT time.Duration = 1 * time.Second
)

// The signup email counts toward the daily limit.
const (
	VERIFICATION_RESEND_COOLDOWN  time.Duration = 60 * time.Second
	VERIFICATION_SEND_WINDOW      time.Duration = 24 * time.Hour
	VERIFICATION_DAILY_SEND_LIMIT int64         = 5
)

// The index names must match the gorm tags on users.User.
const (
	PG_UNIQUE_VIOLATION         string = "23505"
	USERS_EMAIL_UNIQUE_INDEX    string = "idx_users_email"
	USERS_USERNAME_UNIQUE_INDEX string = "idx_users_username"
)

// Each argon2id hash uses 64 MB; this caps total hashing memory at 256 MB.
const (
	ARGON2_MAX_CONCURRENT_HASHES int = 4
	ARGON2_HASH_PARTS            int = 6
)

// HS256 can be brute-forced offline from any issued token if the secret is short.
const JWT_SECRET_MIN_BYTES int = 32
