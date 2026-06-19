package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	sharedauth "github.com/panadolextra91/myiu-lite/backend/internal/shared/auth"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/middleware"
)

type Handler struct {
	svc *Service
	cfg config.Config
}

func RegisterRoutes(r *gin.Engine, pool *pgxpool.Pool, cfg config.Config) {
	q := db.New(pool)
	repo := NewRepository(q)
	svc := NewService(repo, cfg)
	h := &Handler{svc: svc, cfg: cfg}

	publicGroup := r.Group("/auth")
	{
		publicGroup.POST("/login", h.Login)
		publicGroup.POST("/refresh", h.Refresh)
	}

	authGroup := r.Group("/auth")
	{
		authGroup.POST("/change-password", middleware.AuthMiddleware(pool, cfg), h.ChangePassword)
		authGroup.POST("/logout", middleware.AuthMiddleware(pool, cfg), h.Logout)
		authGroup.GET("/me", middleware.AuthMiddleware(pool, cfg), h.Me)
		authGroup.GET("/role-check", middleware.AuthMiddleware(pool, cfg), middleware.RequireRole(db.UserRoleAdmin), func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})
	}
}

func (h *Handler) Login(c *gin.Context) {
	if c.ContentType() != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, errorEnvelope("unsupported_media_type", "unsupported media type"))
		return
	}

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("bad_request", "invalid request body"))
		return
	}

	user, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if err == ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, errorEnvelope("invalid_credentials", "Invalid username or password"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", "internal server error"))
		return
	}

	accessToken, err := sharedauth.Mint([]byte(h.cfg.JWTSecret), user.ID, string(user.Role), "access", 15*time.Minute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", "internal server error"))
		return
	}

	refreshToken, err := sharedauth.Mint([]byte(h.cfg.JWTSecret), user.ID, string(user.Role), "refresh", 7*24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", "internal server error"))
		return
	}

	c.SetCookie("access_token", accessToken, int(15*60), "/", "", h.cfg.CookieSecure, true)
	c.SetCookie("refresh_token", refreshToken, int(7*24*60*60), "/", "", h.cfg.CookieSecure, true)

	c.JSON(http.StatusOK, MeResponse{
		ID:                 user.ID,
		Username:           user.Username,
		Role:               string(user.Role),
		MustChangePassword: user.MustChangePassword,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie("access_token", "", -1, "/", "", h.cfg.CookieSecure, true)
	c.SetCookie("refresh_token", "", -1, "/", "", h.cfg.CookieSecure, true)
	c.Status(http.StatusOK)
}

func (h *Handler) Me(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errorEnvelope("unauthorized", "unauthorized"))
		return
	}

	user, err := h.svc.GetUser(c.Request.Context(), userID.(int64))
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorEnvelope("unauthorized", "unauthorized"))
		return
	}

	c.JSON(http.StatusOK, MeResponse{
		ID:                 user.ID,
		Username:           user.Username,
		Role:               string(user.Role),
		MustChangePassword: user.MustChangePassword,
	})
}

func (h *Handler) ChangePassword(c *gin.Context) {
	if c.ContentType() != "application/json" {
		c.JSON(http.StatusUnsupportedMediaType, errorEnvelope("unsupported_media_type", "unsupported media type"))
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errorEnvelope("bad_request", "invalid request body"))
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, errorEnvelope("unauthorized", "unauthorized"))
		return
	}

	err := h.svc.ChangePassword(c.Request.Context(), userID.(int64), req.CurrentPassword, req.NewPassword, req.ConfirmPassword)
	if err != nil {
		if err == ErrConfirmMismatch {
			c.JSON(http.StatusBadRequest, errorEnvelope("confirm_mismatch", "confirm password mismatch"))
			return
		}
		if err == ErrTooShort {
			c.JSON(http.StatusBadRequest, errorEnvelope("password_too_short", "password too short"))
			return
		}
		if err == ErrCurrentPasswordWrong {
			c.JSON(http.StatusBadRequest, errorEnvelope("current_password_invalid", "current password invalid"))
			return
		}
		if err == ErrSameAsCurrent {
			c.JSON(http.StatusBadRequest, errorEnvelope("same_as_current", "same as current password"))
			return
		}
		c.JSON(http.StatusInternalServerError, errorEnvelope("internal_error", "internal server error"))
		return
	}

	c.SetCookie("access_token", "", -1, "/", "", h.cfg.CookieSecure, true)
	c.SetCookie("refresh_token", "", -1, "/", "", h.cfg.CookieSecure, true)
	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully. Please log in again."})
}

func (h *Handler) Refresh(c *gin.Context) {
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorEnvelope("refresh_invalid", "refresh invalid"))
		return
	}

	newAccess, role, userID, err := h.svc.Refresh(c.Request.Context(), cookie, h.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, errorEnvelope("refresh_invalid", "refresh invalid"))
		return
	}

	c.SetCookie("access_token", newAccess, int(15*time.Minute.Seconds()), "/", "", h.cfg.CookieSecure, true)
	c.JSON(http.StatusOK, gin.H{"message": "refreshed successfully", "role": role, "user_id": userID})
}
