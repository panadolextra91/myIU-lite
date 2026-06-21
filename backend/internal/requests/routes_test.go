package requests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression guard for the Phase-5 auth-wiring bug: the requests routes must sit
// BEHIND AuthMiddleware, not just RequireRole. With AuthMiddleware in the chain, an
// unauthenticated request is rejected at auth with 401. If AuthMiddleware is removed
// (the bug), RequireRole runs first with no "role" in context and returns 403 — so
// this test goes RED the moment the wiring regresses.
func TestRoutesSitBehindAuthMiddleware(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	require.NoError(t, err)
	defer pool.Close()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, pool, config.Config{})

	for _, path := range []string{
		"/api/student/requests",
		"/api/lecturer/requests",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"%s: unauthenticated request must hit AuthMiddleware (401), not RequireRole (403)", path)
	}
}
