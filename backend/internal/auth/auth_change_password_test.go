package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/auth"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/middleware"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"fmt"
)

func TestChangePasswordIntegration(t *testing.T) {
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
	testUsername := fmt.Sprintf("change_pwd_tester_%d", suffix)

	var userID int64
	err = pool.QueryRow(ctx, "INSERT INTO users (username, password_hash, role, must_change_password, password_changed_at) VALUES ($1, $2, 'student', true, now() - interval '1 hour') RETURNING id", testUsername, string(testUserHash)).Scan(&userID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
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

	router.GET("/api/dummy", middleware.AuthMiddleware(pool, cfg), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	var accessTokenCookie *http.Cookie

	t.Run("login", func(t *testing.T) {
		reqBody := auth.LoginRequest{Username: testUsername, Password: "123456"}
		bodyBytes, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		for _, c := range w.Result().Cookies() {
			if c.Name == "access_token" {
				accessTokenCookie = c
			}
		}
		require.NotNil(t, accessTokenCookie)
	})

	t.Run("non-allowlisted route returns 403", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/dummy", nil)
		req.AddCookie(accessTokenCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusForbidden, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		errMap := resp["error"].(map[string]interface{})
		require.Equal(t, "password_change_required", errMap["code"])
	})

	t.Run("GET /auth/me returns 200 (allow-listed)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		req.AddCookie(accessTokenCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("POST /auth/change-password with wrong current", func(t *testing.T) {
		reqBody := auth.ChangePasswordRequest{
			CurrentPassword: "wrong",
			NewPassword:     "newpass1",
			ConfirmPassword: "newpass1",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(accessTokenCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		errMap := resp["error"].(map[string]interface{})
		require.Equal(t, "current_password_invalid", errMap["code"])
	})

	t.Run("POST /auth/change-password too short", func(t *testing.T) {
		reqBody := auth.ChangePasswordRequest{
			CurrentPassword: "123456",
			NewPassword:     "short",
			ConfirmPassword: "short",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(accessTokenCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		errMap := resp["error"].(map[string]interface{})
		require.Equal(t, "password_too_short", errMap["code"])
	})

	t.Run("POST /auth/change-password same as current", func(t *testing.T) {
		reqBody := auth.ChangePasswordRequest{
			CurrentPassword: "123456",
			NewPassword:     "123456",
			ConfirmPassword: "123456",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(accessTokenCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		errMap := resp["error"].(map[string]interface{})
		require.Equal(t, "same_as_current", errMap["code"])
	})

	t.Run("POST /auth/change-password confirm mismatch", func(t *testing.T) {
		reqBody := auth.ChangePasswordRequest{
			CurrentPassword: "123456",
			NewPassword:     "newpass1",
			ConfirmPassword: "newpass2",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(accessTokenCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusBadRequest, w.Code)
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		errMap := resp["error"].(map[string]interface{})
		require.Equal(t, "confirm_mismatch", errMap["code"])
	})

	t.Run("POST /auth/change-password success", func(t *testing.T) {
		reqBody := auth.ChangePasswordRequest{
			CurrentPassword: "123456",
			NewPassword:     "newpass1",
			ConfirmPassword: "newpass1",
		}
		bodyBytes, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/auth/change-password", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
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

	// To test that the old access cookie is invalid, we need to wait a second so that the password_changed_at stamp is 
	// strictly greater than the token's issued at time. Since timestamps are at the second precision in Postgres and JWT.
	t.Run("old access token is now invalid", func(t *testing.T) {
		// Just sleep a tiny bit to be safe, though the UpdatePasswordAndStamp should be > IssuedAt
		time.Sleep(1 * time.Second)
		req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
		req.AddCookie(accessTokenCookie)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
