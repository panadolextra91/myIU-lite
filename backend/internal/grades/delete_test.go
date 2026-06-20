package grades_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/grades"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/stretchr/testify/require"
)

func ptr(s string) *string { return &s }

func setupDeleteTestDB(t *testing.T) (*pgxpool.Pool, *grades.Service, *db.Queries) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	q := db.New(pool)
	repo := grades.NewRepository(q)
	svc := grades.NewService(pool, repo)

	return pool, svc, q
}

func TestDeleteScheme(t *testing.T) {
	pool, svc, _ := setupDeleteTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	// Setup data
	ts := time.Now().UnixNano()
	var courseID int64
	err := pool.QueryRow(ctx, fmt.Sprintf(`INSERT INTO courses (code, name, term, start_date, end_date) VALUES ('GDEL_%d', 'Grade Del', 'Fall', now(), now() + interval '1 month') RETURNING id`, ts)).Scan(&courseID)
	require.NoError(t, err)

	var lecturerID, stID int64
	lName := fmt.Sprintf("lect_%d", ts)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'lecturer') RETURNING id`, lName).Scan(&lecturerID)
	require.NoError(t, err)

	stName := fmt.Sprintf("st_%d", ts)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'student') RETURNING id`, stName).Scan(&stID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO course_lecturers (course_id, lecturer_id) VALUES ($1, $2)`, courseID, lecturerID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO student_enrollments (course_id, student_id) VALUES ($1, $2)`, courseID, stID)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM grade_scores WHERE student_id = $1`, stID)
		_, _ = pool.Exec(ctx, `DELETE FROM grade_publications WHERE student_id = $1`, stID)
		
		var schemeID int64
		_ = pool.QueryRow(ctx, `SELECT id FROM grade_schemes WHERE course_id = $1`, courseID).Scan(&schemeID)
		if schemeID > 0 {
			_, _ = pool.Exec(ctx, `DELETE FROM grade_components WHERE scheme_id = $1`, schemeID)
			_, _ = pool.Exec(ctx, `DELETE FROM grade_schemes WHERE id = $1`, schemeID)
		}
		
		_, _ = pool.Exec(ctx, `DELETE FROM student_enrollments WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM course_lecturers WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, lecturerID, stID)
	})

	t.Run("create scheme with components, delete succeeds", func(t *testing.T) {
		components := []grades.ComponentInput{
			{
				Name:       "Test",
				Weight:     100,
				SourceType: ptr("MANUAL"),
			},
		}
		_, err := svc.CreateScheme(ctx, courseID, lecturerID, components)
		require.NoError(t, err)

		// Delete scheme
		err = svc.DeleteScheme(ctx, courseID, lecturerID)
		require.NoError(t, err)

		// Verify scheme is deleted
		_, err = svc.GetScheme(ctx, courseID, lecturerID, "lecturer")
		require.ErrorIs(t, err, grades.ErrNotFound)
	})

	t.Run("create scheme with components and score, delete refused", func(t *testing.T) {
		// Re-create scheme
		components := []grades.ComponentInput{
			{
				Name:       "Test",
				Weight:     100,
				SourceType: ptr("MANUAL"),
			},
		}
		_, err := svc.CreateScheme(ctx, courseID, lecturerID, components)
		require.NoError(t, err)

		sch, err := svc.GetScheme(ctx, courseID, lecturerID, "lecturer")
		require.NoError(t, err)
		require.Len(t, sch.Components, 1)

		// Add score
		err = svc.EnterScore(ctx, courseID, sch.Components[0].ID, stID, 90, lecturerID)
		require.NoError(t, err)

		// Delete scheme should fail
		err = svc.DeleteScheme(ctx, courseID, lecturerID)
		require.ErrorIs(t, err, grades.ErrSchemeImmutable)
	})
}
