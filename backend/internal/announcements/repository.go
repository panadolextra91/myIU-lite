package announcements

import (
	"context"

	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

func (r *Repository) ListCourseAnnouncements(ctx context.Context, courseID int64) ([]db.Announcement, error) {
	return r.q.ListCourseAnnouncements(ctx, courseID)
}

func (r *Repository) GetAnnouncementByID(ctx context.Context, id int64) (db.Announcement, error) {
	return r.q.GetAnnouncementByID(ctx, id)
}

func (r *Repository) ListAnnouncementsForStudent(ctx context.Context, params db.ListAnnouncementsForStudentParams) ([]db.Announcement, error) {
	return r.q.ListAnnouncementsForStudent(ctx, params)
}

func (r *Repository) ListAnnouncementRecipients(ctx context.Context, announcementID int64) ([]int64, error) {
	return r.q.ListAnnouncementRecipients(ctx, announcementID)
}
