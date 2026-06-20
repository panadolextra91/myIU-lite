package enrollments

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

func (r *Repository) EnrollStudent(ctx context.Context, arg db.EnrollStudentParams) (int64, error) {
	return r.q.EnrollStudent(ctx, arg)
}

func (r *Repository) AssignLecturer(ctx context.Context, arg db.AssignLecturerParams) (int64, error) {
	return r.q.AssignLecturer(ctx, arg)
}

func (r *Repository) RemoveStudent(ctx context.Context, arg db.RemoveStudentParams) (int64, error) {
	return r.q.RemoveStudent(ctx, arg)
}

func (r *Repository) UnassignLecturer(ctx context.Context, arg db.UnassignLecturerParams) (int64, error) {
	return r.q.UnassignLecturer(ctx, arg)
}

func (r *Repository) GetUserIDsByRole(ctx context.Context, arg db.GetUserIDsByRoleParams) ([]db.GetUserIDsByRoleRow, error) {
	return r.q.GetUserIDsByRole(ctx, arg)
}
