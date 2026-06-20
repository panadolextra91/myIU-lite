package courses

import "time"

type CreateCourseRequest struct {
	Code      string `json:"code" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Term      string `json:"term" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

type UpdateCourseRequest struct {
	Code      string `json:"code" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Term      string `json:"term" binding:"required"`
	StartDate string `json:"start_date" binding:"required"`
	EndDate   string `json:"end_date" binding:"required"`
}

type CourseResponse struct {
	ID        int64     `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Term      string    `json:"term"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	CreatedAt time.Time `json:"created_at"`
}

type PaginatedCourses struct {
	Data  []CourseResponse `json:"data"`
	Total int64            `json:"total"`
}

type RosterUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
}

func errorEnvelope(code, message string) map[string]interface{} {
	return map[string]interface{}{
		"error": map[string]interface{}{"code": code, "message": message},
	}
}
