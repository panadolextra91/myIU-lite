package courses

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func parseDate(d string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", d); err == nil {
		return t, nil
	}
	if t, err := time.Parse("02/01/2006", d); err == nil {
		return t, nil
	}
	return time.Time{}, ErrInvalidDateFormat
}

func (s *Service) CreateCourse(ctx context.Context, code, name, term, startDateStr, endDateStr string, actorID int64) (db.Course, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	term = strings.TrimSpace(term)

	if code == "" || name == "" || term == "" {
		return db.Course{}, ErrRequiredFields
	}

	startDate, err := parseDate(startDateStr)
	if err != nil {
		return db.Course{}, err
	}

	endDate, err := parseDate(endDateStr)
	if err != nil {
		return db.Course{}, err
	}

	if endDate.Before(startDate) {
		return db.Course{}, ErrInvalidDates
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Course{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	course, err := qtx.CreateCourse(ctx, db.CreateCourseParams{
		Code:      code,
		Name:      name,
		Term:      term,
		StartDate: pgtype.Date{Time: startDate, Valid: true},
		EndDate:   pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		return db.Course{}, err
	}

	if err := auditlogs.WriteAudit(ctx, qtx, actorID, auditlogs.COURSE_CREATE, auditlogs.TargetTypeCourse, &course.ID, nil, nil); err != nil {
		return db.Course{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Course{}, err
	}

	return course, nil
}

func (s *Service) GetCourse(ctx context.Context, id int64) (db.Course, error) {
	course, err := s.repo.GetCourseByID(ctx, id)
	if err != nil {
		return db.Course{}, ErrCourseNotFound
	}
	return course, nil
}

func (s *Service) ListCourses(ctx context.Context, term *string, search *string, limit, offset int32) ([]db.Course, int64, error) {
	arg := db.ListCoursesParams{Limit: limit, Offset: offset}
	cArg := db.CountCoursesParams{}
	if term != nil && *term != "" {
		arg.Term = pgtype.Text{String: *term, Valid: true}
		cArg.Term = pgtype.Text{String: *term, Valid: true}
	}
	if search != nil && *search != "" {
		arg.Search = pgtype.Text{String: *search, Valid: true}
		cArg.Search = pgtype.Text{String: *search, Valid: true}
	}

	courses, err := s.repo.ListCourses(ctx, arg)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repo.CountCourses(ctx, cArg)
	if err != nil {
		return nil, 0, err
	}

	return courses, count, nil
}

func (s *Service) UpdateCourse(ctx context.Context, id int64, code, name, term, startDateStr, endDateStr string, actorID int64) (db.Course, error) {
	code = strings.TrimSpace(code)
	name = strings.TrimSpace(name)
	term = strings.TrimSpace(term)

	if code == "" || name == "" || term == "" {
		return db.Course{}, ErrRequiredFields
	}

	startDate, err := parseDate(startDateStr)
	if err != nil {
		return db.Course{}, err
	}

	endDate, err := parseDate(endDateStr)
	if err != nil {
		return db.Course{}, err
	}

	if endDate.Before(startDate) {
		return db.Course{}, ErrInvalidDates
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Course{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	course, err := qtx.UpdateCourse(ctx, db.UpdateCourseParams{
		ID:        id,
		Code:      code,
		Name:      name,
		Term:      term,
		StartDate: pgtype.Date{Time: startDate, Valid: true},
		EndDate:   pgtype.Date{Time: endDate, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Course{}, ErrCourseNotFound
		}
		return db.Course{}, err
	}

	if err := auditlogs.WriteAudit(ctx, qtx, actorID, auditlogs.COURSE_UPDATE, auditlogs.TargetTypeCourse, &id, nil, nil); err != nil {
		return db.Course{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Course{}, err
	}

	return course, nil
}

func (s *Service) SoftDeleteCourse(ctx context.Context, id int64, actorID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	_, err = qtx.GetCourseByID(ctx, id)
	if err != nil {
		return ErrCourseNotFound
	}

	err = qtx.SoftDeleteCourse(ctx, id)
	if err != nil {
		return err
	}

	if err := auditlogs.WriteAudit(ctx, qtx, actorID, auditlogs.COURSE_DELETE, auditlogs.TargetTypeCourse, &id, nil, nil); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *Service) ListCourseStudents(ctx context.Context, id int64) ([]db.ListCourseStudentsRow, error) {
	_, err := s.repo.GetCourseByID(ctx, id)
	if err != nil {
		return nil, ErrCourseNotFound
	}
	return s.repo.ListCourseStudents(ctx, id)
}

func (s *Service) ListCourseLecturers(ctx context.Context, id int64) ([]db.ListCourseLecturersRow, error) {
	_, err := s.repo.GetCourseByID(ctx, id)
	if err != nil {
		return nil, ErrCourseNotFound
	}
	return s.repo.ListCourseLecturers(ctx, id)
}
