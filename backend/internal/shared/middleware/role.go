package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

func RequireRole(roles ...db.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRoleStr, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "role_forbidden", "message": "role forbidden"}})
			return
		}

		userRole := db.UserRole(userRoleStr.(string))
		isAllowed := false
		for _, role := range roles {
			if userRole == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "role_forbidden", "message": "role forbidden"}})
			return
		}

		c.Next()
	}
}
