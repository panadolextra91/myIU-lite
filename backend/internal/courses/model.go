package courses

import "errors"

var (
	ErrCourseNotFound    = errors.New("course not found or deleted")
	ErrInvalidDates      = errors.New("end date must be on or after start date")
	ErrRequiredFields    = errors.New("code, name, and term are required")
	ErrInvalidDateFormat = errors.New("invalid date format, expected YYYY-MM-DD or DD/MM/YYYY")
)
