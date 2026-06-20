package quizzes

import "errors"

var (
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not_found")
	ErrInvalidDates    = errors.New("close date must be after open date")
	ErrPoolTooSmall    = errors.New("max_questions cannot exceed pool_size")
	ErrInvalidQuestion = errors.New("invalid question format")
)

type QuizOption struct {
	ID         int64
	QuestionID int64
	Text       string
	IsCorrect  bool
}

type RowError struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ImportError struct {
	RowErrors []RowError
}

func (e *ImportError) Error() string {
	return "import validation failed"
}
