package requests

import (
	"time"

	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

type CreateRequestRequest struct {
	Type               string `json:"type" binding:"required,oneof=LEAVE_EARLY ABSENCE CUSTOM"`
	Title              string `json:"title" binding:"required,max=200"`
	Body               string `json:"body" binding:"required,max=5000"`
	TargetedLecturerID int64  `json:"targeted_lecturer_id" binding:"required"`
}

type ReplyRequestRequest struct {
	Decision string `json:"decision" binding:"required,oneof=APPROVED DENIED"`
	Note     string `json:"note" binding:"max=5000"`
}

type RequestResponse struct {
	ID                 int64      `json:"id"`
	CourseID           int64      `json:"course_id"`
	StudentID          int64      `json:"student_id"`
	TargetedLecturerID int64      `json:"targeted_lecturer_id"`
	Type               string     `json:"type"`
	Title              string     `json:"title"`
	Body               string     `json:"body"`
	Status             string     `json:"status"`
	ReplyNote          *string    `json:"reply_note"`
	CreatedAt          time.Time  `json:"created_at"`
	RepliedAt          *time.Time `json:"replied_at"`
}

func mapToResponse(req db.Request) RequestResponse {
	res := RequestResponse{
		ID:                 req.ID,
		CourseID:           req.CourseID,
		StudentID:          req.StudentID,
		TargetedLecturerID: req.TargetedLecturerID,
		Type:               req.Type,
		Title:              req.Title,
		Body:               req.Body,
		Status:             req.Status,
		CreatedAt:          req.CreatedAt.Time,
	}
	if req.ReplyNote.Valid {
		res.ReplyNote = &req.ReplyNote.String
	}
	if req.RepliedAt.Valid {
		res.RepliedAt = &req.RepliedAt.Time
	}
	return res
}

func errorEnvelope(code, message string) map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	}
}
