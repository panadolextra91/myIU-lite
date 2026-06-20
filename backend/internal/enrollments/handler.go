package enrollments

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

	g := r.Group("/admin/courses/:id")
	g.Use(middleware.AuthMiddleware(pool, cfg), middleware.RequireRole(db.UserRoleAdmin))
	{
		g.POST("/students/import", h.ImportStudents)
		g.POST("/lecturers/import", h.ImportLecturers)
		g.DELETE("/students/:studentId", h.RemoveStudent)
		g.DELETE("/lecturers/:lecturerId", h.UnassignLecturer)
	}
}

func (h *Handler) ImportStudents(c *gin.Context) {
	h.handleImport(c, "student")
}

func (h *Handler) ImportLecturers(c *gin.Context) {
	h.handleImport(c, "lecturer")
}

func (h *Handler) handleImport(c *gin.Context, role string) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "Invalid course ID"))
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20)

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_FILE", "A CSV file is required"))
		return
	}
	defer file.Close()

	actorID := c.GetInt64("user_id")

	added, rowErrs, err := h.svc.ImportMembers(c.Request.Context(), courseID, role, file, actorID)
	if err != nil {
		if errors.Is(err, ErrValidation) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": rowErrs})
			return
		}
		if errors.Is(err, ErrCourseNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("NOT_FOUND", "Course not found"))
			return
		}
		if errors.Is(err, ErrNoValidRows) {
			c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_FILE", "File is empty or contains no data rows"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to import members"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"imported": added})
}

func (h *Handler) RemoveStudent(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "Invalid course ID"))
		return
	}
	studentID, err := strconv.ParseInt(c.Param("studentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "Invalid student ID"))
		return
	}

	actorID := c.GetInt64("user_id")
	err = h.svc.RemoveStudent(c.Request.Context(), courseID, studentID, actorID)
	if err != nil {
		if errors.Is(err, ErrNotEnrolled) {
			c.JSON(http.StatusNotFound, errorEnvelope("NOT_FOUND", "Student is not enrolled in this course"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to remove student"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) UnassignLecturer(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "Invalid course ID"))
		return
	}
	lecturerID, err := strconv.ParseInt(c.Param("lecturerId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "Invalid lecturer ID"))
		return
	}

	actorID := c.GetInt64("user_id")
	err = h.svc.UnassignLecturer(c.Request.Context(), courseID, lecturerID, actorID)
	if err != nil {
		if errors.Is(err, ErrNotEnrolled) {
			c.JSON(http.StatusNotFound, errorEnvelope("NOT_FOUND", "Lecturer is not assigned to this course"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to unassign lecturer"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
