package grades

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportScoresCSV(t *testing.T) {
	pool, svc := setupTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	// 1. Create a course and users
	var courseID int64
	err := pool.QueryRow(ctx, `INSERT INTO courses (code, name, term, start_date, end_date) VALUES ('CSV101', 'CSV Tests', 'Fall', now(), now() + interval '1 month') RETURNING id`).Scan(&courseID)
	require.NoError(t, err)

	var lecturerID, student1ID, student2ID int64
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role, full_name) VALUES ('csv_lec', 'hash', 'lecturer', 'Lec') RETURNING id`).Scan(&lecturerID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role, full_name) VALUES ('csv_stu1', 'hash', 'student', 'Stu1') RETURNING id`).Scan(&student1ID)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role, full_name) VALUES ('csv_stu2', 'hash', 'student', 'Stu2') RETURNING id`).Scan(&student2ID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO course_lecturers (course_id, lecturer_id) VALUES ($1, $2)`, courseID, lecturerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO student_enrollments (course_id, student_id) VALUES ($1, $2)`, courseID, student1ID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO student_enrollments (course_id, student_id) VALUES ($1, $2)`, courseID, student2ID)
	require.NoError(t, err)

	// Clean up
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM grade_scores WHERE component_id IN (SELECT c.id FROM grade_components c JOIN grade_schemes s ON c.scheme_id = s.id WHERE s.course_id = $1)`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM grade_components WHERE scheme_id IN (SELECT id FROM grade_schemes WHERE course_id = $1)`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM grade_schemes WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM student_enrollments WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM course_lecturers WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2, $3)`, lecturerID, student1ID, student2ID)
	}()

	// 2. Create Grade Scheme
	var schemeID int64
	err = pool.QueryRow(ctx, `INSERT INTO grade_schemes (course_id, created_by) VALUES ($1, $2) RETURNING id`, courseID, lecturerID).Scan(&schemeID)
	require.NoError(t, err)

	var weight pgtype.Numeric
	_ = weight.Scan("100")
	var compID int64
	err = pool.QueryRow(ctx, `INSERT INTO grade_components (scheme_id, parent_id, name, weight, source_type) VALUES ($1, NULL, 'Project', $2, 'MANUAL') RETURNING id`, schemeID, weight).Scan(&compID)
	require.NoError(t, err)

	// 3. Test CSV with one bad row -> all-or-nothing (422)
	csv1 := "student_id,score\ncsv_stu1,90\ncsv_stu2,150\n"
	errErrs, err := svc.ImportScoresCSV(ctx, courseID, compID, lecturerID, strings.NewReader(csv1))
	require.Error(t, err)
	require.ErrorIs(t, err, ErrValidation)
	require.Len(t, errErrs, 1)
	assert.Equal(t, "score", errErrs[0].Field)
	assert.Contains(t, errErrs[0].Message, "must be between 0 and 100")

	// Ensure zero rows written
	var count int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM grade_scores WHERE component_id = $1`, compID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	// 4. Test Clean CSV
	csv2 := "student_id,score\ncsv_stu1,90\ncsv_stu2,85.5\n"
	errErrs2, err2 := svc.ImportScoresCSV(ctx, courseID, compID, lecturerID, strings.NewReader(csv2))
	require.NoError(t, err2)
	assert.Empty(t, errErrs2)

	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM grade_scores WHERE component_id = $1`, compID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	// Verify scores
	scores, err := svc.repo.ListScoresForStudent(ctx, db.ListScoresForStudentParams{CourseID: courseID, StudentID: student1ID})
	require.NoError(t, err)
	require.Len(t, scores, 1)
	f, _ := scores[0].Score.Float64Value()
	assert.Equal(t, 90.0, f.Float64)
}
