package grades

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

func (r *Repository) CreateGradeScheme(ctx context.Context, arg db.CreateGradeSchemeParams) (db.GradeScheme, error) {
	return r.q.CreateGradeScheme(ctx, arg)
}

func (r *Repository) GetSchemeByCourse(ctx context.Context, courseID int64) (db.GradeScheme, error) {
	return r.q.GetSchemeByCourse(ctx, courseID)
}

func (r *Repository) InsertGradeComponent(ctx context.Context, arg db.InsertGradeComponentParams) (db.GradeComponent, error) {
	return r.q.InsertGradeComponent(ctx, arg)
}

func (r *Repository) ListSchemeComponents(ctx context.Context, courseID int64) ([]db.GradeComponent, error) {
	return r.q.ListSchemeComponents(ctx, courseID)
}

func (r *Repository) UpsertGradeScore(ctx context.Context, arg db.UpsertGradeScoreParams) error {
	return r.q.UpsertGradeScore(ctx, arg)
}

func (r *Repository) ListScoresForStudent(ctx context.Context, arg db.ListScoresForStudentParams) ([]db.GradeScore, error) {
	return r.q.ListScoresForStudent(ctx, arg)
}

func (r *Repository) ComputeQuizAverage(ctx context.Context, arg db.ComputeQuizAverageParams) (db.ComputeQuizAverageRow, error) {
	return r.q.ComputeQuizAverage(ctx, arg)
}

func (r *Repository) ComputeAssignmentAverage(ctx context.Context, arg db.ComputeAssignmentAverageParams) (db.ComputeAssignmentAverageRow, error) {
	return r.q.ComputeAssignmentAverage(ctx, arg)
}

func (r *Repository) CountSchemeScores(ctx context.Context, schemeID int64) (int64, error) {
	return r.q.CountSchemeScores(ctx, schemeID)
}

func (r *Repository) CountSchemePublications(ctx context.Context, schemeID int64) (int64, error) {
	return r.q.CountSchemePublications(ctx, schemeID)
}

func (r *Repository) DeleteSchemeIfEmpty(ctx context.Context, arg db.DeleteSchemeIfEmptyParams) (int64, error) {
	return r.q.DeleteSchemeIfEmpty(ctx, arg)
}

func (r *Repository) DeleteSchemeComponents(ctx context.Context, schemeID int64) error {
	return r.q.DeleteSchemeComponents(ctx, schemeID)
}
