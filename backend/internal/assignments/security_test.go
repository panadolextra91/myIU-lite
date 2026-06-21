package assignments_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/assignments"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/authz"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/jackc/pgx/v5/pgtype"
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
		f := testutil.SetupQuizzesFixture(t, ctx, pool)
		// We can reuse the same helper even though it creates quizzes
		// Create an assignment
		q := db.New(pool)
		var maxScore pgtype.Numeric
		_ = maxScore.Scan("100")
		assignment, err := q.CreateAssignment(ctx, db.CreateAssignmentParams{
			CourseID:  f.CourseID,
			Title:     "Test Assign",
			Deadline:  pgtype.Timestamptz{Valid: true, Time: time.Now().Add(24 * time.Hour)},
			CreatedBy: f.LecturerID,
			MaxScore:  maxScore,
		})
		require.NoError(t, err)

		// Create a submission directly
		var subID int64
		err = pool.QueryRow(ctx, "INSERT INTO submissions (assignment_id, student_id, version, cloudinary_public_id, cloudinary_format, original_filename, is_late) VALUES ($1, $2, 1, 'pid', 'pdf', 'test.pdf', false) RETURNING id", assignment.ID, f.StudentID).Scan(&subID)
		require.NoError(t, err)

		// Simulate grading failure
		// The score is clamped so it won't fail the Go validations.
		// To cause a DB failure *during* the transaction after the submission is updated, we can pass a feedback string that exceeds DB column length? No, text has no limit.
		// Alternatively, we can drop the notifications table in another connection! Or just rely on the fact that if we pass an invalid assignment_id, it fails early. 
		// Actually, to truly test rollback, let's drop notifications table momentarily (in a transaction) so grading fails at the notification step.
		tx, _ := pool.Begin(ctx)
		_, _ = tx.Exec(ctx, "ALTER TABLE notifications RENAME TO notifications_hidden")
		_ = tx.Commit(ctx)
		// Restore in a defer so a failed assertion or panic below can never leave the
		// shared notifications table renamed for the rest of the suite. NOTE: this
		// rename is GLOBAL/committed — the whole DB suite must run with `go test -p 1`
		// (see scripts/check.sh / ci.yml) so no parallel package inserts a notification
		// during this window.
		defer func() {
			tx2, _ := pool.Begin(ctx)
			_, _ = tx2.Exec(ctx, "ALTER TABLE notifications_hidden RENAME TO notifications")
			_ = tx2.Commit(ctx)
		}()

		err = service.GradeSubmission(ctx, f.CourseID, assignment.ID, subID, 100, "Good", f.LecturerID)
		assert.Error(t, err)

		// GradedAt should be null for the real submission because it was rolled back
		var gradedAt pgtype.Timestamptz
		err = pool.QueryRow(ctx, "SELECT graded_at FROM submissions WHERE id = $1", subID).Scan(&gradedAt)
		require.NoError(t, err)
		assert.False(t, gradedAt.Valid, "GradedAt should be null, tx must be rolled back")
	})

	t.Run("Enrollment Authz - Lecturer Grade Access", func(t *testing.T) {
		f := testutil.SetupQuizzesFixture(t, ctx, pool)
		
		// Create another lecturer not in the course
		uniqueStr := fmt.Sprintf("other-%d", time.Now().UnixNano())
		otherLecturer, err := db.New(pool).CreateUser(ctx, db.CreateUserParams{
			Username: "L2" + uniqueStr,
			Role: db.UserRoleLecturer,
		})
		require.NoError(t, err)

		err = service.GradeSubmission(ctx, f.CourseID, 1, 1, 100, "", otherLecturer)
		assert.ErrorIs(t, err, authz.ErrForbidden)
	})
}
