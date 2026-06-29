package quizzes

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/authz"
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

func (s *Service) isLecturerOfCourse(ctx context.Context, courseID, lecturerID int64) (bool, error) {
	lecturers, err := s.q.ListCourseLecturers(ctx, courseID)
	if err != nil {
		return false, err
	}
	for _, l := range lecturers {
		if l.LecturerID == lecturerID {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) CreateQuiz(ctx context.Context, courseID int64, req CreateQuizRequest, lecturerID int64) (db.Quiz, error) {
	ok, err := s.isLecturerOfCourse(ctx, courseID, lecturerID)
	if err != nil {
		return db.Quiz{}, err
	}
	if !ok {
		return db.Quiz{}, ErrForbidden
	}

	// Block creating a quiz on a soft-deleted (archived) course — GetCourseByID
	// filters `deleted_at IS NULL`. ponytail: the membership query above already
	// confirmed the DB is reachable, so any error here means the course is gone.
	if _, err := s.q.GetCourseByID(ctx, courseID); err != nil {
		return db.Quiz{}, ErrForbidden
	}

	if req.MaxQuestions > req.PoolSize {
		return db.Quiz{}, ErrPoolTooSmall
	}
	if req.OpenAt != nil && req.CloseAt != nil && !req.CloseAt.After(*req.OpenAt) {
		return db.Quiz{}, ErrInvalidDates
	}

	var openAt, closeAt pgtype.Timestamptz
	if req.OpenAt != nil {
		openAt = pgtype.Timestamptz{Time: *req.OpenAt, Valid: true}
	}
	if req.CloseAt != nil {
		closeAt = pgtype.Timestamptz{Time: *req.CloseAt, Valid: true}
	}

	var num pgtype.Numeric
	_ = num.Scan(fmt.Sprintf("%f", req.MaxGrade))

	return s.repo.CreateQuiz(ctx, db.CreateQuizParams{
		CourseID:     courseID,
		Title:        req.Title,
		PoolSize:     pgtype.Int4{Int32: req.PoolSize, Valid: true},
		MaxQuestions: pgtype.Int4{Int32: req.MaxQuestions, Valid: true},
		MaxGrade:     num,
		Shuffle:      pgtype.Bool{Bool: req.Shuffle, Valid: true},
		RetakeCount:  pgtype.Int4{Int32: req.RetakeCount, Valid: true},
		OpenAt:       openAt,
		CloseAt:      closeAt,
		CreatedBy:    lecturerID,
	})
}

func (s *Service) ImportQuestionsCSV(ctx context.Context, courseID, quizID int64, r io.Reader, lecturerID int64) error {
	ok, err := s.isLecturerOfCourse(ctx, courseID, lecturerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}

	if _, err := authz.AssertQuizInCourse(ctx, s.pool, quizID, courseID); err != nil {
		return err
	}

	csvReader := csv.NewReader(r)
	records, err := csvReader.ReadAll()
	if err != nil {
		return err
	}

	if len(records) < 2 {
		return nil
	}

	var rowErrs []RowError
	for i, row := range records[1:] {
		if len(row) != 6 {
			rowErrs = append(rowErrs, RowError{Row: i + 2, Field: "row", Message: "must have exactly 6 columns: question,A,B,C,D,correct"})
			continue
		}
		for j := 0; j < 5; j++ {
			if strings.TrimSpace(row[j]) == "" {
				rowErrs = append(rowErrs, RowError{Row: i + 2, Field: "choices", Message: "question and all 4 choices must not be empty"})
				break
			}
		}
		correct := strings.TrimSpace(strings.ToUpper(row[5]))
		if correct != "A" && correct != "B" && correct != "C" && correct != "D" {
			rowErrs = append(rowErrs, RowError{Row: i + 2, Field: "correct", Message: "correct must be A, B, C, or D"})
		}
	}

	if len(rowErrs) > 0 {
		return &ImportError{RowErrors: rowErrs}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	for _, row := range records[1:] {
		q, err := qtx.InsertQuestion(ctx, db.InsertQuestionParams{
			QuizID:       quizID,
			Prompt:       pgtype.Text{String: row[0], Valid: true},
			QuestionType: "single",
		})
		if err != nil {
			return err
		}

		correctLetter := strings.TrimSpace(strings.ToUpper(row[5]))
		for idx, letter := range []string{"A", "B", "C", "D"} {
			isCorrect := (letter == correctLetter)
			_, err = qtx.InsertOption(ctx, db.InsertOptionParams{
				QuestionID: q.ID,
				Text:       pgtype.Text{String: row[idx+1], Valid: true},
				IsCorrect:  isCorrect,
			})
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit(ctx)
}

func (s *Service) AddUIQuestion(ctx context.Context, courseID, quizID int64, req UIQuestionRequest, lecturerID int64) error {
	ok, err := s.isLecturerOfCourse(ctx, courseID, lecturerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}

	if _, err := authz.AssertQuizInCourse(ctx, s.pool, quizID, courseID); err != nil {
		return err
	}

	correctCount := 0
	for _, opt := range req.Options {
		if opt.IsCorrect {
			correctCount++
		}
	}

	if req.QuestionType != "single" && req.QuestionType != "multi" {
		return fmt.Errorf("%w: invalid question type, must be 'single' or 'multi'", ErrInvalidQuestion)
	}
	if req.QuestionType == "single" && correctCount != 1 {
		return fmt.Errorf("%w: single choice must have exactly 1 correct option", ErrInvalidQuestion)
	}
	if req.QuestionType == "multi" && correctCount < 1 {
		return fmt.Errorf("%w: multi choice must have at least 1 correct option", ErrInvalidQuestion)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.q.WithTx(tx)

	q, err := qtx.InsertQuestion(ctx, db.InsertQuestionParams{
		QuizID:       quizID,
		Prompt:       pgtype.Text{String: req.Prompt, Valid: true},
		QuestionType: req.QuestionType,
	})
	if err != nil {
		return err
	}

	for _, opt := range req.Options {
		_, err = qtx.InsertOption(ctx, db.InsertOptionParams{
			QuestionID: q.ID,
			Text:       pgtype.Text{String: opt.Text, Valid: true},
			IsCorrect:  opt.IsCorrect,
		})
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
