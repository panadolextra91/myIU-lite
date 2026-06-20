package courses

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
}

func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool, cfg config.Config) {
	q := db.New(pool)
	repo := NewRepository(q)
	svc := NewService(pool, repo)
	h := &Handler{svc: svc}

	g := r.Group("/admin/courses")
	g.Use(middleware.AuthMiddleware(pool, cfg), middleware.RequireRole(db.UserRoleAdmin))
	{
		g.POST("", h.CreateCourse)
		g.GET("", h.ListCourses)
		g.GET("/:id", h.GetCourse)
		g.PUT("/:id", h.UpdateCourse)
		g.DELETE("/:id", h.DeleteCourse)
		g.GET("/:id/students", h.ListStudents)
		g.GET("/:id/lecturers", h.ListLecturers)
	}

	lec := r.Group("/api/lecturer/courses")
	lec.Use(middleware.AuthMiddleware(pool, cfg), middleware.RequireRole(db.UserRoleLecturer))
	{
		lec.GET("/:id/students", h.ListStudents)
	}
}

func mapToResponse(c db.Course) CourseResponse {
	return CourseResponse{
		ID:        c.ID,
		Code:      c.Code,
		Name:      c.Name,
		Term:      c.Term,
		StartDate: c.StartDate.Time.Format("2006-01-02"),
		EndDate:   c.EndDate.Time.Format("2006-01-02"),
		CreatedAt: c.CreatedAt.Time,
	}
}

func (h *Handler) CreateCourse(c *gin.Context) {
	var req CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_REQUEST", err.Error()))
		return
	}

	actorID := c.GetInt64("user_id")
	course, err := h.svc.CreateCourse(c.Request.Context(), req.Code, req.Name, req.Term, req.StartDate, req.EndDate, actorID)
	if err != nil {
		if errors.Is(err, ErrInvalidDates) || errors.Is(err, ErrInvalidDateFormat) || errors.Is(err, ErrRequiredFields) {
			c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_DATA", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to create course"))
		return
	}

	c.JSON(http.StatusCreated, mapToResponse(course))
}

func (h *Handler) ListCourses(c *gin.Context) {
	limit := int32(50)
	offset := int32(0)

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.ParseInt(l, 10, 32); err == nil && parsed > 0 {
			limit = int32(parsed)
			if limit > 200 {
				limit = 200
			}
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.ParseInt(o, 10, 32); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	var term *string
	if t := c.Query("term"); t != "" {
		term = &t
	}

	var search *string
	if s := c.Query("search"); s != "" {
		search = &s
	}

	courses, total, err := h.svc.ListCourses(c.Request.Context(), term, search, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to list courses"))
		return
	}

	res := make([]CourseResponse, len(courses))
	for i, c := range courses {
		res[i] = mapToResponse(c)
	}

	c.JSON(http.StatusOK, PaginatedCourses{
		Data:  res,
		Total: total,
	})
}

func (h *Handler) GetCourse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "Invalid course ID"))
		return
	}

	course, err := h.svc.GetCourse(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("NOT_FOUND", "Course not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to get course"))
		return
	}

	c.JSON(http.StatusOK, mapToResponse(course))
}

func (h *Handler) UpdateCourse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "Invalid course ID"))
		return
	}

	var req UpdateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_REQUEST", err.Error()))
		return
	}

	actorID := c.GetInt64("user_id")
	course, err := h.svc.UpdateCourse(c.Request.Context(), id, req.Code, req.Name, req.Term, req.StartDate, req.EndDate, actorID)
	if err != nil {
		if errors.Is(err, ErrInvalidDates) || errors.Is(err, ErrInvalidDateFormat) || errors.Is(err, ErrRequiredFields) {
			c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_DATA", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to update course"))
		return
	}

	c.JSON(http.StatusOK, mapToResponse(course))
}

func (h *Handler) DeleteCourse(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "Invalid course ID"))
		return
	}

	actorID := c.GetInt64("user_id")
	err = h.svc.SoftDeleteCourse(c.Request.Context(), id, actorID)
	if err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("NOT_FOUND", "Course not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to delete course"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) ListStudents(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "Invalid course ID"))
		return
	}

	students, err := h.svc.ListCourseStudents(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("NOT_FOUND", "Course not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to get students"))
		return
	}

	res := make([]RosterUser, len(students))
	for i, s := range students {
		res[i] = RosterUser{
			ID:       s.StudentID,
			Username: s.Username,
			FullName: s.FullName.String,
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (h *Handler) ListLecturers(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "Invalid course ID"))
		return
	}

	lecturers, err := h.svc.ListCourseLecturers(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCourseNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("NOT_FOUND", "Course not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to get lecturers"))
		return
	}

	res := make([]RosterUser, len(lecturers))
	for i, l := range lecturers {
		res[i] = RosterUser{
			ID:       l.LecturerID,
			Username: l.Username,
			FullName: l.FullName.String,
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": res})
}
