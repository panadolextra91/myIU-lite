package users

import "errors"

var (
	ErrInvalidDOBFormat = errors.New("invalid date of birth format, expected DD/MM/YYYY")
	ErrUserNotFound     = errors.New("user not found or inactive")
	ErrDuplicateUser    = errors.New("user already exists")
)

type RowError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ParsedAccount struct {
	ID       string
	FullName string
	DOB      string
	RowIndex int
}
