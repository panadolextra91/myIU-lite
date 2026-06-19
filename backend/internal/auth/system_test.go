package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/panadolextra91/myiu-lite/backend/internal/auth"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
)

func TestSystemNoLogin_Integration(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	cfg := config.Config{
		JWTSecret: "test-secret",
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	auth.RegisterRoutes(router, pool, cfg)

	// Attempt to login as system user
	reqBody := auth.LoginRequest{
		Username: "__system__",
		Password: "!",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify we get a 401 Unauthorized
	require.Equal(t, http.StatusUnauthorized, w.Code)
}
