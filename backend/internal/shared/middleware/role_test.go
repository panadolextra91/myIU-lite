package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/middleware"
	"github.com/stretchr/testify/require"
)

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		userRole     string
		allowedRoles []db.UserRole
		expectedCode int
	}{
		{
			name:         "admin passes admin role",
			userRole:     string(db.UserRoleAdmin),
			allowedRoles: []db.UserRole{db.UserRoleAdmin},
			expectedCode: http.StatusOK,
		},
		{
			name:         "lecturer rejected from admin role",
			userRole:     string(db.UserRoleLecturer),
			allowedRoles: []db.UserRole{db.UserRoleAdmin},
			expectedCode: http.StatusForbidden,
		},
		{
			name:         "student passes student/lecturer",
			userRole:     string(db.UserRoleStudent),
			allowedRoles: []db.UserRole{db.UserRoleStudent, db.UserRoleLecturer},
			expectedCode: http.StatusOK,
		},
		{
			name:         "missing role in context",
			userRole:     "", // simulate missing
			allowedRoles: []db.UserRole{db.UserRoleAdmin},
			expectedCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()

			router.Use(func(c *gin.Context) {
				if tt.userRole != "" {
					c.Set("role", tt.userRole)
				}
				c.Next()
			})

			router.GET("/test", middleware.RequireRole(tt.allowedRoles...), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			require.Equal(t, tt.expectedCode, w.Code)
		})
	}
}
