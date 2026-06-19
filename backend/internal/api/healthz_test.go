package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/api"
	"github.com/panadolextra91/myiu-lite/backend/internal/db"
	"github.com/stretchr/testify/require"
)

func TestHealthz_Integration(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	queries := db.New(pool)
	
	// Verify seeded admin count
	count, err := queries.CountUsers(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), count, "Expected exactly 1 seeded admin")

	// Verify /healthz HTTP response
	router := api.NewRouter(pool)
	
	req, _ := http.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	require.Equal(t, http.StatusOK, w.Code)
	
	var body map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	require.Equal(t, "ok", body["status"])
}
