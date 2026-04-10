package users

import "time"

type VerificationModel struct {
	HashCode string
	ExpiresAt time.Time
    VerifiedAt *time.Time

}
