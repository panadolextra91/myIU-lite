package courses

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

func (r *Repository) CreateCourse(ctx context.Context, arg db.CreateCourseParams) (db.Course, error) {
	return r.q.CreateCourse(ctx, arg)
}

func (r *Repository) GetCourseByID(ctx context.Context, id int64) (db.Course, error) {
	return r.q.GetCourseByID(ctx, id)
}

func (r *Repository) ListCourses(ctx context.Context, arg db.ListCoursesParams) ([]db.Course, error) {
	return r.q.ListCourses(ctx, arg)
}

func (r *Repository) CountCourses(ctx context.Context, arg db.CountCoursesParams) (int64, error) {
	return r.q.CountCourses(ctx, arg)
}

func (r *Repository) UpdateCourse(ctx context.Context, arg db.UpdateCourseParams) (db.Course, error) {
	return r.q.UpdateCourse(ctx, arg)
}

func (r *Repository) SoftDeleteCourse(ctx context.Context, id int64) error {
	return r.q.SoftDeleteCourse(ctx, id)
}

func (r *Repository) ListCourseStudents(ctx context.Context, courseID int64) ([]db.ListCourseStudentsRow, error) {
	return r.q.ListCourseStudents(ctx, courseID)
}

func (r *Repository) ListCourseLecturers(ctx context.Context, courseID int64) ([]db.ListCourseLecturersRow, error) {
	return r.q.ListCourseLecturers(ctx, courseID)
}
