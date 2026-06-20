package assignments

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

func (r *Repository) CreateAssignment(ctx context.Context, arg db.CreateAssignmentParams) (db.Assignment, error) {
	return r.q.CreateAssignment(ctx, arg)
}

func (r *Repository) GetAssignmentByID(ctx context.Context, id int64) (db.Assignment, error) {
	return r.q.GetAssignmentByID(ctx, id)
}

func (r *Repository) ListCourseAssignments(ctx context.Context, courseID int64) ([]db.Assignment, error) {
	return r.q.ListCourseAssignments(ctx, courseID)
}

func (r *Repository) InsertSubmissionVersion(ctx context.Context, arg db.InsertSubmissionVersionParams) (db.Submission, error) {
	return r.q.InsertSubmissionVersion(ctx, arg)
}

func (r *Repository) GetMaxSubmissionVersion(ctx context.Context, arg db.GetMaxSubmissionVersionParams) (int32, error) {
	return r.q.GetMaxSubmissionVersion(ctx, arg)
}

func (r *Repository) GetActiveSubmission(ctx context.Context, arg db.GetActiveSubmissionParams) (db.Submission, error) {
	return r.q.GetActiveSubmission(ctx, arg)
}

func (r *Repository) ListSubmissionVersions(ctx context.Context, arg db.ListSubmissionVersionsParams) ([]db.Submission, error) {
	return r.q.ListSubmissionVersions(ctx, arg)
}

func (r *Repository) GetSubmissionByID(ctx context.Context, id int64) (db.Submission, error) {
	return r.q.GetSubmissionByID(ctx, id)
}
