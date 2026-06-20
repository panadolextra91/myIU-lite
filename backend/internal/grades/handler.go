package grades

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
	svc := NewService(pool, repo)
	h := &Handler{svc: svc, cfg: cfg}

	student := r.Group("/api/student")
	student.Use(middleware.RequireRole(db.UserRoleStudent))
	student.GET("/courses/:id/grades", h.GetStudentGrades)

	lecturer := r.Group("/api/lecturer", middleware.RequireRole(db.UserRoleLecturer))
	{
		lecturer.POST("/courses/:id/grade-scheme", h.handleCreateScheme)
		lecturer.GET("/courses/:id/grade-scheme", h.handleGetScheme)
		lecturer.DELETE("/courses/:id/grade-scheme", h.handleDeleteScheme)
		lecturer.PUT("/courses/:id/grade-components/:componentId/scores", h.handleEnterScore)
		lecturer.POST("/courses/:id/grade-components/:componentId/scores/import", h.handleImportScores)
		lecturer.GET("/courses/:id/grades", h.handleGetGrades)
		lecturer.POST("/courses/:id/grade-components/:componentId/publish", h.PublishComponent)
	}
}

func (h *Handler) handleCreateScheme(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_course_id", "invalid course id format"))
		return
	}

	var req SchemeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request", err.Error()))
		return
	}

	lecturerID := c.GetInt64("user_id")
	resp, err := h.svc.CreateScheme(c.Request.Context(), courseID, lecturerID, req.Components)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "not a lecturer of this course"))
		} else if errors.Is(err, ErrSchemeExists) {
			c.JSON(http.StatusConflict, errorEnvelope("scheme_exists", err.Error()))
		} else if errors.Is(err, ErrValidation) {
			c.JSON(http.StatusUnprocessableEntity, errorEnvelope("validation_failed", err.Error()))
		} else {
			c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", err.Error()))
		}
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) handleGetScheme(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_course_id", "invalid course id format"))
		return
	}

	userID := c.GetInt64("user_id")
	role := c.GetString("user_role")

	resp, err := h.svc.GetScheme(c.Request.Context(), courseID, userID, role)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "forbidden"))
		} else if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("not_found", "grade scheme not found"))
		} else {
			c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) handleDeleteScheme(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_course_id", "invalid course id format"))
		return
	}

	lecturerID := c.GetInt64("user_id")
	err = h.svc.DeleteScheme(c.Request.Context(), courseID, lecturerID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "forbidden"))
		} else if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("not_found", "grade scheme not found"))
		} else if errors.Is(err, ErrSchemeImmutable) {
			c.JSON(http.StatusConflict, errorEnvelope("immutable", "scheme is immutable (scores or publications exist)"))
		} else {
			c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", err.Error()))
		}
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) handleEnterScore(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_course_id", "invalid course id format"))
		return
	}

	componentID, err := strconv.ParseInt(c.Param("componentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_component_id", "invalid component id format"))
		return
	}

	var req ScoreEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request", err.Error()))
		return
	}

	lecturerID := c.GetInt64("user_id")
	err = h.svc.EnterScore(c.Request.Context(), courseID, componentID, req.StudentID, req.Score, lecturerID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "forbidden"))
		} else if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("not_found", "component not found"))
		} else if errors.Is(err, ErrValidation) {
			c.JSON(http.StatusUnprocessableEntity, errorEnvelope("validation_failed", err.Error()))
		} else {
			c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", err.Error()))
		}
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) handleGetGrades(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_course_id", "invalid course id format"))
		return
	}
	
	studentIDStr := c.Query("student_id")
	if studentIDStr == "" {
		repo := db.New(h.svc.pool)
		students, err := repo.ListCourseStudents(c.Request.Context(), courseID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", err.Error()))
			return
		}
		
		var results []OverallResponse
		for _, st := range students {
			resp, err := h.svc.ComputeOverallForStudent(c.Request.Context(), courseID, st.StudentID)
			if err != nil {
				if !errors.Is(err, ErrNotFound) {
					c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", err.Error()))
					return
				}
				c.JSON(http.StatusNotFound, errorEnvelope("not_found", "no grade scheme"))
				return
			}
			results = append(results, resp)
		}
		c.JSON(http.StatusOK, results)
		return
	}

	studentID, err := strconv.ParseInt(studentIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_student_id", "invalid student id format"))
		return
	}

	resp, err := h.svc.ComputeOverallForStudent(c.Request.Context(), courseID, studentID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("not_found", "grade scheme not found"))
		} else {
			c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) handleImportScores(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_course_id", "invalid course id format"))
		return
	}

	componentID, err := strconv.ParseInt(c.Param("componentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_component_id", "invalid component id format"))
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20) // 5 MB
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request", "file is required"))
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType != "text/csv" && contentType != "application/vnd.ms-excel" && contentType != "text/plain" {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_file", "file must be a CSV"))
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", "failed to open file"))
		return
	}
	defer f.Close()

	lecturerID := c.GetInt64("user_id")
	rowErrs, err := h.svc.ImportScoresCSV(c.Request.Context(), courseID, componentID, lecturerID, f)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "forbidden"))
		} else if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("not_found", "component not found"))
		} else if errors.Is(err, ErrValidation) {
			if len(rowErrs) > 0 {
				c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": rowErrs})
			} else {
				c.JSON(http.StatusUnprocessableEntity, errorEnvelope("validation_failed", err.Error()))
			}
		} else {
			c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", err.Error()))
		}
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) PublishComponent(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request", "invalid course id"))
		return
	}
	componentID, err := strconv.ParseInt(c.Param("componentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request", "invalid component id"))
		return
	}
	lecturerID := c.GetInt64("user_id")

	err = h.svc.PublishComponent(c.Request.Context(), courseID, componentID, lecturerID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", err.Error()))
			return
		}
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("not_found", err.Error()))
			return
		}
		if errors.Is(err, ErrValidation) {
			c.JSON(http.StatusUnprocessableEntity, errorEnvelope("validation_error", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) GetStudentGrades(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request", "invalid course id"))
		return
	}
	studentID := c.GetInt64("user_id")

	resp, err := h.svc.GetStudentGrades(c.Request.Context(), courseID, studentID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}
