package grades

import (
	"errors"
)

var (
	ErrValidation      = errors.New("validation failed")
	ErrSchemeExists    = errors.New("grade scheme already exists for this course")
	ErrSchemeImmutable = errors.New("grade scheme is immutable")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
)

type RowError struct {
	Row     int    `json:"row"`
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}
