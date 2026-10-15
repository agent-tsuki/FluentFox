package users

import (
	"time"

	"github.com/google/uuid"
)

type VerificationModel struct {
	UserID   uuid.UUID
	HashCode string
	ExpiresAt time.Time
    VerifiedAt *time.Time

}
