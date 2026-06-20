package lifecycle

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSweep(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	q := db.New(pool)

	sysID, err := q.GetSystemUserID(ctx)
	if err != nil {
		// if system user not seeded, create a fake one
		sysID = 1
	}

	// Insert one stale course
	_, err = pool.Exec(ctx, `INSERT INTO courses (code, name, term, start_date, end_date) VALUES ('STALE101', 'Stale Course', 'Spring 2026', now() - interval '4 months', now() - interval '2 months')`)
	require.NoError(t, err)

	// Insert one recent course
	_, err = pool.Exec(ctx, `INSERT INTO courses (code, name, term, start_date, end_date) VALUES ('RECENT101', 'Recent Course', 'Spring 2026', now() - interval '2 months', now() - interval '1 day')`)
	require.NoError(t, err)

	n, err := runSweep(ctx, pool, db.New(pool), sysID)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, n, int64(1))

	// Ensure idempotent
	n2, err := runSweep(ctx, pool, db.New(pool), sysID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n2)

	// Clean up so tests don't pollute
	_, _ = pool.Exec(ctx, `DELETE FROM courses WHERE code IN ('STALE101', 'RECENT101')`)
}

func TestSweepNoOp(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	q := db.New(pool)

	sysID, err := q.GetSystemUserID(ctx)
	if err != nil {
		sysID = 1
	}

	// Run sweep multiple times to ensure no-op is 0
	_, _ = runSweep(ctx, pool, db.New(pool), sysID)

	n, err := runSweep(ctx, pool, db.New(pool), sysID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), n)
}
