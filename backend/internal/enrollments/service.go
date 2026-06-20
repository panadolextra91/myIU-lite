package enrollments

import (
	"context"
	"io"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/auditlogs"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

type Service struct {
	pool *pgxpool.Pool
	repo *Repository
	q    *db.Queries
}

func NewService(pool *pgxpool.Pool, repo *Repository) *Service {
	return &Service{pool: pool, repo: repo, q: db.New(pool)}
}

func (s *Service) ImportMembers(ctx context.Context, courseID int64, role string, r io.Reader, actorID int64) (int64, []RowError, error) {
	// check course exists and active
	_, err := s.q.GetCourseByID(ctx, courseID)
	if err != nil {
		return 0, nil, ErrCourseNotFound
	}

	parsed, rowErrs := ParseCSV(r, role)
	if len(rowErrs) > 0 {
		return 0, rowErrs, ErrValidation
	}

	if len(parsed) == 0 {
		return 0, nil, ErrNoValidRows
	}

	usernames := make([]string, len(parsed))
	for i, p := range parsed {
		usernames[i] = p.Username
	}

	validUsers, err := s.repo.GetUserIDsByRole(ctx, db.GetUserIDsByRoleParams{
		Column1: usernames,
		Role:    db.UserRole(role),
	})
	if err != nil {
		return 0, nil, err
	}

	validMap := make(map[string]int64)
	for _, u := range validUsers {
		validMap[u.Username] = u.ID
	}

	for _, p := range parsed {
		if _, ok := validMap[p.Username]; !ok {
			rowErrs = append(rowErrs, RowError{
				Row:     p.RowIndex,
				Field:   role + "_id",
				Message: "invalid, not found, or inactive user",
			})
		}
	}

	if len(rowErrs) > 0 {
		return 0, rowErrs, ErrValidation
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)
	repoTx := NewRepository(qtx)

	var added int64
	for _, p := range parsed {
		uid := validMap[p.Username]
		if role == "student" {
			n, err := repoTx.EnrollStudent(ctx, db.EnrollStudentParams{
				CourseID:  courseID,
				StudentID: uid,
			})
			if err != nil {
				return 0, nil, err
			}
			added += n
		} else {
			n, err := repoTx.AssignLecturer(ctx, db.AssignLecturerParams{
				CourseID:   courseID,
				LecturerID: uid,
			})
			if err != nil {
				return 0, nil, err
			}
			added += n
		}
	}

	if added > 0 {
		action := auditlogs.ENROLL_IMPORT
		if role == "lecturer" {
			action = auditlogs.LECTURER_IMPORT
		}
		
		err = auditlogs.WriteAudit(ctx, qtx, actorID, action, auditlogs.TargetTypeCourse, &courseID, &added, nil)
		if err != nil {
			return 0, nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, nil, err
	}

	return added, nil, nil
}

func (s *Service) RemoveStudent(ctx context.Context, courseID int64, studentID int64, actorID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	n, err := qtx.RemoveStudent(ctx, db.RemoveStudentParams{
		CourseID:  courseID,
		StudentID: studentID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotEnrolled
	}

	if err := auditlogs.WriteAudit(ctx, qtx, actorID, auditlogs.STUDENT_REMOVED_FROM_COURSE, auditlogs.TargetTypeCourse, &courseID, nil, nil); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Service) UnassignLecturer(ctx context.Context, courseID int64, lecturerID int64, actorID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	n, err := qtx.UnassignLecturer(ctx, db.UnassignLecturerParams{
		CourseID:   courseID,
		LecturerID: lecturerID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotEnrolled
	}

	if err := auditlogs.WriteAudit(ctx, qtx, actorID, auditlogs.LECTURER_UNASSIGNED_FROM_COURSE, auditlogs.TargetTypeCourse, &courseID, nil, nil); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
