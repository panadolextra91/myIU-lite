package assignments

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/authz"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/cloudinary"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/middleware"
)

func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool, cfg config.Config, cld *cloudinary.Client) {
	repo := NewRepository(db.New(pool))
	service := NewService(pool, repo, cld)
	handler := &Handler{service: service}

	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware(pool, cfg))

	lecturer := api.Group("/lecturer")
	lecturer.Use(middleware.RequireRole(db.UserRoleLecturer))
	{
		lecturer.POST("/courses/:id/assignments", handler.CreateAssignment)
		lecturer.GET("/courses/:id/assignments", handler.ListAssignments)
		lecturer.GET("/courses/:id/assignments/:aid/submissions/:sid/download-url", handler.GetDownloadURL)
		lecturer.POST("/courses/:id/assignments/:aid/submissions/:sid/grade", handler.GradeSubmission)
	}

	student := api.Group("/student")
	student.Use(middleware.RequireRole(db.UserRoleStudent))
	{
		student.GET("/courses/:id/assignments", handler.ListAssignments)
		student.GET("/courses/:id/assignments/:aid/submissions", handler.ListSubmissions)
		student.POST("/courses/:id/assignments/:aid/submissions", handler.SubmitAssignment)
		student.GET("/courses/:id/assignments/:aid/submissions/:sid/download-url", handler.GetDownloadURL)
	}
}

type Handler struct {
	service *Service
}

func (h *Handler) CreateAssignment(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid course id"))
		return
	}

	var req CreateAssignmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request", err.Error()))
		return
	}

	lecturerID := c.GetInt64("user_id")
	assignment, err := h.service.CreateAssignment(c.Request.Context(), courseID, req, lecturerID)
	if err != nil {
		if err == ErrForbidden {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "not a lecturer of this course"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, mapAssignment(assignment))
}

func (h *Handler) ListAssignments(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid course id"))
		return
	}

	userID := c.GetInt64("user_id")
	role := c.GetString("role")

	assignments, err := h.service.ListCourseAssignments(c.Request.Context(), courseID, userID, role)
	if err != nil {
		if errors.Is(err, authz.ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "access denied"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	res := make([]AssignmentResponse, len(assignments))
	for i, a := range assignments {
		res[i] = mapAssignment(a)
	}

	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (h *Handler) ListSubmissions(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid course id"))
		return
	}
	assignmentID, err := strconv.ParseInt(c.Param("aid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid assignment id"))
		return
	}

	userID := c.GetInt64("user_id")

	subs, err := h.service.ListSubmissions(c.Request.Context(), courseID, assignmentID, userID)
	if err != nil {
		if errors.Is(err, authz.ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "access denied"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	res := make([]SubmissionResponse, len(subs))
	for i, s := range subs {
		res[i] = SubmissionResponse{
			ID:               s.ID,
			Version:          s.Version,
			OriginalFilename: s.OriginalFilename,
			IsLate:           s.IsLate,
			SubmittedAt:      s.SubmittedAt.Time,
		}
		if s.Score.Valid {
			score, _ := s.Score.Float64Value()
			if score.Valid {
				res[i].Score = &score.Float64
			}
		}
		if s.Feedback.Valid {
			res[i].Feedback = &s.Feedback.String
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (h *Handler) SubmitAssignment(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid course id"))
		return
	}

	assignmentID, err := strconv.ParseInt(c.Param("aid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid assignment id"))
		return
	}

	if c.Request.ContentLength > 10<<20 {
		c.JSON(http.StatusRequestEntityTooLarge, errorEnvelope("payload_too_large", "file size exceeds 10MB limit"))
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request", "invalid form file"))
		return
	}
	defer file.Close()

	// Sniff magic bytes
	headerBytes := make([]byte, 512)
	n, err := io.ReadAtLeast(file, headerBytes, 1)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		c.JSON(http.StatusInternalServerError, errorEnvelope("read_error", "failed to read file header"))
		return
	}
	headerBytes = headerBytes[:n]

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("read_error", "failed to read file header"))
		return
	}

	mtype := mimetype.Detect(headerBytes)
	mime := mtype.String()

	ext := strings.ToLower(filepath.Ext(header.Filename))

	isValidExt := ext == ".pdf" || ext == ".zip"
	isValidMime := mime == "application/pdf" || mime == "application/zip" || mime == "application/x-zip-compressed"

	if !isValidExt || !isValidMime {
		c.JSON(http.StatusUnsupportedMediaType, errorEnvelope("unsupported_media_type", "only PDF or ZIP files are allowed"))
		return
	}

	studentID := c.GetInt64("user_id")

	submission, deadline, err := h.service.Submit(c.Request.Context(), courseID, assignmentID, studentID, file, header.Filename)
	if err != nil {
		if err == ErrWindowClosed {
			c.JSON(http.StatusUnprocessableEntity, errorEnvelope("window_closed", "the submission window has closed"))
			return
		}
		if err == ErrForbidden {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "not enrolled in this course"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": mapSubmission(submission, deadline)})
}

func (h *Handler) GetDownloadURL(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid course id"))
		return
	}

	assignmentID, err := strconv.ParseInt(c.Param("aid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid assignment id"))
		return
	}

	submissionID, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid submission id"))
		return
	}

	userID := c.GetInt64("user_id")
	role := c.GetString("role")

	url, err := h.service.DownloadURL(c.Request.Context(), courseID, assignmentID, submissionID, userID, role)
	if err != nil {
		if err == ErrForbidden {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "access denied"))
			return
		}
		if err == ErrNotFound {
			c.JSON(http.StatusNotFound, errorEnvelope("not_found", "submission not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": url})
}

type GradeRequest struct {
	Score    *float64 `json:"score" binding:"required"`
	Feedback string   `json:"feedback"`
}

func (h *Handler) GradeSubmission(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid course id"))
		return
	}

	assignmentID, err := strconv.ParseInt(c.Param("aid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid assignment id"))
		return
	}

	submissionID, err := strconv.ParseInt(c.Param("sid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_id", "invalid submission id"))
		return
	}

	var req GradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("invalid_request", err.Error()))
		return
	}

	lecturerID := c.GetInt64("user_id")

	err = h.service.GradeSubmission(c.Request.Context(), courseID, assignmentID, submissionID, *req.Score, req.Feedback, lecturerID)
	if err != nil {
		if err == ErrForbidden {
			c.JSON(http.StatusForbidden, errorEnvelope("forbidden", "access denied"))
			return
		}
		if err == ErrNotFound {
			c.JSON(http.StatusNotFound, errorEnvelope("not_found", "submission not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("server_error", err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func mapAssignment(a db.Assignment) AssignmentResponse {
	var threshold *int32
	if a.LateThresholdDays.Valid {
		v := a.LateThresholdDays.Int32
		threshold = &v
	}
	return AssignmentResponse{
		ID:                a.ID,
		Title:             a.Title,
		Description:       a.Description.String,
		Deadline:          a.Deadline.Time,
		AcceptLate:        a.AcceptLate,
		LateThresholdDays: threshold,
		CreatedAt:         a.CreatedAt.Time,
	}
}

func mapSubmission(s db.Submission, deadline time.Time) SubmissionResponse {
	var score *float64
	if s.Score.Valid {
		f, _ := s.Score.Float64Value()
		score = &f.Float64
	}
	var feedback *string
	if s.Feedback.Valid {
		feedback = &s.Feedback.String
	}

	lateDuration := ""
	if s.IsLate {
		dur := s.SubmittedAt.Time.Sub(deadline)
		if dur > 0 {
			days := int(dur.Hours() / 24)
			hours := int(dur.Hours()) % 24
			mins := int(dur.Minutes()) % 60

			parts := []string{}
			if days > 0 {
				parts = append(parts, fmt.Sprintf("%dd", days))
			}
			if hours > 0 {
				parts = append(parts, fmt.Sprintf("%dh", hours))
			}
			if mins > 0 {
				parts = append(parts, fmt.Sprintf("%dm", mins))
			}
			if len(parts) == 0 {
				parts = append(parts, "<1m")
			}
			lateDuration = strings.Join(parts, " ") + " late"
		}
	}

	return SubmissionResponse{
		ID:               s.ID,
		Version:          s.Version,
		OriginalFilename: s.OriginalFilename,
		IsLate:           s.IsLate,
		SubmittedAt:      s.SubmittedAt.Time,
		LateDuration:     lateDuration,
		Score:            score,
		Feedback:         feedback,
	}
}
