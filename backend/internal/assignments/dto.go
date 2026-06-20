package assignments

import (
	"time"
)

type CreateAssignmentRequest struct {
	Title             string    `json:"title" binding:"required"`
	Description       string    `json:"description"`
	Deadline          time.Time `json:"deadline" binding:"required"`
	AcceptLate        bool      `json:"accept_late"`
	LateThresholdDays *int32    `json:"late_threshold_days"`
	MaxScore          float64   `json:"max_score" binding:"required,gt=0"`
}

type AssignmentResponse struct {
	ID                   int64      `json:"id"`
	Title                string     `json:"title"`
	Description          string     `json:"description"`
	Deadline             time.Time  `json:"deadline"`
	AcceptLate           bool       `json:"accept_late"`
	LateThresholdDays    *int32     `json:"late_threshold_days"`
	MaxScore             float64    `json:"max_score"`
	GradingFinalizedAt   *time.Time `json:"grading_finalized_at"`
	CreatedAt            time.Time  `json:"created_at"`
}

type SubmissionResponse struct {
	ID               int64       `json:"id"`
	Version          int32       `json:"version"`
	OriginalFilename string      `json:"original_filename"`
	IsLate           bool        `json:"is_late"`
	SubmittedAt      time.Time   `json:"submitted_at"`
	LateDuration     string      `json:"late_duration"`
	Score            *float64    `json:"score"`
	Feedback         *string     `json:"feedback"`
}

func errorEnvelope(code, message string) map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{"code": code, "message": message},
	}
}
