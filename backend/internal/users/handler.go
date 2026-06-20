package users

import (
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

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

	g := r.Group("/admin")
	g.Use(middleware.AuthMiddleware(pool, cfg), middleware.RequireRole(db.UserRoleAdmin))
	{
		g.POST("/users", h.CreateUser)
		g.POST("/students/import", h.ImportStudents)
		g.POST("/lecturers/import", h.ImportLecturers)
		g.POST("/users/:id/reset-password", h.ResetPassword)
		g.GET("/users", h.ListUsers)
	}
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_REQUEST", err.Error()))
		return
	}

	actorID := c.GetInt64("user_id")
	role := db.UserRole(req.Role)
	if role != db.UserRoleStudent && role != db.UserRoleLecturer {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ROLE", "role must be student or lecturer"))
		return
	}

	id, err := h.svc.CreateAccount(c.Request.Context(), role, req.ID, req.FullName, req.DOB, actorID)
	if err != nil {
		if errors.Is(err, ErrDuplicateUser) {
			c.JSON(http.StatusConflict, errorEnvelope("USER_EXISTS", "User already exists"))
			return
		}
		if errors.Is(err, ErrInvalidDOBFormat) {
			c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_DOB", err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handler) ImportStudents(c *gin.Context) {
	h.handleImport(c, db.UserRoleStudent)
}

func (h *Handler) ImportLecturers(c *gin.Context) {
	h.handleImport(c, db.UserRoleLecturer)
}

func (h *Handler) handleImport(c *gin.Context, role db.UserRole) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 5<<20) // 5MB cap
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_FILE", "Upload a valid CSV file"))
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	contentType := header.Header.Get("Content-Type")
	if ext != ".csv" && contentType != "text/csv" && contentType != "application/vnd.ms-excel" {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_FILE", "File must be a CSV"))
		return
	}

	actorID := c.GetInt64("user_id")
	count, rowErrs, err := h.svc.ImportAccounts(c.Request.Context(), role, file, actorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to import accounts"))
		return
	}

	if len(rowErrs) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"errors": rowErrs})
		return
	}

	c.JSON(http.StatusOK, gin.H{"imported": count})
}

func (h *Handler) ResetPassword(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("INVALID_ID", "Invalid user ID"))
		return
	}

	actorID := c.GetInt64("user_id")
	err = h.svc.ResetPassword(c.Request.Context(), id, actorID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			c.JSON(http.StatusNotFound, errorEnvelope("NOT_FOUND", "User not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to reset password"))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (h *Handler) ListUsers(c *gin.Context) {
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

	var role *db.UserRole
	if r := c.Query("role"); r != "" {
		v := db.UserRole(r)
		role = &v
	}

	var search *string
	if s := c.Query("search"); s != "" {
		search = &s
	}

	users, total, err := h.svc.ListUsers(c.Request.Context(), role, search, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("INTERNAL_ERROR", "Failed to fetch users"))
		return
	}

	res := make([]UserResponse, len(users))
	for i, u := range users {
		var dob string
		if u.DateOfBirth.Valid {
			dob = u.DateOfBirth.Time.Format("02/01/2006")
		}
		res[i] = UserResponse{
			ID:                 u.ID,
			Username:           u.Username,
			FullName:           u.FullName.String,
			Role:               string(u.Role),
			DOB:                dob,
			MustChangePassword: u.MustChangePassword,
			CreatedAt:          u.CreatedAt.Time,
		}
	}

	c.JSON(http.StatusOK, PaginatedUsers{
		Data:  res,
		Total: total,
	})
}
