package announcements

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

func (s *Service) CreateAnnouncement(ctx context.Context, courseID, authorID int64, req CreateAnnouncementRequest) (db.Announcement, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, authorID, db.UserRoleLecturer); err != nil {
		return db.Announcement{}, ErrForbidden
	}

	q := db.New(s.pool)

	// Resolve recipients
	var targetStudentIDs []int64
	if req.AudienceType == "ALL_STUDENTS" {
		students, err := q.ListCourseStudents(ctx, courseID)
		if err != nil {
			return db.Announcement{}, fmt.Errorf("list students: %w", err)
		}
		for _, st := range students {
			targetStudentIDs = append(targetStudentIDs, st.StudentID)
		}
	} else if req.AudienceType == "SPECIFIC_STUDENTS" {
		if len(req.StudentIDs) == 0 {
			return db.Announcement{}, ErrValidation
		}
		students, err := q.ListCourseStudents(ctx, courseID)
		if err != nil {
			return db.Announcement{}, fmt.Errorf("list students: %w", err)
		}
		enrolledMap := make(map[int64]bool)
		for _, st := range students {
			enrolledMap[st.StudentID] = true
		}
		for _, id := range req.StudentIDs {
			if !enrolledMap[id] {
				return db.Announcement{}, ErrNotEnrolled
			}
			targetStudentIDs = append(targetStudentIDs, id)
		}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return db.Announcement{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := q.WithTx(tx)

	// 1. Insert announcement
	ann, err := qtx.InsertAnnouncement(ctx, db.InsertAnnouncementParams{
		CourseID:     courseID,
		AuthorID:     authorID,
		Title:        req.Title,
		Body:         req.Body,
		AudienceType: req.AudienceType,
	})
	if err != nil {
		return db.Announcement{}, fmt.Errorf("insert announcement: %w", err)
	}

	// 2. Insert specific recipients if needed
	if req.AudienceType == "SPECIFIC_STUDENTS" {
		for _, id := range targetStudentIDs {
			if err := qtx.InsertAnnouncementRecipient(ctx, db.InsertAnnouncementRecipientParams{
				AnnouncementID: ann.ID,
				StudentID:      id,
			}); err != nil {
				return db.Announcement{}, fmt.Errorf("insert recipient: %w", err)
			}
		}
	}

	// 3. Insert notification fan-out
	link := fmt.Sprintf("/courses/%d/announcements/%d", courseID, ann.ID)
	for _, id := range targetStudentIDs {
		_, err = qtx.InsertNotification(ctx, db.InsertNotificationParams{
			RecipientID:  id,
			Type:         "ANNOUNCEMENT",
			Title:        req.Title,
			Body:         req.Body,
			ResourceType: pgtype.Text{String: "announcement", Valid: true},
			ResourceID:   pgtype.Int8{Int64: ann.ID, Valid: true},
			Link:         pgtype.Text{String: link, Valid: true},
		})
		if err != nil {
			return db.Announcement{}, fmt.Errorf("insert notification: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Announcement{}, fmt.Errorf("commit tx: %w", err)
	}

	return ann, nil
}

func (s *Service) ListForCourse(ctx context.Context, courseID, lecturerID int64) ([]db.Announcement, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, lecturerID, db.UserRoleLecturer); err != nil {
		return nil, ErrForbidden
	}
	return s.repo.ListCourseAnnouncements(ctx, courseID)
}

func (s *Service) ListForStudent(ctx context.Context, courseID, studentID int64) ([]db.Announcement, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, studentID, db.UserRoleStudent); err != nil {
		return nil, ErrForbidden
	}
	return s.repo.ListAnnouncementsForStudent(ctx, db.ListAnnouncementsForStudentParams{
		CourseID:  courseID,
		StudentID: studentID,
	})
}

func (s *Service) GetByID(ctx context.Context, courseID, announcementID, userID int64, role db.UserRole) (db.Announcement, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, userID, role); err != nil {
		return db.Announcement{}, ErrForbidden
	}

	ann, err := s.repo.GetAnnouncementByID(ctx, announcementID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Announcement{}, ErrNotFound
		}
		return db.Announcement{}, err
	}
	if ann.CourseID != courseID {
		return db.Announcement{}, ErrNotFound
	}

	// For student, check visibility
	if role == db.UserRoleStudent {
		if ann.AudienceType == "SPECIFIC_STUDENTS" {
			recips, err := s.repo.ListAnnouncementRecipients(ctx, announcementID)
			if err != nil {
				return db.Announcement{}, fmt.Errorf("list recipients: %w", err)
			}
			found := false
			for _, r := range recips {
				if r == userID {
					found = true
					break
				}
			}
			if !found {
				return db.Announcement{}, ErrNotFound
			}
		}
	}

	return ann, nil
}
