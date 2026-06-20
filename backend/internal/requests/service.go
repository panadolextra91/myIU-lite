package requests

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/authz"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

type Service struct {
	repo *Repository
	pool *pgxpool.Pool
}

func NewService(repo *Repository, pool *pgxpool.Pool) *Service {
	return &Service{repo: repo, pool: pool}
}

func (s *Service) CreateRequest(ctx context.Context, courseID, studentID int64, req CreateRequestRequest) (db.Request, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, studentID, db.UserRoleStudent); err != nil {
		return db.Request{}, ErrForbidden
	}

	q := db.New(s.pool)

	// Validate targeted lecturer
	lecturers, err := q.ListCourseLecturers(ctx, courseID)
	if err != nil {
		return db.Request{}, fmt.Errorf("list lecturers: %w", err)
	}
	found := false
	for _, l := range lecturers {
		if l.LecturerID == req.TargetedLecturerID {
			found = true
			break
		}
	}
	if !found {
		return db.Request{}, ErrValidation
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Request{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := q.WithTx(tx)

	r, err := qtx.InsertRequest(ctx, db.InsertRequestParams{
		CourseID:           courseID,
		StudentID:          studentID,
		TargetedLecturerID: req.TargetedLecturerID,
		Type:               req.Type,
		Title:              req.Title,
		Body:               req.Body,
	})
	if err != nil {
		return db.Request{}, fmt.Errorf("insert request: %w", err)
	}

	link := "/lecturer/requests"
	_, err = qtx.InsertNotification(ctx, db.InsertNotificationParams{
		RecipientID:  req.TargetedLecturerID,
		Type:         "REQUEST_CREATED",
		Title:        "New request",
		Body:         fmt.Sprintf("%s: %s", req.Type, req.Title),
		ResourceType: pgtype.Text{String: "request", Valid: true},
		ResourceID:   pgtype.Int8{Int64: r.ID, Valid: true},
		Link:         pgtype.Text{String: link, Valid: true},
	})
	if err != nil {
		return db.Request{}, fmt.Errorf("insert notification: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Request{}, fmt.Errorf("commit tx: %w", err)
	}

	return r, nil
}

func (s *Service) ReplyRequest(ctx context.Context, requestID, lecturerID int64, req ReplyRequestRequest) (db.Request, error) {
	q := db.New(s.pool)
	
	// Check visibility and targeted
	existing, err := q.GetRequestByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Request{}, ErrNotFound
		}
		return db.Request{}, fmt.Errorf("get request: %w", err)
	}

	if existing.TargetedLecturerID != lecturerID {
		return db.Request{}, ErrNotTargeted
	}

	if existing.Status != "PENDING" {
		return db.Request{}, ErrAlreadyClosed
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Request{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := q.WithTx(tx)

	noteParam := pgtype.Text{String: req.Note, Valid: req.Note != ""}
	r, err := qtx.ReplyRequest(ctx, db.ReplyRequestParams{
		ID:                 requestID,
		TargetedLecturerID: lecturerID,
		Status:             req.Decision,
		ReplyNote:          noteParam,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Request{}, ErrAlreadyClosed
		}
		return db.Request{}, fmt.Errorf("reply request: %w", err)
	}

	body := req.Decision
	if req.Note != "" {
		body += "\n" + req.Note
	}
	link := "/student/requests"

	_, err = qtx.InsertNotification(ctx, db.InsertNotificationParams{
		RecipientID:  existing.StudentID,
		Type:         "REQUEST_REPLIED",
		Title:        fmt.Sprintf("Request %s", req.Decision),
		Body:         body,
		ResourceType: pgtype.Text{String: "request", Valid: true},
		ResourceID:   pgtype.Int8{Int64: r.ID, Valid: true},
		Link:         pgtype.Text{String: link, Valid: true},
	})
	if err != nil {
		return db.Request{}, fmt.Errorf("insert notification: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Request{}, fmt.Errorf("commit tx: %w", err)
	}

	return r, nil
}

func (s *Service) ListForLecturer(ctx context.Context, lecturerID int64) ([]db.Request, error) {
	return s.repo.ListLecturerRequests(ctx, lecturerID)
}

func (s *Service) ListForStudent(ctx context.Context, studentID int64) ([]db.Request, error) {
	return s.repo.ListStudentRequests(ctx, studentID)
}

func (s *Service) GetByID(ctx context.Context, requestID, userID int64, role db.UserRole) (db.Request, error) {
	r, err := s.repo.GetRequestByID(ctx, requestID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Request{}, ErrNotFound
		}
		return db.Request{}, err
	}

	if role == db.UserRoleStudent && r.StudentID != userID {
		return db.Request{}, ErrNotFound
	}
	if role == db.UserRoleLecturer && r.TargetedLecturerID != userID {
		return db.Request{}, ErrNotFound
	}

	return r, nil
}
