package users

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

func TestImportAllOrNothing(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	q := db.New(pool)
	repo := NewRepository(q)
	svc := NewService(pool, repo)

	// Count users before
	countBefore, err := repo.CountUsers(ctx, db.CountUsersParams{})
	require.NoError(t, err)

	// Provide a CSV with 1 valid row and 1 invalid row
	csvData := `student_id,full_name,dob
S12345,John Doe,01/01/2000
S12346,Jane Doe,invalid-date`

	actorID := int64(1) // Assuming SYSTEM or similar exists

	count, rowErrs, err := svc.ImportAccounts(ctx, db.UserRoleStudent, bytes.NewBufferString(csvData), actorID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Len(t, rowErrs, 1)
	assert.Equal(t, 3, rowErrs[0].Row)
	assert.Equal(t, "dob", rowErrs[0].Field)

	// Count users after, should be exactly the same
	countAfter, err := repo.CountUsers(ctx, db.CountUsersParams{})
	require.NoError(t, err)
	assert.Equal(t, countBefore, countAfter)
}
