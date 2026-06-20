package assignments

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/authz"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/cloudinary"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrWindowClosed = errors.New("window_closed")
	ErrNotFound     = errors.New("not_found")
)

type Service struct {
	pool *pgxpool.Pool
	repo *Repository
	q    *db.Queries
	cld  *cloudinary.Client
}

func NewService(pool *pgxpool.Pool, repo *Repository, cld *cloudinary.Client) *Service {
	return &Service{pool: pool, repo: repo, q: db.New(pool), cld: cld}
}

func (s *Service) CreateAssignment(ctx context.Context, courseID int64, req CreateAssignmentRequest, lecturerID int64) (db.Assignment, error) {
	lecturers, err := s.q.ListCourseLecturers(ctx, courseID)
	if err != nil {
		return db.Assignment{}, err
	}
	isLecturer := false
	for _, l := range lecturers {
		if l.LecturerID == lecturerID {
			isLecturer = true
			break
		}
	}
	if !isLecturer {
		return db.Assignment{}, ErrForbidden
	}

	var threshold pgtype.Int4
	if req.LateThresholdDays != nil {
		threshold = pgtype.Int4{Int32: *req.LateThresholdDays, Valid: true}
	}

	arg := db.CreateAssignmentParams{
		CourseID:          courseID,
		Title:             req.Title,
		Description:       pgtype.Text{String: req.Description, Valid: req.Description != ""},
		Deadline:          pgtype.Timestamptz{Time: req.Deadline, Valid: true},
		AcceptLate:        req.AcceptLate,
		LateThresholdDays: threshold,
		CreatedBy:         lecturerID,
	}
	return s.repo.CreateAssignment(ctx, arg)
}

func (s *Service) ListCourseAssignments(ctx context.Context, courseID, userID int64, role string) ([]db.Assignment, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, userID, db.UserRole(role)); err != nil {
		return nil, err
	}
	return s.repo.ListCourseAssignments(ctx, courseID)
}

func (s *Service) Submit(ctx context.Context, courseID, assignmentID, studentID int64, fileReader io.Reader, filename string) (db.Submission, time.Time, error) {
	students, err := s.q.ListCourseStudents(ctx, courseID)
	if err != nil {
		return db.Submission{}, time.Time{}, err
	}
	isStudent := false
	for _, st := range students {
		if st.StudentID == studentID {
			isStudent = true
			break
		}
	}
	if !isStudent {
		return db.Submission{}, time.Time{}, ErrForbidden
	}

	assignment, err := s.repo.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		return db.Submission{}, time.Time{}, err
	}

	if assignment.CourseID != courseID {
		return db.Submission{}, time.Time{}, ErrForbidden
	}

	now := time.Now()
	deadline := assignment.Deadline.Time
	var isLate bool

	if now.After(deadline) {
		if !assignment.AcceptLate {
			return db.Submission{}, time.Time{}, ErrWindowClosed
		}
		if assignment.LateThresholdDays.Valid {
			maxLate := deadline.AddDate(0, 0, int(assignment.LateThresholdDays.Int32))
			if now.After(maxLate) {
				return db.Submission{}, time.Time{}, ErrWindowClosed
			}
		}
		isLate = true
	}

	folder := fmt.Sprintf("course_%d_assignment_%d", courseID, assignmentID)
	publicID, format, err := s.cld.Upload(ctx, fileReader, folder)
	if err != nil {
		return db.Submission{}, time.Time{}, err
	}

	arg := db.InsertSubmissionVersionParams{
		AssignmentID:       assignmentID,
		StudentID:          studentID,
		CloudinaryPublicID: publicID,
		CloudinaryFormat:   format,
		OriginalFilename:   filename,
		IsLate:             isLate,
	}

	sub, err := s.repo.InsertSubmissionVersion(ctx, arg)
	return sub, deadline, err
}

func (s *Service) DownloadURL(ctx context.Context, courseID, assignmentID, submissionID, userID int64, role string) (string, error) {
	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return "", err
	}

	assignment, err := s.repo.GetAssignmentByID(ctx, assignmentID)
	if err != nil || assignment.CourseID != courseID || sub.AssignmentID != assignmentID {
		return "", ErrNotFound
	}

	if role == "student" {
		if sub.StudentID != userID {
			return "", ErrForbidden
		}
	} else if role == "lecturer" {
		lecturers, err := s.q.ListCourseLecturers(ctx, courseID)
		if err != nil {
			return "", err
		}
		isLecturer := false
		for _, l := range lecturers {
			if l.LecturerID == userID {
				isLecturer = true
				break
			}
		}
		if !isLecturer {
			return "", ErrForbidden
		}
	} else {
		return "", ErrForbidden
	}

	return s.cld.SignedDownloadURL(sub.CloudinaryPublicID, sub.CloudinaryFormat)
}

func (s *Service) GradeSubmission(ctx context.Context, courseID, assignmentID, submissionID int64, score float64, feedback string, lecturerID int64) error {
	lecturers, err := s.q.ListCourseLecturers(ctx, courseID)
	if err != nil {
		return err
	}
	isLecturer := false
	for _, l := range lecturers {
		if l.LecturerID == lecturerID {
			isLecturer = true
			break
		}
	}
	if !isLecturer {
		return ErrForbidden
	}

	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return err
	}

	if sub.CourseID != courseID || sub.AssignmentID != assignmentID {
		return ErrNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.q.WithTx(tx)

	var fdbk pgtype.Text
	if feedback != "" {
		fdbk = pgtype.Text{String: feedback, Valid: true}
	}
	var num pgtype.Numeric
	_ = num.Scan(fmt.Sprintf("%f", score))
	
	err = qtx.UpsertSubmissionGrade(ctx, db.UpsertSubmissionGradeParams{
		ID:       submissionID,
		Score:    num,
		Feedback: fdbk,
		GradedBy: pgtype.Int8{Int64: lecturerID, Valid: true},
	})
	if err != nil {
		return err
	}

	scoreStr := fmt.Sprintf("%.2f", score)
	title := "Assignment Graded"
	body := fmt.Sprintf("Your assignment %q has been graded. Score: %s.", sub.AssignmentTitle, scoreStr)
	link := fmt.Sprintf("/courses/%d/assignments/%d", courseID, assignmentID)

	_, err = qtx.InsertNotification(ctx, db.InsertNotificationParams{
		RecipientID:  sub.StudentID,
		Type:         "ASSIGNMENT_GRADED",
		Title:        title,
		Body:         body,
		ResourceType: pgtype.Text{String: "assignment", Valid: true},
		ResourceID:   pgtype.Int8{Int64: assignmentID, Valid: true},
		Link:         pgtype.Text{String: link, Valid: true},
	})
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

