package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/auth"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthRefreshIntegration(t *testing.T) {
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

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	testUserHash, _ := bcrypt.GenerateFromPassword([]byte("123456"), 12)

	suffix := time.Now().UnixNano()
	adminUsername := fmt.Sprintf("refresh_tester_admin_%d", suffix)
	lecturerUsername := fmt.Sprintf("refresh_tester_lecturer_%d", suffix)

	var adminID int64
	err = pool.QueryRow(ctx, "INSERT INTO users (username, password_hash, role, must_change_password, password_changed_at) VALUES ($1, $2, 'admin', false, now() - interval '1 hour') RETURNING id", adminUsername, string(testUserHash)).Scan(&adminID)
	require.NoError(t, err)

	var lecturerID int64
	err = pool.QueryRow(ctx, "INSERT INTO users (username, password_hash, role, must_change_password, password_changed_at) VALUES ($1, $2, 'lecturer', false, now() - interval '1 hour') RETURNING id", lecturerUsername, string(testUserHash)).Scan(&lecturerID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(), "DELETE FROM users WHERE id IN ($1, $2)", adminID, lecturerID)
		if err != nil {
			t.Logf("cleanup failed: %v", err)
		}
		pool.Close()
	})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.SetSameSite(http.SameSiteLaxMode)
		c.Next()
	})
	auth.RegisterRoutes(router, pool, cfg)

	var adminRefreshCookie *http.Cookie
	var adminAccessCookie *http.Cookie
	var lecturerAccessCookie *http.Cookie

	t.Run("login as admin to get refresh token", func(t *testing.T) {
		reqBody := auth.LoginRequest{Username: adminUsername, Password: "123456"}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		for _, c := range w.Result().Cookies() {
			if c.Name == "refresh_token" {
				adminRefreshCookie = c
			}
			if c.Name == "access_token" {
				adminAccessCookie = c
			}
		}
		require.NotNil(t, adminRefreshCookie)
	})

	t.Run("login as lecturer to get access token", func(t *testing.T) {
		reqBody := auth.LoginRequest{Username: lecturerUsername, Password: "123456"}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		for _, c := range w.Result().Cookies() {
			if c.Name == "access_token" {
				lecturerAccessCookie = c
			}
		}
	})

	t.Run("role-check passes for admin", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/role-check", nil)
		req.AddCookie(adminAccessCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("role-check fails for lecturer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/role-check", nil)
		req.AddCookie(lecturerAccessCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		errMap := resp["error"].(map[string]interface{})
		require.Equal(t, "role_forbidden", errMap["code"])
	})

	t.Run("POST /auth/refresh with valid refresh cookie", func(t *testing.T) {
		time.Sleep(1 * time.Second) // ensure the new token has a different timestamp
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.AddCookie(adminRefreshCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)

		var newAccessCookie *http.Cookie
		for _, c := range w.Result().Cookies() {
			if c.Name == "access_token" {
				newAccessCookie = c
			}
		}
		require.NotNil(t, newAccessCookie)
		require.NotEqual(t, adminAccessCookie.Value, newAccessCookie.Value) // New token minted
	})

	t.Run("refresh token used as access token fails", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: adminRefreshCookie.Value})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("access token used as refresh token fails", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.AddCookie(&http.Cookie{Name: "refresh_token", Value: adminAccessCookie.Value})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("POST /auth/refresh after password change fails", func(t *testing.T) {
		_, err := pool.Exec(context.Background(), "UPDATE users SET password_changed_at = now() WHERE id = $1", adminID)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
		req.AddCookie(adminRefreshCookie) 
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
		var resp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		errMap := resp["error"].(map[string]interface{})
		require.Equal(t, "refresh_invalid", errMap["code"])
	})
}
