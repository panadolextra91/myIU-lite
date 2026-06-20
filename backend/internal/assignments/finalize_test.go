package assignments

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFinalizeGrading(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	q := db.New(pool)
	uniqueStr := fmt.Sprintf("%d", time.Now().UnixNano())

	// Seed lecturer
	lecturerID, err := q.CreateUser(ctx, db.CreateUserParams{
		Username:           "L" + uniqueStr,
		PasswordHash:       "hash",
		Role:               db.UserRoleLecturer,
		FullName:           pgtype.Text{String: "Lecturer", Valid: true},
		DateOfBirth:        pgtype.Date{Valid: true, Time: time.Now()},
		MustChangePassword: false,
	})
	require.NoError(t, err)

	// Seed another lecturer NOT in the course
	otherLecturerID, err := q.CreateUser(ctx, db.CreateUserParams{
		Username:           "L2" + uniqueStr,
		PasswordHash:       "hash",
		Role:               db.UserRoleLecturer,
		FullName:           pgtype.Text{String: "Other Lecturer", Valid: true},
		DateOfBirth:        pgtype.Date{Valid: true, Time: time.Now()},
		MustChangePassword: false,
	})
	require.NoError(t, err)

	// Seed course
	course, err := q.CreateCourse(ctx, db.CreateCourseParams{
		Code:      "C" + uniqueStr,
		Name:      "Course " + uniqueStr,
		Term:      "Fall 2026",
		StartDate: pgtype.Date{Valid: true, Time: time.Now()},
		EndDate:   pgtype.Date{Valid: true, Time: time.Now().Add(90 * 24 * time.Hour)},
	})
	require.NoError(t, err)

	// Assign lecturer to course
	_, err = q.AssignLecturer(ctx, db.AssignLecturerParams{
		CourseID:   course.ID,
		LecturerID: lecturerID,
	})
	require.NoError(t, err)

	// Create assignment with max_score
	var maxScore pgtype.Numeric
	_ = maxScore.Scan("80")
	assignment, err := q.CreateAssignment(ctx, db.CreateAssignmentParams{
		CourseID:          course.ID,
		Title:             "Test Assignment " + uniqueStr,
		Description:       pgtype.Text{String: "desc", Valid: true},
		Deadline:          pgtype.Timestamptz{Time: time.Now().Add(24 * time.Hour), Valid: true},
		AcceptLate:        false,
		LateThresholdDays: pgtype.Int4{},
		CreatedBy:         lecturerID,
		MaxScore:          maxScore,
	})
	require.NoError(t, err)

	// Cleanup
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM assignments WHERE id = $1", assignment.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM course_lecturers WHERE course_id = $1", course.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM courses WHERE id = $1", course.ID)
		_, _ = pool.Exec(ctx, "DELETE FROM users WHERE id IN ($1, $2)", lecturerID, otherLecturerID)
	})

	// Verify: grading_finalized_at starts NULL
	assert.False(t, assignment.GradingFinalizedAt.Valid, "grading_finalized_at should start NULL")

	// Verify: max_score is stored correctly
	maxScoreFloat, _ := assignment.MaxScore.Float64Value()
	assert.Equal(t, float64(80), maxScoreFloat.Float64)

	repo := NewRepository(q)
	service := NewService(pool, repo, nil) // cld not needed

	// Test 1: Non-member lecturer is rejected (authz guard)
	t.Run("Non-member lecturer is forbidden", func(t *testing.T) {
		_, err := service.FinalizeGrading(ctx, course.ID, assignment.ID, otherLecturerID)
		assert.ErrorIs(t, err, ErrForbidden, "Non-member lecturer should be forbidden")
	})

	// Test 2: Happy path — finalize sets grading_finalized_at
	// RED-WHEN-REVERTED: if the `grading_finalized_at = now()` UPDATE in FinalizeAssignmentGrading
	// is removed, the SELECT below would return a NULL grading_finalized_at, and this assertion
	// would fail — proving the test is not trivially passing.
	t.Run("Finalize sets grading_finalized_at", func(t *testing.T) {
		result, err := service.FinalizeGrading(ctx, course.ID, assignment.ID, lecturerID)
		require.NoError(t, err)
		assert.True(t, result.GradingFinalizedAt.Valid, "grading_finalized_at should be set after finalize")

		// Also verify by re-reading from DB
		reread, err := q.GetAssignmentByID(ctx, assignment.ID)
		require.NoError(t, err)
		assert.True(t, reread.GradingFinalizedAt.Valid, "grading_finalized_at should persist in DB")
	})

	// Test 3: Idempotent — already finalized returns not_found (no rows affected)
	t.Run("Already finalized returns ErrNotFound", func(t *testing.T) {
		_, err := service.FinalizeGrading(ctx, course.ID, assignment.ID, lecturerID)
		assert.ErrorIs(t, err, ErrNotFound, "Already-finalized assignment should return not_found")
	})
}
