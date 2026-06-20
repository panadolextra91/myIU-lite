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

func setupScoreTestDB(t *testing.T) (*pgxpool.Pool, *grades.Service, *db.Queries) {
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

func TestEnterScore(t *testing.T) {
	pool, svc, _ := setupScoreTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	// Setup data
	ts := time.Now().UnixNano()
	var courseID int64
	err := pool.QueryRow(ctx, fmt.Sprintf(`INSERT INTO courses (code, name, term, start_date, end_date) VALUES ('GSCO_%d', 'Grade Sco', 'Fall', now(), now() + interval '1 month') RETURNING id`, ts)).Scan(&courseID)
	require.NoError(t, err)

	var lecturerID, stEnrolled, stNotEnrolled int64
	lName := fmt.Sprintf("lect_%d", ts)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'lecturer') RETURNING id`, lName).Scan(&lecturerID)
	require.NoError(t, err)

	st1Name := fmt.Sprintf("st_enr_%d", ts)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'student') RETURNING id`, st1Name).Scan(&stEnrolled)
	require.NoError(t, err)

	st2Name := fmt.Sprintf("st_not_%d", ts)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'student') RETURNING id`, st2Name).Scan(&stNotEnrolled)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO course_lecturers (course_id, lecturer_id) VALUES ($1, $2)`, courseID, lecturerID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO student_enrollments (course_id, student_id) VALUES ($1, $2)`, courseID, stEnrolled)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM grade_scores WHERE student_id IN ($1, $2)`, stEnrolled, stNotEnrolled)
		_, _ = pool.Exec(ctx, `DELETE FROM grade_publications WHERE student_id IN ($1, $2)`, stEnrolled, stNotEnrolled)
		
		var schemeID int64
		_ = pool.QueryRow(ctx, `SELECT id FROM grade_schemes WHERE course_id = $1`, courseID).Scan(&schemeID)
		if schemeID > 0 {
			_, _ = pool.Exec(ctx, `DELETE FROM grade_components WHERE scheme_id = $1`, schemeID)
			_, _ = pool.Exec(ctx, `DELETE FROM grade_schemes WHERE id = $1`, schemeID)
		}
		
		_, _ = pool.Exec(ctx, `DELETE FROM student_enrollments WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM course_lecturers WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2, $3)`, lecturerID, stEnrolled, stNotEnrolled)
	})

	components := []grades.ComponentInput{
		{
			Name:       "Test Score",
			Weight:     100,
			SourceType: ptr("MANUAL"),
		},
	}
	_, err = svc.CreateScheme(ctx, courseID, lecturerID, components)
	require.NoError(t, err)

	sch, err := svc.GetScheme(ctx, courseID, lecturerID, "lecturer")
	require.NoError(t, err)
	require.Len(t, sch.Components, 1)
	compID := sch.Components[0].ID

	t.Run("score 0 for enrolled student succeeds", func(t *testing.T) {
		err := svc.EnterScore(ctx, courseID, compID, stEnrolled, 0, lecturerID)
		require.NoError(t, err)
	})

	t.Run("score for not enrolled student fails with ErrNotFound", func(t *testing.T) {
		err := svc.EnterScore(ctx, courseID, compID, stNotEnrolled, 90, lecturerID)
		require.ErrorIs(t, err, grades.ErrNotFound)
	})
}
