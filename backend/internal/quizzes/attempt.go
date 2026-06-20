package quizzes

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/authz"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
)

const (
	StatusInProgress    = "IN_PROGRESS"
	StatusSubmitted     = "SUBMITTED"
	StatusAutoSubmitted = "AUTO_SUBMITTED"
)

func (s *Service) StartAttempt(ctx context.Context, courseID, quizID, studentID int64) (*StudentQuizAttemptView, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, studentID, db.UserRoleStudent); err != nil {
		return nil, err
	}
	q, err := authz.AssertQuizInCourse(ctx, s.pool, quizID, courseID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if q.OpenAt.Valid && now.Before(q.OpenAt.Time) {
		return nil, errors.New("quiz is not open yet")
	}
	if q.CloseAt.Valid && now.After(q.CloseAt.Time) {
		return nil, errors.New("quiz is closed")
	}

	attempt, err := s.q.GetInProgressAttempt(ctx, db.GetInProgressAttemptParams{
		QuizID:    quizID,
		StudentID: studentID,
	})
	if err == nil {
		return s.buildAttemptView(ctx, attempt, q)
	}

	count, err := s.q.CountAttempts(ctx, db.CountAttemptsParams{
		QuizID:    quizID,
		StudentID: studentID,
	})
	if err != nil {
		return nil, err
	}

	if q.RetakeCount.Valid && int32(count) >= q.RetakeCount.Int32 {
		return nil, errors.New("retake limit reached")
	}

	attempt, err = s.q.StartAttempt(ctx, db.StartAttemptParams{
		QuizID:        quizID,
		StudentID:     studentID,
		AttemptNumber: pgtype.Int4{Int32: int32(count) + 1, Valid: true},
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			attempt, err = s.q.GetInProgressAttempt(ctx, db.GetInProgressAttemptParams{
				QuizID:    quizID,
				StudentID: studentID,
			})
			if err == nil {
				return s.buildAttemptView(ctx, attempt, q)
			}
		}
		return nil, err
	}

	err = s.seedAttempt(ctx, attempt, q)
	if err != nil {
		return nil, err
	}

	return s.buildAttemptView(ctx, attempt, q)
}

func (s *Service) seedAttempt(ctx context.Context, attempt db.QuizAttempt, q db.Quiz) error {
	questions, err := s.q.ListQuestionsForQuiz(ctx, q.ID)
	if err != nil {
		return err
	}

	if q.MaxQuestions.Valid && len(questions) < int(q.MaxQuestions.Int32) {
		return errors.New("quiz does not have enough authored questions")
	}

	r := rand.New(rand.NewPCG(uint64(attempt.ID), uint64(q.ID)))

	if q.Shuffle.Valid && q.Shuffle.Bool {
		r.Shuffle(len(questions), func(i, j int) {
			questions[i], questions[j] = questions[j], questions[i]
		})
	}

	m := len(questions)
	if q.MaxQuestions.Valid && int(q.MaxQuestions.Int32) < m {
		m = int(q.MaxQuestions.Int32)
	}

	questions = questions[:m]

	for _, qq := range questions {
		err = s.q.InsertAttemptAnswer(ctx, db.InsertAttemptAnswerParams{
			AttemptID:         attempt.ID,
			QuestionID:        qq.ID,
			SelectedOptionIds: []int64{},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) GetAttempt(ctx context.Context, courseID, quizID, attemptID, studentID int64) (*StudentQuizAttemptView, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, studentID, db.UserRoleStudent); err != nil {
		return nil, err
	}
	q, err := authz.AssertQuizInCourse(ctx, s.pool, quizID, courseID)
	if err != nil {
		return nil, err
	}

	attempt, err := s.q.GetAttemptByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.StudentID != studentID || attempt.QuizID != quizID {
		return nil, errors.New("forbidden")
	}

	now := time.Now()
	if attempt.Status == StatusInProgress && q.CloseAt.Valid && now.After(q.CloseAt.Time) {
		_, err = s.gradeAndSubmit(ctx, attempt, q, true)
		if err != nil {
			return nil, err
		}
		attempt, err = s.q.GetAttemptByID(ctx, attemptID)
		if err != nil {
			return nil, err
		}
	}

	return s.buildAttemptView(ctx, attempt, q)
}

func (s *Service) buildAttemptView(ctx context.Context, attempt db.QuizAttempt, q db.Quiz) (*StudentQuizAttemptView, error) {
	answers, err := s.q.ListAttemptAnswers(ctx, attempt.ID)
	if err != nil {
		return nil, err
	}

	var questionsView []StudentQuestionView
	selectedOptions := make(map[int64][]int64)

	r := rand.New(rand.NewPCG(uint64(attempt.ID), uint64(q.ID)))

	allQ, err := s.q.ListQuestionsForQuiz(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	qMap := make(map[int64]db.QuizQuestion)
	for _, qq := range allQ {
		qMap[qq.ID] = qq
	}

	allOptions, err := s.q.ListOptionsForQuiz(ctx, q.ID)
	if err != nil {
		return nil, err
	}
	optsMap := make(map[int64][]db.QuizQuestionOption)
	for _, o := range allOptions {
		optsMap[o.QuestionID] = append(optsMap[o.QuestionID], o)
	}

	for _, ans := range answers {
		qq, ok := qMap[ans.QuestionID]
		if !ok {
			continue
		}

		opts := optsMap[qq.ID]

		if q.Shuffle.Valid && q.Shuffle.Bool {
			r.Shuffle(len(opts), func(i, j int) {
				opts[i], opts[j] = opts[j], opts[i]
			})
		}

		var optsView []StudentOptionView
		for _, o := range opts {
			optsView = append(optsView, StudentOptionView{ID: o.ID, Text: o.Text.String})
		}

		questionsView = append(questionsView, StudentQuestionView{
			ID:           qq.ID,
			Prompt:       qq.Prompt.String,
			QuestionType: qq.QuestionType,
			Options:      optsView,
		})

		selectedOptions[qq.ID] = ans.SelectedOptionIds
	}

	view := &StudentQuizAttemptView{
		ID:              attempt.ID,
		QuizID:          attempt.QuizID,
		AttemptNumber:   attempt.AttemptNumber.Int32,
		Status:          attempt.Status,
		StartedAt:       attempt.StartedAt.Time,
		Questions:       questionsView,
		SelectedOptions: selectedOptions,
	}

	if attempt.Score.Valid {
		v := s.numericToFloat(attempt.Score)
		view.Score = &v
	}
	if attempt.SubmittedAt.Valid {
		view.SubmittedAt = &attempt.SubmittedAt.Time
	}

	now := time.Now()
	isClosed := q.CloseAt.Valid && now.After(q.CloseAt.Time)
	isTerminal := attempt.Status != StatusInProgress

	if isTerminal && isClosed {
		correctOptions := make(map[int64][]int64)
		for _, qq := range questionsView {
			opts := optsMap[qq.ID]
			var corrects []int64
			for _, o := range opts {
				if o.IsCorrect {
					corrects = append(corrects, o.ID)
				}
			}
			correctOptions[qq.ID] = corrects
		}
		view.CorrectOptions = correctOptions
	}

	return view, nil
}

func (s *Service) numericToFloat(num pgtype.Numeric) float64 {
	f, _ := num.Float64Value()
	return f.Float64
}

func (s *Service) floatToNumeric(f float64) (pgtype.Numeric, error) {
	var num pgtype.Numeric
	err := num.Scan(fmt.Sprintf("%f", f))
	return num, err
}

func (s *Service) SubmitAttempt(ctx context.Context, courseID, quizID, attemptID, studentID int64, req SubmitAttemptRequest) (*SubmitAttemptResponse, error) {
	if err := authz.AssertCourseMember(ctx, s.pool, courseID, studentID, db.UserRoleStudent); err != nil {
		return nil, err
	}
	q, err := authz.AssertQuizInCourse(ctx, s.pool, quizID, courseID)
	if err != nil {
		return nil, err
	}

	attempt, err := s.q.GetAttemptByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.StudentID != studentID || attempt.QuizID != quizID {
		return nil, errors.New("forbidden")
	}

	if attempt.Status != StatusInProgress {
		maxScore, _ := s.q.GetMaxScore(ctx, db.GetMaxScoreParams{QuizID: q.ID, StudentID: studentID})
		return &SubmitAttemptResponse{
			Score:         s.numericToFloat(attempt.Score),
			OfficialScore: s.numericToFloat(maxScore),
			Status:        attempt.Status,
		}, nil
	}

	answers, err := s.q.ListAttemptAnswers(ctx, attempt.ID)
	if err != nil {
		return nil, err
	}
	validQIDs := make(map[int64]bool)
	for _, a := range answers {
		validQIDs[a.QuestionID] = true
	}

	for qID, opts := range req.Answers {
		if !validQIDs[qID] {
			return nil, errors.New("invalid question ID submitted for this attempt")
		}
		err = s.q.UpdateAttemptAnswer(ctx, db.UpdateAttemptAnswerParams{
			AttemptID:         attempt.ID,
			QuestionID:        qID,
			SelectedOptionIds: opts,
		})
		if err != nil {
			return nil, err
		}
	}

	now := time.Now()
	isAuto := q.CloseAt.Valid && now.After(q.CloseAt.Time)
	score, err := s.gradeAndSubmit(ctx, attempt, q, isAuto)
	if err != nil {
		return nil, err
	}

	maxScore, _ := s.q.GetMaxScore(ctx, db.GetMaxScoreParams{QuizID: q.ID, StudentID: studentID})

	status := StatusSubmitted
	if isAuto {
		status = StatusAutoSubmitted
	}

	return &SubmitAttemptResponse{
		Score:         score,
		OfficialScore: s.numericToFloat(maxScore),
		Status:        status,
	}, nil
}

func (s *Service) gradeAndSubmit(ctx context.Context, attempt db.QuizAttempt, q db.Quiz, isAuto bool) (float64, error) {
	answers, err := s.q.ListAttemptAnswers(ctx, attempt.ID)
	if err != nil {
		return 0, err
	}

	allOptions, err := s.q.ListOptionsForQuiz(ctx, q.ID)
	if err != nil {
		return 0, err
	}
	optsMap := make(map[int64][]db.QuizQuestionOption)
	for _, o := range allOptions {
		optsMap[o.QuestionID] = append(optsMap[o.QuestionID], o)
	}

	var totalCorrect float64
	for _, ans := range answers {
		opts := optsMap[ans.QuestionID]

		var correctIds []int64
		for _, o := range opts {
			if o.IsCorrect {
				correctIds = append(correctIds, o.ID)
			}
		}

		if isExactMatch(ans.SelectedOptionIds, correctIds) {
			totalCorrect++
		}
	}

	score := float64(0)
	if len(answers) > 0 {
		score = (totalCorrect / float64(len(answers))) * s.numericToFloat(q.MaxGrade)
	}

	pgScore, err := s.floatToNumeric(score)
	if err != nil {
		return 0, fmt.Errorf("failed to convert score: %w", err)
	}

	var rowsAffected int64
	if isAuto {
		rowsAffected, err = s.q.MarkAttemptAutoSubmitted(ctx, db.MarkAttemptAutoSubmittedParams{
			ID:          attempt.ID,
			Score:       pgScore,
			SubmittedAt: q.CloseAt,
		})
	} else {
		rowsAffected, err = s.q.MarkAttemptSubmitted(ctx, db.MarkAttemptSubmittedParams{
			ID:    attempt.ID,
			Score: pgScore,
		})
	}

	if err != nil {
		return 0, err
	}

	if rowsAffected == 0 {
		att, err := s.q.GetAttemptByID(ctx, attempt.ID)
		if err != nil {
			return 0, err
		}
		return s.numericToFloat(att.Score), nil
	}

	return score, nil
}

func isExactMatch(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[int64]bool)
	for _, id := range a {
		m[id] = true
	}
	for _, id := range b {
		if !m[id] {
			return false
		}
	}
	return true
}

func (s *Service) OfficialScore(ctx context.Context, quizID, studentID int64) (float64, error) {
	maxScore, err := s.q.GetMaxScore(ctx, db.GetMaxScoreParams{QuizID: quizID, StudentID: studentID})
	if err != nil {
		return 0, err
	}
	return s.numericToFloat(maxScore), nil
}
