package testutil

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/stretchr/testify/require"
)

type Fixture struct {
	LecturerID int64
	StudentID  int64
	CourseID   int64
	QuizID     int64
	Questions  []db.QuizQuestion
	Options    []db.QuizQuestionOption
}

func SetupQuizzesFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) Fixture {
	q := db.New(pool)
	
	uniqueStr := fmt.Sprintf("%d", time.Now().UnixNano())
	
	// Create lecturer
	lecturer, err := q.CreateUser(ctx, db.CreateUserParams{
		Username:           "L" + uniqueStr,
		PasswordHash:       "hash",
		Role:               db.UserRoleLecturer,
		FullName:           pgtype.Text{String: "Lecturer " + uniqueStr, Valid: true},
		DateOfBirth:        pgtype.Date{Valid: true, Time: time.Now()},
		MustChangePassword: false,
	})
	require.NoError(t, err)

	// Create student
	student, err := q.CreateUser(ctx, db.CreateUserParams{
		Username:           "S" + uniqueStr,
		PasswordHash:       "hash",
		Role:               db.UserRoleStudent,
		FullName:           pgtype.Text{String: "Student " + uniqueStr, Valid: true},
		DateOfBirth:        pgtype.Date{Valid: true, Time: time.Now()},
		MustChangePassword: false,
	})
	require.NoError(t, err)

	// Create course
	course, err := q.CreateCourse(ctx, db.CreateCourseParams{
		Code: "C" + uniqueStr,
		Name: "Course " + uniqueStr,
		Term: "Fall 2026",
	})
	require.NoError(t, err)

	// Enroll lecturer
	_, err = q.AssignLecturer(ctx, db.AssignLecturerParams{
		CourseID:   course.ID,
		LecturerID: lecturer,
	})
	require.NoError(t, err)

	// Enroll student
	_, err = q.EnrollStudent(ctx, db.EnrollStudentParams{
		CourseID:  course.ID,
		StudentID: student,
	})
	require.NoError(t, err)

	// Create quiz
	quiz, err := q.CreateQuiz(ctx, db.CreateQuizParams{
		CourseID:    course.ID,
		Title:       "Quiz " + uniqueStr,
		OpenAt:      pgtype.Timestamptz{Valid: true, Time: time.Now().Add(-1 * time.Hour)},
		CloseAt:     pgtype.Timestamptz{Valid: true, Time: time.Now().Add(1 * time.Hour)}, // Open window by default
		CreatedBy:   lecturer,
		MaxGrade:    pgtype.Numeric{Int: nil, Valid: false}, // optional
	})
	require.NoError(t, err)

	// Create questions
	q1, err := q.InsertQuestion(ctx, db.InsertQuestionParams{
		QuizID:       quiz.ID,
		QuestionType: "single",
		Prompt:       pgtype.Text{String: "Q1", Valid: true},
	})
	require.NoError(t, err)

	q2, err := q.InsertQuestion(ctx, db.InsertQuestionParams{
		QuizID:       quiz.ID,
		QuestionType: "single",
		Prompt:       pgtype.Text{String: "Q2", Valid: true},
	})
	require.NoError(t, err)

	// Create options
	o1, _ := q.InsertOption(ctx, db.InsertOptionParams{QuestionID: q1.ID, Text: pgtype.Text{String: "A1", Valid: true}, IsCorrect: true})
	o2, _ := q.InsertOption(ctx, db.InsertOptionParams{QuestionID: q1.ID, Text: pgtype.Text{String: "B1", Valid: true}, IsCorrect: false})
	o3, _ := q.InsertOption(ctx, db.InsertOptionParams{QuestionID: q2.ID, Text: pgtype.Text{String: "A2", Valid: true}, IsCorrect: true})
	o4, _ := q.InsertOption(ctx, db.InsertOptionParams{QuestionID: q2.ID, Text: pgtype.Text{String: "B2", Valid: true}, IsCorrect: false})

	t.Cleanup(func() {
		// Clean up
		pool.Exec(ctx, "DELETE FROM quiz_question_options WHERE question_id IN ($1, $2)", q1.ID, q2.ID)
		pool.Exec(ctx, "DELETE FROM quiz_questions WHERE quiz_id = $1", quiz.ID)
		pool.Exec(ctx, "DELETE FROM quizzes WHERE id = $1", quiz.ID)
		pool.Exec(ctx, "DELETE FROM course_lecturers WHERE course_id = $1", course.ID)
		pool.Exec(ctx, "DELETE FROM student_enrollments WHERE course_id = $1", course.ID)
		pool.Exec(ctx, "DELETE FROM courses WHERE id = $1", course.ID)
		pool.Exec(ctx, "DELETE FROM users WHERE id IN ($1, $2)", lecturer, student)
	})

	return Fixture{
		LecturerID: lecturer,
		StudentID:  student,
		CourseID:   course.ID,
		QuizID:     quiz.ID,
		Questions:  []db.QuizQuestion{q1, q2},
		Options:    []db.QuizQuestionOption{o1, o2, o3, o4},
	}
}
