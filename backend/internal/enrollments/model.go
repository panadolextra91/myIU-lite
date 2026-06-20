package enrollments

import "errors"

var (
	ErrCourseNotFound = errors.New("course not found or deleted")
	ErrNoValidRows    = errors.New("no valid rows to import")
	ErrValidation     = errors.New("validation failed")
	ErrNotEnrolled    = errors.New("user not enrolled or assigned")
)

type RowError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}
