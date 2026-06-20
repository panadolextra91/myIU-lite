package grades

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, *Service) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	q := db.New(pool)
	repo := NewRepository(q)
	svc := NewService(pool, repo)

	return pool, svc
}

func TestComputeOverallForStudent(t *testing.T) {
	pool, svc := setupTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	// 1. Create a course and user
	var courseID int64
	err := pool.QueryRow(ctx, `INSERT INTO courses (code, name, term, start_date, end_date) VALUES ('GRADE101', 'Grades', 'Fall', now(), now() + interval '1 month') RETURNING id`).Scan(&courseID)
	require.NoError(t, err)

	var lecturerID, studentID int64
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role, full_name) VALUES ('grade_lec', 'hash', 'lecturer', 'Lec') RETURNING id`).Scan(&lecturerID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role, full_name) VALUES ('grade_stu', 'hash', 'student', 'Stu') RETURNING id`).Scan(&studentID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO course_lecturers (course_id, lecturer_id) VALUES ($1, $2)`, courseID, lecturerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO student_enrollments (course_id, student_id) VALUES ($1, $2)`, courseID, studentID)
	require.NoError(t, err)

	// Clean up at the end
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM grade_scores WHERE student_id = $1`, studentID)
		_, _ = pool.Exec(ctx, `DELETE FROM grade_components WHERE scheme_id IN (SELECT id FROM grade_schemes WHERE course_id = $1)`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM grade_schemes WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM submissions WHERE student_id = $1`, studentID)
		_, _ = pool.Exec(ctx, `DELETE FROM assignments WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM quiz_attempts WHERE student_id = $1`, studentID)
		_, _ = pool.Exec(ctx, `DELETE FROM quizzes WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM student_enrollments WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM course_lecturers WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2)`, lecturerID, studentID)
	}()

	// 2. Create quizzes and attempts
	var q1ID, q2ID int64
	// q1: past close_at, max_grade=10
	err = pool.QueryRow(ctx, `INSERT INTO quizzes (course_id, title, max_grade, close_at, created_by) VALUES ($1, 'Q1', 10, now() - interval '1 day', $2) RETURNING id`, courseID, lecturerID).Scan(&q1ID)
	require.NoError(t, err)
	// q2: past close_at, max_grade=20
	err = pool.QueryRow(ctx, `INSERT INTO quizzes (course_id, title, max_grade, close_at, created_by) VALUES ($1, 'Q2', 20, now() - interval '1 day', $2) RETURNING id`, courseID, lecturerID).Scan(&q2ID)
	require.NoError(t, err)

	// Student attempts q1, gets 8. (8/10 = 80%)
	_, err = pool.Exec(ctx, `INSERT INTO quiz_attempts (quiz_id, student_id, score, status, started_at) VALUES ($1, $2, 8, 'SUBMITTED', now())`, q1ID, studentID)
	require.NoError(t, err)
	// Student skips q2 -> gets 0. (0/20 = 0%)
	// Quiz Average = (80 + 0) / 2 = 40.

	// 3. Create assignments and submissions
	var a1ID int64
	// a1: grading NOT finalized initially, max_score=50
	err = pool.QueryRow(ctx, `INSERT INTO assignments (course_id, title, deadline, max_score, created_by) VALUES ($1, 'A1', now(), 50, $2) RETURNING id`, courseID, lecturerID).Scan(&a1ID)
	require.NoError(t, err)
	// Student submits 25. (25/50 = 50%)
	_, err = pool.Exec(ctx, `INSERT INTO submissions (assignment_id, student_id, version, score, graded_at, graded_by, cloudinary_public_id, original_filename, cloudinary_format, is_late) VALUES ($1, $2, 1, 25, now(), $3, 'dummy_id', 'dummy.pdf', 'pdf', false)`, a1ID, studentID, lecturerID)
	require.NoError(t, err)

	// 4. Create Grade Scheme
	// Inclass(50) [Quizzes(60) + Assignments(40)], Midterm(25), Final(25)
	var stManual, stAuto, akQuiz, akAss pgtype.Text
	_ = stManual.Scan("MANUAL")
	_ = stAuto.Scan("AUTO")
	_ = akQuiz.Scan("QUIZ_AVERAGE")
	_ = akAss.Scan("ASSIGNMENT_AVERAGE")

	var inclassWeight, quizWeight, assWeight, mtWeight, fnWeight pgtype.Numeric
	_ = inclassWeight.Scan("50")
	_ = quizWeight.Scan("60")
	_ = assWeight.Scan("40")
	_ = mtWeight.Scan("25")
	_ = fnWeight.Scan("25")

	var schemeID int64
	err = pool.QueryRow(ctx, `INSERT INTO grade_schemes (course_id, created_by) VALUES ($1, $2) RETURNING id`, courseID, lecturerID).Scan(&schemeID)
	require.NoError(t, err)

	var inclassID, mtID, fnID int64
	err = pool.QueryRow(ctx, `INSERT INTO grade_components (scheme_id, parent_id, name, weight) VALUES ($1, NULL, 'Inclass', $2) RETURNING id`, schemeID, inclassWeight).Scan(&inclassID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO grade_components (scheme_id, parent_id, name, weight, source_type) VALUES ($1, NULL, 'Midterm', $2, 'MANUAL') RETURNING id`, schemeID, mtWeight).Scan(&mtID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO grade_components (scheme_id, parent_id, name, weight, source_type) VALUES ($1, NULL, 'Final', $2, 'MANUAL') RETURNING id`, schemeID, fnWeight).Scan(&fnID)
	require.NoError(t, err)

	// Sub-components of Inclass
	_, err = pool.Exec(ctx, `INSERT INTO grade_components (scheme_id, parent_id, name, weight, source_type, auto_kind) VALUES ($1, $2, 'Quizzes', $3, 'AUTO', 'QUIZ_AVERAGE')`, schemeID, inclassID, quizWeight)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO grade_components (scheme_id, parent_id, name, weight, source_type, auto_kind) VALUES ($1, $2, 'Assignments', $3, 'AUTO', 'ASSIGNMENT_AVERAGE')`, schemeID, inclassID, assWeight)
	require.NoError(t, err)

	// Enter MANUAL scores: Midterm 90, Final 80
	err = svc.EnterScore(ctx, courseID, mtID, studentID, 90, lecturerID)
	require.NoError(t, err)
	err = svc.EnterScore(ctx, courseID, fnID, studentID, 80, lecturerID)
	require.NoError(t, err)

	// 5. Test Compute (assignment not finalized -> excluded)
	// Quizzes = 40. Assignments = excluded (not 0).
	// Inclass = Quizzes(40) * 0.6 + Assignments(0) = 24.  Wait, if it's 0 because missing, it counts as 0? No, if it's excluded from eligible, does computeLeaf return 0?
	// Ah, my computeLeaf returns 0 for assignments if count=0. Let's see:
	// ComputeAssignmentAverage returns 0 if no eligible assignments.
	// So Assignments = 0.
	// Inclass = 40 * 0.6 + 0 * 0.4 = 24.
	// Overall = Inclass(24)*0.5 + Midterm(90)*0.25 + Final(80)*0.25 = 12 + 22.5 + 20 = 54.5.

	res, err := svc.ComputeOverallForStudent(ctx, courseID, studentID)
	require.NoError(t, err)
	assert.Equal(t, 54.5, res.Overall)

	// 6. Finalize Assignment -> eligible
	_, err = pool.Exec(ctx, `UPDATE assignments SET grading_finalized_at = now() WHERE id = $1`, a1ID)
	require.NoError(t, err)

	// Now Assignments = 50.
	// Inclass = Quizzes(40)*0.6 + Assignments(50)*0.4 = 24 + 20 = 44.
	// Overall = Inclass(44)*0.5 + Midterm(90)*0.25 + Final(80)*0.25 = 22 + 22.5 + 20 = 64.5.

	res2, err := svc.ComputeOverallForStudent(ctx, courseID, studentID)
	require.NoError(t, err)
	assert.Equal(t, 64.5, res2.Overall)
}
