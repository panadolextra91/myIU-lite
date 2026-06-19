package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/auth"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"time"
	"context"
	"fmt"
)

func TestAuthLoginIntegration(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	cfg := config.Config{
		DatabaseURL:   dbURL,
		JWTSecret:     "testsecret",
		CloudinaryURL: "cloudinary://test",
		Port:          "8080",
		CookieSecure:  false,
	}

	ctx := t.Context()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.SetSameSite(http.SameSiteLaxMode)
		c.Next()
	})
	auth.RegisterRoutes(router, pool, cfg)

	testUserHash, _ := bcrypt.GenerateFromPassword([]byte("123456"), 12)
	suffix := time.Now().UnixNano()
	adminUsername := fmt.Sprintf("login_tester_admin_%d", suffix)

	var adminID int64
	err = pool.QueryRow(context.Background(), "INSERT INTO users (username, password_hash, role, must_change_password, password_changed_at) VALUES ($1, $2, 'admin', true, now() - interval '1 hour') RETURNING id", adminUsername, string(testUserHash)).Scan(&adminID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", adminID)
		if err != nil {
			t.Logf("cleanup failed: %v", err)
		}
		pool.Close()
	})

	var accessTokenCookie *http.Cookie

	t.Run("POST /auth/login success", func(t *testing.T) {
		reqBody := auth.LoginRequest{Username: adminUsername, Password: "123456"}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		cookies := w.Result().Cookies()
		require.Len(t, cookies, 2)

		var access, refresh *http.Cookie
		for _, c := range cookies {
			if c.Name == "access_token" {
				access = c
			}
			if c.Name == "refresh_token" {
				refresh = c
			}
		}
		require.NotNil(t, access)
		require.NotNil(t, refresh)
		require.True(t, access.HttpOnly)
		require.True(t, refresh.HttpOnly)
		require.Equal(t, http.SameSiteLaxMode, access.SameSite)
		require.Equal(t, http.SameSiteLaxMode, refresh.SameSite)

		accessTokenCookie = access
	})

	t.Run("POST /auth/login wrong password", func(t *testing.T) {
		reqBody := auth.LoginRequest{Username: adminUsername, Password: "wrongpassword"}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		errMap := resp["error"].(map[string]interface{})
		require.Equal(t, "invalid_credentials", errMap["code"])
	})

	t.Run("POST /auth/login wrong content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte("username=admin")))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnsupportedMediaType, w.Code)
	})

	t.Run("GET /auth/me success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		req.AddCookie(accessTokenCookie)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		var resp auth.MeResponse
		json.Unmarshal(w.Body.Bytes(), &resp)
		require.Equal(t, "admin", resp.Role)
		require.True(t, resp.MustChangePassword)
	})

	t.Run("POST /auth/logout", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
		req.AddCookie(accessTokenCookie)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		cookies := w.Result().Cookies()
		require.Len(t, cookies, 2)
		for _, c := range cookies {
			require.Equal(t, -1, c.MaxAge)
		}
	})
}
