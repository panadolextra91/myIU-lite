package requests

import "errors"

var (
	ErrForbidden     = errors.New("forbidden")
	ErrNotFound      = errors.New("request not found")
	ErrValidation    = errors.New("validation failed")
	ErrNotTargeted   = errors.New("not the targeted lecturer")
	ErrAlreadyClosed = errors.New("request is already closed or not pending")
)
