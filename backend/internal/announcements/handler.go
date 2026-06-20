package announcements

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/middleware"
)

type Handler struct {
	svc *Service
	cfg config.Config
}

func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool, cfg config.Config) {
	repo := NewRepository(db.New(pool))
	svc := NewService(repo, pool)
	h := &Handler{svc: svc, cfg: cfg}

	lecturer := r.Group("/api/lecturer", middleware.RequireRole(db.UserRoleLecturer))
	{
		lecturer.POST("/courses/:id/announcements", h.CreateAnnouncement)
		lecturer.GET("/courses/:id/announcements", h.ListForCourse)
	}

	student := r.Group("/api/student", middleware.RequireRole(db.UserRoleStudent))
	{
		student.GET("/courses/:id/announcements", h.ListForStudent)
		student.GET("/announcements/:id", h.GetAnnouncement)
	}
}

func (h *Handler) CreateAnnouncement(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_course_id", "invalid course id format"))
		return
	}

	var req CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request", err.Error()))
		return
	}

	authorID := c.GetInt64("user_id")
	ann, err := h.svc.CreateAnnouncement(c.Request.Context(), courseID, authorID, req)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "not a lecturer of this course"))
			return
		}
		if errors.Is(err, ErrValidation) || errors.Is(err, ErrNotEnrolled) {
			c.JSON(http.StatusUnprocessableEntity, errorEnvelope("validation_error", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, AnnouncementResponse{
		ID:           ann.ID,
		CourseID:     ann.CourseID,
		AuthorID:     ann.AuthorID,
		Title:        ann.Title,
		Body:         ann.Body,
		AudienceType: ann.AudienceType,
		CreatedAt:    ann.CreatedAt.Time,
	})
}

func (h *Handler) ListForCourse(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_course_id", "invalid course id format"))
		return
	}
	lecturerID := c.GetInt64("user_id")

	anns, err := h.svc.ListForCourse(c.Request.Context(), courseID, lecturerID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "forbidden"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	res := make([]AnnouncementResponse, 0, len(anns))
	for _, a := range anns {
		res = append(res, AnnouncementResponse{
			ID:           a.ID,
			CourseID:     a.CourseID,
			AuthorID:     a.AuthorID,
			Title:        a.Title,
			Body:         a.Body,
			AudienceType: a.AudienceType,
			CreatedAt:    a.CreatedAt.Time,
		})
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListForStudent(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_course_id", "invalid course id format"))
		return
	}
	studentID := c.GetInt64("user_id")

	anns, err := h.svc.ListForStudent(c.Request.Context(), courseID, studentID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "forbidden"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	res := make([]AnnouncementResponse, 0, len(anns))
	for _, a := range anns {
		res = append(res, AnnouncementResponse{
			ID:           a.ID,
			CourseID:     a.CourseID,
			AuthorID:     a.AuthorID,
			Title:        a.Title,
			Body:         a.Body,
			AudienceType: a.AudienceType,
			CreatedAt:    a.CreatedAt.Time,
		})
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetAnnouncement(c *gin.Context) {
	announcementID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_announcement_id", "invalid format"))
		return
	}
	
	courseIDStr := c.Query("course_id")
	courseID, err := strconv.ParseInt(courseIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_course_id", "course_id query param is required"))
		return
	}

	userID := c.GetInt64("user_id")
	role := db.UserRole(c.GetString("role"))

	ann, err := h.svc.GetByID(c.Request.Context(), courseID, announcementID, userID, role)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "forbidden"))
			return
		}
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("not_found", "not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	c.JSON(http.StatusOK, AnnouncementResponse{
		ID:           ann.ID,
		CourseID:     ann.CourseID,
		AuthorID:     ann.AuthorID,
		Title:        ann.Title,
		Body:         ann.Body,
		AudienceType: ann.AudienceType,
		CreatedAt:    ann.CreatedAt.Time,
	})
}
