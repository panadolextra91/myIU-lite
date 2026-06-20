package users

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

func TestResetPassword(t *testing.T) {
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

	// create a dummy user
	username := "reset_test_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	id, err := svc.CreateAccount(ctx, db.UserRoleStudent, username, "Reset User", "15/08/1999", 1)
	require.NoError(t, err)

	defer func() {
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	}()

	// wait a moment so timestamp differences are obvious, though not strictly necessary
	time.Sleep(10 * time.Millisecond)

	err = svc.ResetPassword(ctx, id, 1)
	require.NoError(t, err)

	user, err := repo.GetUserByID(ctx, id)
	require.NoError(t, err)

	assert.True(t, user.MustChangePassword)
	// password_changed_at shouldn't be null, and should be recent
	assert.True(t, user.PasswordChangedAt.Valid)
}
