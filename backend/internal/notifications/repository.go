package notifications

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

func (r *Repository) InsertNotification(ctx context.Context, arg db.InsertNotificationParams) (db.Notification, error) {
	return r.q.InsertNotification(ctx, arg)
}

func (r *Repository) ListNotifications(ctx context.Context, arg db.ListNotificationsParams) ([]db.Notification, error) {
	return r.q.ListNotifications(ctx, arg)
}

func (r *Repository) CountUnread(ctx context.Context, recipientID int64) (int64, error) {
	return r.q.CountUnread(ctx, recipientID)
}

func (r *Repository) MarkRead(ctx context.Context, arg db.MarkReadParams) error {
	return r.q.MarkRead(ctx, arg)
}
