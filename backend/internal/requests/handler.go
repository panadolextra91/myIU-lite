package requests

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/authz"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/middleware"
)

type Handler struct {
	svc *Service
	pool *pgxpool.Pool
}

func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool, cfg config.Config) {
	repo := NewRepository(db.New(pool))
	svc := NewService(repo, pool)
	h := &Handler{svc: svc, pool: pool}

	student := r.Group("/api/student", middleware.RequireRole(db.UserRoleStudent))
	{
		student.POST("/courses/:id/requests", h.CreateRequest)
		student.GET("/requests", h.ListStudentRequests)
		student.GET("/courses/:id/lecturers", h.ListCourseLecturers)
		student.GET("/requests/:id", h.GetRequest)
	}

	lecturer := r.Group("/api/lecturer", middleware.RequireRole(db.UserRoleLecturer))
	{
		lecturer.GET("/requests", h.ListLecturerRequests)
		lecturer.POST("/requests/:id/reply", h.ReplyRequest)
		lecturer.GET("/requests/:id", h.GetRequest)
	}
}

func (h *Handler) CreateRequest(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_course_id", "invalid format"))
		return
	}

	var req CreateRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request", err.Error()))
		return
	}

	studentID := c.GetInt64("user_id")
	r, err := h.svc.CreateRequest(c.Request.Context(), courseID, studentID, req)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "forbidden"))
			return
		}
		if errors.Is(err, ErrValidation) {
			c.JSON(http.StatusUnprocessableEntity, errorEnvelope("validation_error", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, mapToResponse(r))
}

func (h *Handler) ReplyRequest(c *gin.Context) {
	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request_id", "invalid format"))
		return
	}

	var req ReplyRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request", err.Error()))
		return
	}

	lecturerID := c.GetInt64("user_id")
	r, err := h.svc.ReplyRequest(c.Request.Context(), requestID, lecturerID, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("not_found", "not found"))
			return
		}
		if errors.Is(err, ErrNotTargeted) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", err.Error()))
			return
		}
		if errors.Is(err, ErrAlreadyClosed) {
			c.JSON(http.StatusConflict, errorEnvelope("conflict", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	c.JSON(http.StatusOK, mapToResponse(r))
}

func (h *Handler) ListStudentRequests(c *gin.Context) {
	studentID := c.GetInt64("user_id")
	reqs, err := h.svc.ListForStudent(c.Request.Context(), studentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}
	res := make([]RequestResponse, 0, len(reqs))
	for _, r := range reqs {
		res = append(res, mapToResponse(r))
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ListLecturerRequests(c *gin.Context) {
	lecturerID := c.GetInt64("user_id")
	reqs, err := h.svc.ListForLecturer(c.Request.Context(), lecturerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}
	res := make([]RequestResponse, 0, len(reqs))
	for _, r := range reqs {
		res = append(res, mapToResponse(r))
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) GetRequest(c *gin.Context) {
	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid format"))
		return
	}

	userID := c.GetInt64("user_id")
	role := db.UserRole(c.GetString("role"))

	r, err := h.svc.GetByID(c.Request.Context(), requestID, userID, role)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("not_found", "not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	c.JSON(http.StatusOK, mapToResponse(r))
}

func (h *Handler) ListCourseLecturers(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid format"))
		return
	}
	
	studentID := c.GetInt64("user_id")
	if err := authz.AssertCourseMember(c.Request.Context(), h.pool, courseID, studentID, db.UserRoleStudent); err != nil {
		c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "forbidden"))
		return
	}

	q := db.New(h.pool)
	lecturers, err := q.ListCourseLecturers(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": lecturers})
}
