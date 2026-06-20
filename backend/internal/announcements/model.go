package announcements

import "errors"

var (
	ErrNotFound      = errors.New("announcement not found")
	ErrForbidden     = errors.New("forbidden")
	ErrValidation    = errors.New("validation failed")
	ErrNotEnrolled   = errors.New("one or more target students are not enrolled")
)
