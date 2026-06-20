package announcements

import "time"

type CreateAnnouncementRequest struct {
	Title        string  `json:"title" binding:"required,max=200"`
	Body         string  `json:"body" binding:"required,max=5000"`
	AudienceType string  `json:"audience_type" binding:"required,oneof=ALL_STUDENTS SPECIFIC_STUDENTS"`
	StudentIDs   []int64 `json:"student_ids"` // Required if SPECIFIC_STUDENTS
}

type AnnouncementResponse struct {
	ID           int64     `json:"id"`
	CourseID     int64     `json:"course_id"`
	AuthorID     int64     `json:"author_id"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	AudienceType string    `json:"audience_type"`
	CreatedAt    time.Time `json:"created_at"`
}

func errorEnvelope(code, message string) map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{"code": code, "message": message},
	}
}
