package users

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

type Repository struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) *Repository {
	return &Repository{q: q}
}

func (r *Repository) WithTx(tx pgx.Tx) *db.Queries {
	return r.q.WithTx(tx)
}

func (r *Repository) CreateUser(ctx context.Context, arg db.CreateUserParams) (int64, error) {
	return r.q.CreateUser(ctx, arg)
}

func (r *Repository) GetUserByID(ctx context.Context, id int64) (db.User, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *Repository) ResetUserPassword(ctx context.Context, arg db.ResetUserPasswordParams) error {
	return r.q.ResetUserPassword(ctx, arg)
}

func (r *Repository) ListUsers(ctx context.Context, arg db.ListUsersParams) ([]db.User, error) {
	return r.q.ListUsers(ctx, arg)
}

func (r *Repository) CountUsers(ctx context.Context, arg db.CountUsersParams) (int64, error) {
	return r.q.CountUsers(ctx, arg)
}

func (r *Repository) GetActiveUsernames(ctx context.Context, usernames []string) ([]string, error) {
	return r.q.GetActiveUsernames(ctx, usernames)
}
