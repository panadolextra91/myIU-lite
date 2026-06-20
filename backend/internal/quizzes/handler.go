package quizzes

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"path/filepath"
	"time"

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
	repo := NewRepository(db.New(pool))
	svc := NewService(pool, repo)
	h := &Handler{svc: svc}

	api := r.Group("/api")
	api.Use(middleware.AuthMiddleware(pool, cfg))

	lecturer := api.Group("/lecturer")
	lecturer.Use(middleware.RequireRole(db.UserRoleLecturer))
	{
		lecturer.POST("/courses/:id/quizzes", h.CreateQuiz)
		lecturer.GET("/courses/:id/quizzes", h.ListQuizzes)
		lecturer.POST("/courses/:id/quizzes/:qid/questions/import", h.ImportCSV)
		lecturer.POST("/courses/:id/quizzes/:qid/questions", h.AddUIQuestion)
	}
}

func (h *Handler) CreateQuiz(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "invalid course id"))
		return
	}

	var req CreateQuizRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_REQUEST", err.Error()))
		return
	}

	lecturerID := c.GetInt64("user_id")

	quiz, err := h.svc.CreateQuiz(c.Request.Context(), courseID, req, lecturerID)
	if err != nil {
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("FORBIDDEN", "not a lecturer of this course"))
			return
		}
		if errors.Is(err, ErrPoolTooSmall) || errors.Is(err, ErrInvalidDates) {
			c.JSON(http.StatusUnprocessableEntity, errorEnvelope("VALIDATION_ERROR", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("SERVER_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": quiz})
}

func (h *Handler) ListQuizzes(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "invalid course id"))
		return
	}

	quizzes, err := h.svc.repo.ListCourseQuizzes(c.Request.Context(), courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("SERVER_ERROR", err.Error()))
		return
	}
	
	// Format the response array (simplified mapping)
	res := make([]QuizResponse, len(quizzes))
	for i, q := range quizzes {
		var openAt, closeAt *time.Time
		if q.OpenAt.Valid {
			openAt = &q.OpenAt.Time
		}
		if q.CloseAt.Valid {
			closeAt = &q.CloseAt.Time
		}
		maxGrade, _ := q.MaxGrade.Float64Value()
		res[i] = QuizResponse{
			ID:           q.ID,
			Title:        q.Title,
			PoolSize:     q.PoolSize.Int32,
			MaxQuestions: q.MaxQuestions.Int32,
			MaxGrade:     maxGrade.Float64,
			Shuffle:      q.Shuffle.Bool,
			RetakeCount:  q.RetakeCount.Int32,
			OpenAt:       openAt,
			CloseAt:      closeAt,
			CreatedAt:    q.CreatedAt.Time,
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": res})
}

func (h *Handler) ImportCSV(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "invalid course id"))
		return
	}
	quizID, err := strconv.ParseInt(c.Param("qid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "invalid quiz id"))
		return
	}

	if c.Request.ContentLength > 10<<20 {
		c.JSON(http.StatusRequestEntityTooLarge, errorEnvelope("PAYLOAD_TOO_LARGE", "file exceeds 10MB"))
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_REQUEST", "invalid form file"))
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".csv" {
		c.JSON(http.StatusUnsupportedMediaType, errorEnvelope("UNSUPPORTED_MEDIA_TYPE", "only .csv allowed"))
		return
	}

	lecturerID := c.GetInt64("user_id")

	err = h.svc.ImportQuestionsCSV(c.Request.Context(), courseID, quizID, file, lecturerID)
	if err != nil {
		var importErr *ImportError
		if errors.As(err, &importErr) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": importErr.RowErrors})
			return
		}
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("FORBIDDEN", "access denied"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("SERVER_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success"})
}

func (h *Handler) AddUIQuestion(c *gin.Context) {
	courseID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "invalid course id"))
		return
	}
	quizID, err := strconv.ParseInt(c.Param("qid"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "invalid quiz id"))
		return
	}

	var req UIQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_REQUEST", err.Error()))
		return
	}

	lecturerID := c.GetInt64("user_id")

	err = h.svc.AddUIQuestion(c.Request.Context(), courseID, quizID, req, lecturerID)
	if err != nil {
		if errors.Is(err, ErrInvalidQuestion) {
			c.JSON(http.StatusUnprocessableEntity, errorEnvelope("VALIDATION_ERROR", err.Error()))
			return
		}
		if errors.Is(err, ErrForbidden) {
			c.JSON(http.StatusForbidden, errorEnvelope("FORBIDDEN", "access denied"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("SERVER_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success"})
}
