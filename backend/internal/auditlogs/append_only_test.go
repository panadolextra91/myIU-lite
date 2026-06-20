package auditlogs_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestAppendOnly_Integration(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration test")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	require.NoError(t, err)
	defer conn.Close(ctx)

	var id int64
	err = conn.QueryRow(ctx, "INSERT INTO audit_log (action) VALUES ('TEST_APPEND_ONLY') RETURNING id").Scan(&id)
	require.NoError(t, err, "INSERT should succeed")

	// Test UPDATE
	_, err = conn.Exec(ctx, "UPDATE audit_log SET action = 'MODIFIED' WHERE id = $1", id)
	require.Error(t, err, "UPDATE should fail")
	require.True(t, strings.Contains(err.Error(), "append-only"), "Error should mention append-only")

	// Test DELETE
	_, err = conn.Exec(ctx, "DELETE FROM audit_log WHERE id = $1", id)
	require.Error(t, err, "DELETE should fail")
	require.True(t, strings.Contains(err.Error(), "append-only"), "Error should mention append-only")
}
