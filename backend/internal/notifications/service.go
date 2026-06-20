package notifications

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
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

func (s *Service) ListForUser(ctx context.Context, userID int64, limit, offset int32) ([]db.Notification, error) {
	return s.repo.ListNotifications(ctx, db.ListNotificationsParams{
		RecipientID: userID,
		Limit:       limit,
		Offset:      offset,
	})
}

func (s *Service) UnreadCount(ctx context.Context, userID int64) (int64, error) {
	return s.repo.CountUnread(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, notificationID int64, userID int64) error {
	rows, err := s.repo.MarkRead(ctx, db.MarkReadParams{
		ID:          notificationID,
		RecipientID: userID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotificationNotFound
	}
	return nil
}
