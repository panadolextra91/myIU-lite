package enrollments

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrollIdempotent(t *testing.T) {
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

	// Create a course
	var courseID int64
	err = pool.QueryRow(ctx, `INSERT INTO courses (code, name, term, start_date, end_date) VALUES ('IDEM101', 'Idempotent', 'Fall', now(), now() + interval '1 month') RETURNING id`).Scan(&courseID)
	require.NoError(t, err)
	defer pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, courseID)

	// Create 3 students
	var sA, sB, sC int64
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role, full_name) VALUES ('studentA', 'hash', 'student', 'A') RETURNING id`).Scan(&sA)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role, full_name) VALUES ('studentB', 'hash', 'student', 'B') RETURNING id`).Scan(&sB)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role, full_name) VALUES ('studentC', 'hash', 'student', 'C') RETURNING id`).Scan(&sC)
	require.NoError(t, err)

	defer pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2, $3)`, sA, sB, sC)
	defer pool.Exec(ctx, `DELETE FROM student_enrollments WHERE course_id = $1`, courseID)

	csv1 := "student_id\nstudentA\nstudentB"
	added, errs, err := svc.ImportMembers(ctx, courseID, "student", strings.NewReader(csv1), 1)
	require.NoError(t, err)
	assert.Empty(t, errs)
	assert.Equal(t, int64(2), added)

	csv2 := "student_id\nstudentA\nstudentB\nstudentC"
	added2, errs2, err2 := svc.ImportMembers(ctx, courseID, "student", strings.NewReader(csv2), 1)
	require.NoError(t, err2)
	assert.Empty(t, errs2)
	assert.Equal(t, int64(1), added2)

	var count int
	pool.QueryRow(ctx, `SELECT COUNT(*) FROM student_enrollments WHERE course_id = $1`, courseID).Scan(&count)
	assert.Equal(t, 3, count)
}
