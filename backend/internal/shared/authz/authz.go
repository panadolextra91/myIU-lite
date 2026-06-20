package authz

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

var (
	ErrForbidden = errors.New("forbidden")
	ErrNotFound  = errors.New("not found")
)

func AssertCourseMember(ctx context.Context, pool *pgxpool.Pool, courseID, userID int64, role db.UserRole) error {
	q := db.New(pool)
	if role == db.UserRoleStudent {
		enrolled, err := q.IsStudentEnrolled(ctx, db.IsStudentEnrolledParams{
			CourseID:  courseID,
			StudentID: userID,
		})
		if err != nil {
			return err
		}
		if !enrolled {
			return ErrForbidden
		}
	} else if role == db.UserRoleLecturer {
		assigned, err := q.IsLecturerAssigned(ctx, db.IsLecturerAssignedParams{
			CourseID:   courseID,
			LecturerID: userID,
		})
		if err != nil {
			return err
		}
		if !assigned {
			return ErrForbidden
		}
	} else {
		return ErrForbidden
	}
	return nil
}

func AssertQuizInCourse(ctx context.Context, pool *pgxpool.Pool, quizID, courseID int64) (db.Quiz, error) {
	q := db.New(pool)
	quiz, err := q.GetQuizByID(ctx, quizID)
	if err != nil {
		return quiz, err
	}
	if quiz.CourseID != courseID {
		return quiz, ErrNotFound
	}
	return quiz, nil
}
