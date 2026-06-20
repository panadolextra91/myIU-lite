package quizzes

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

func (r *Repository) CreateQuiz(ctx context.Context, arg db.CreateQuizParams) (db.Quiz, error) {
	return r.q.CreateQuiz(ctx, arg)
}

func (r *Repository) GetQuizByID(ctx context.Context, id int64) (db.Quiz, error) {
	return r.q.GetQuizByID(ctx, id)
}

func (r *Repository) ListCourseQuizzes(ctx context.Context, courseID int64) ([]db.Quiz, error) {
	return r.q.ListCourseQuizzes(ctx, courseID)
}

func (r *Repository) InsertQuestion(ctx context.Context, arg db.InsertQuestionParams) (db.QuizQuestion, error) {
	return r.q.InsertQuestion(ctx, arg)
}

func (r *Repository) InsertOption(ctx context.Context, arg db.InsertOptionParams) (db.QuizQuestionOption, error) {
	return r.q.InsertOption(ctx, arg)
}

func (r *Repository) CountQuizQuestions(ctx context.Context, quizID int64) (int64, error) {
	return r.q.CountQuizQuestions(ctx, quizID)
}

func (r *Repository) ListQuestionsForQuiz(ctx context.Context, quizID int64) ([]db.QuizQuestion, error) {
	return r.q.ListQuestionsForQuiz(ctx, quizID)
}

func (r *Repository) ListOptionsForQuestion(ctx context.Context, questionID int64) ([]db.QuizQuestionOption, error) {
	return r.q.ListOptionsForQuestion(ctx, questionID)
}
