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
	"fmt"
	"time"
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

	// (removed countBefore to avoid flaky concurrent scope issue)

	// Provide a CSV with 1 valid row and 1 invalid row, using very unique IDs to avoid flake
	uniqueID1 := fmt.Sprintf("S%d-1", time.Now().UnixNano())
	uniqueID2 := fmt.Sprintf("S%d-2", time.Now().UnixNano())
	csvData := fmt.Sprintf(`student_id,full_name,dob
%s,John Doe,01/01/2000
%s,Jane Doe,invalid-date`, uniqueID1, uniqueID2)

	actorID := int64(1) // Assuming SYSTEM or similar exists

	count, rowErrs, err := svc.ImportAccounts(ctx, db.UserRoleStudent, bytes.NewBufferString(csvData), actorID)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Len(t, rowErrs, 1)
	assert.Equal(t, 3, rowErrs[0].Row)
	assert.Equal(t, "dob", rowErrs[0].Field)

	// Instead of comparing countBefore == countAfter, we explicitly assert that the valid user was NOT inserted (rollback)
	// We can check GetActiveUsernames
	activeUsernames, err := repo.q.GetActiveUsernames(ctx, []string{uniqueID1})
	require.NoError(t, err)
	assert.Len(t, activeUsernames, 0, "No users should have been inserted due to rollback")
}
