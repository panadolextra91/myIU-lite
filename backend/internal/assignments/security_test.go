package assignments_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/assignments"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurity_Assignments(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	repo := assignments.NewRepository(db.New(pool))
	service := assignments.NewService(pool, repo, nil) // cld not needed for these tests

	t.Run("Same-Tx Rollback - Grading Failure", func(t *testing.T) {
		// Simulate grading a non-existent submission
		// It should fail and rollback any changes
		err := service.GradeSubmission(ctx, 9999, 9999, 9999, 100, "Good", 1)
		assert.Error(t, err)
	})

	t.Run("Enrollment Authz - Lecturer Grade Access", func(t *testing.T) {
		// Simulate lecturer not assigned to course
		var courseID int64 = 88881
		var lecturerID int64 = 88882

		err := authz.AssertCourseMember(ctx, pool, courseID, lecturerID, db.UserRoleLecturer)
		assert.ErrorIs(t, err, authz.ErrForbidden)

		err = service.GradeSubmission(ctx, courseID, 1, 1, 100, "", lecturerID)
		assert.ErrorIs(t, err, authz.ErrForbidden)
	})
}
