package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/auth"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

func AuthMiddleware(pool *pgxpool.Pool, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("access_token")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "unauthorized"}})
			return
		}

		claims, err := auth.Parse([]byte(cfg.JWTSecret), cookie)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "unauthorized"}})
			return
		}

		if claims.Type != "access" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "unauthorized"}})
			return
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "unauthorized"}})
			return
		}

		queries := db.New(pool)
		user, err := queries.GetUserByID(c.Request.Context(), userID)
		if err != nil {
			// not found or deleted
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "unauthorized"}})
			return
		}

		if claims.IssuedAt != nil && claims.IssuedAt.Time.Before(user.PasswordChangedAt.Time) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "session_expired", "message": "session expired"}})
			return
		}

		// SEAM: plan 02 will insert step 5 (must_change_password allow-list) here.
		if user.MustChangePassword {
			path := c.Request.URL.Path
			method := c.Request.Method
			isAllowed := (method == http.MethodPost && path == "/auth/change-password") ||
				(method == http.MethodPost && path == "/auth/logout") ||
				(method == http.MethodGet && path == "/auth/me")

			if !isAllowed {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "password_change_required", "message": "password change required"}})
				return
			}
		}

		c.Set("user_id", user.ID)
		c.Set("role", string(user.Role))
		c.Next()
	}
}
