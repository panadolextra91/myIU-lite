package auth

import "errors"

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrCurrentPasswordWrong = errors.New("current password wrong")
	ErrTooShort             = errors.New("password too short")
	ErrSameAsCurrent        = errors.New("same as current password")
	ErrConfirmMismatch      = errors.New("confirm mismatch")
)
