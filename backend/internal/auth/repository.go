package auth

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

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (db.User, error) {
	return r.q.GetUserByUsername(ctx, username)
}

func (r *Repository) GetUserByID(ctx context.Context, id int64) (db.User, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *Repository) UpdatePasswordAndStamp(ctx context.Context, id int64, hash string) error {
	return r.q.UpdatePasswordAndStamp(ctx, db.UpdatePasswordAndStampParams{
		PasswordHash: hash,
		ID:           id,
	})
}
