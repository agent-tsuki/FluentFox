package exceptions

import "errors"

var (
	ErrEmailAlreadyExists  = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrAccountInactive     = errors.New("account is inactive")
	ErrUserNotFound        = errors.New("user not found")
)