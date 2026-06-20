package requests

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

func (r *Repository) InsertRequest(ctx context.Context, arg db.InsertRequestParams) (db.Request, error) {
	return r.q.InsertRequest(ctx, arg)
}

func (r *Repository) ReplyRequest(ctx context.Context, arg db.ReplyRequestParams) (db.Request, error) {
	return r.q.ReplyRequest(ctx, arg)
}

func (r *Repository) ListLecturerRequests(ctx context.Context, targetedLecturerID int64) ([]db.Request, error) {
	return r.q.ListLecturerRequests(ctx, targetedLecturerID)
}

func (r *Repository) ListStudentRequests(ctx context.Context, studentID int64) ([]db.Request, error) {
	return r.q.ListStudentRequests(ctx, studentID)
}

func (r *Repository) GetRequestByID(ctx context.Context, id int64) (db.Request, error) {
	return r.q.GetRequestByID(ctx, id)
}
