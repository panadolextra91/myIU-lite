package quizzes_test

import (
	"context"
	"os"
	"testing"
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/quizzes"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/authz"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurity_Quizzes(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	defer pool.Close()

	repo := quizzes.NewRepository(db.New(pool))
	service := quizzes.NewService(pool, repo)

	t.Run("Enrollment Authz - Student Not Enrolled", func(t *testing.T) {
		f := testutil.SetupQuizzesFixture(t, ctx, pool)
		// Mock another student not enrolled
		uniqueStr := fmt.Sprintf("other-%d", time.Now().UnixNano())
		q := db.New(pool)
		otherStudent, _ := q.CreateUser(ctx, db.CreateUserParams{
			Username: "S2" + uniqueStr,
			Role: db.UserRoleStudent,
		})
		
		err := authz.AssertCourseMember(ctx, pool, f.CourseID, otherStudent, db.UserRoleStudent)
		assert.ErrorIs(t, err, authz.ErrForbidden)

		_, err = service.StartAttempt(ctx, f.CourseID, f.QuizID, otherStudent)
		assert.ErrorIs(t, err, authz.ErrForbidden)
	})

	t.Run("CSV Reject - Format Validations", func(t *testing.T) {
		f := testutil.SetupQuizzesFixture(t, ctx, pool)
		tests := []struct {
			name     string
			csvData  string
			expected string
		}{
			{
				name:     "Missing cols (3 cols)",
				csvData:  "question,A,B\nQ1,AnsA,AnsB\n",
				expected: "row", // row error
			},
			{
				name:     "Invalid correct option E",
				csvData:  "question,A,B,C,D,correct\nQ1,A,B,C,D,E\n",
				expected: "correct",
			},
			{
				name:     "Empty option",
				csvData:  "question,A,B,C,D,correct\nQ1,A,B,,D,A\n",
				expected: "choices",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				reader := bytes.NewReader([]byte(tt.csvData))
				err := service.ImportQuestionsCSV(ctx, f.CourseID, f.QuizID, reader, f.LecturerID)
				require.Error(t, err)
				importErr, ok := err.(*quizzes.ImportError)
				require.True(t, ok, "Expected ImportError")
				require.NotEmpty(t, importErr.RowErrors)
				assert.Equal(t, tt.expected, importErr.RowErrors[0].Field)
			})
		}
	})

	t.Run("Idempotent Submit - Attempt Same Question ID Multiple Times", func(t *testing.T) {
		f := testutil.SetupQuizzesFixture(t, ctx, pool)
		// Start attempt
		attempt, err := service.StartAttempt(ctx, f.CourseID, f.QuizID, f.StudentID)
		require.NoError(t, err)

		req := quizzes.SubmitAttemptRequest{
			Answers: map[int64][]int64{
				99999: {f.Options[0].ID}, // Fake question ID not in quiz
			},
		}

		// It should fail due to invalid question ID validation
		_, err = service.SubmitAttempt(ctx, f.CourseID, f.QuizID, attempt.ID, f.StudentID, req)
		assert.Error(t, err, "Should reject invalid question IDs")
		
		// Attempt again with valid data
		req2 := quizzes.SubmitAttemptRequest{
			Answers: map[int64][]int64{
				f.Questions[0].ID: {f.Options[0].ID}, 
			},
		}
		
		_, err = service.SubmitAttempt(ctx, f.CourseID, f.QuizID, attempt.ID, f.StudentID, req2)
		require.NoError(t, err)
		
		// Double submit -> idempotent/returns same grade
		_, err = service.SubmitAttempt(ctx, f.CourseID, f.QuizID, attempt.ID, f.StudentID, req2)
		require.NoError(t, err)
	})

	t.Run("Non-leakage - Answers Hidden Before Submission", func(t *testing.T) {
		f := testutil.SetupQuizzesFixture(t, ctx, pool)
		// Start attempt
		attempt, err := service.StartAttempt(ctx, f.CourseID, f.QuizID, f.StudentID)
		require.NoError(t, err)
		
		view, err := service.GetAttempt(ctx, f.CourseID, f.QuizID, attempt.ID, f.StudentID)
		require.NoError(t, err)
		
		// Serialize to JSON
		b, err := json.Marshal(view)
		require.NoError(t, err)
		
		jsonStr := string(b)
		// Assert that is_correct and correct_options are NOT leaked
		assert.NotContains(t, jsonStr, `"is_correct"`)
		assert.NotContains(t, jsonStr, `"correct_options"`)
		
		// Now artificially close the quiz by updating close_at to past, and simulate auto-submit
		_, err = pool.Exec(ctx, "UPDATE quizzes SET close_at = now() - interval '1 hour' WHERE id = $1", f.QuizID)
		require.NoError(t, err)
		
		// Attempt is now closed, let's fetch again
		viewClosed, err := service.GetAttempt(ctx, f.CourseID, f.QuizID, attempt.ID, f.StudentID)
		require.NoError(t, err)
		
		bClosed, err := json.Marshal(viewClosed)
		require.NoError(t, err)
		
		jsonClosedStr := string(bClosed)
		// Now it should contain correct options
		assert.Contains(t, jsonClosedStr, `"is_correct"`)
		assert.Contains(t, jsonClosedStr, `"correct_options"`)
	})
}
