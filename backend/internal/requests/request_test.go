package requests_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/requests"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/stretchr/testify/require"
	"fmt"
	"time"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, *requests.Service, *db.Queries) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	q := db.New(pool)
	repo := requests.NewRepository(q)
	svc := requests.NewService(repo, pool)

	return pool, svc, q
}

func TestRequestsIntegration(t *testing.T) {
	pool, svc, q := setupTestDB(t)
	defer pool.Close()
	ctx := context.Background()

	// Seed data
	ts := time.Now().UnixNano()
	var courseID int64
	err := pool.QueryRow(ctx, fmt.Sprintf(`INSERT INTO courses (code, name, term, start_date, end_date) VALUES ('REQ_%d', 'Req', 'Fall', now(), now() + interval '1 month') RETURNING id`, ts)).Scan(&courseID)
	require.NoError(t, err)

	var l1, l2, st int64
	l1Name := fmt.Sprintf("l1_%d", ts)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'lecturer') RETURNING id`, l1Name).Scan(&l1)
	require.NoError(t, err)

	l2Name := fmt.Sprintf("l2_%d", ts)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'lecturer') RETURNING id`, l2Name).Scan(&l2)
	require.NoError(t, err)

	stName := fmt.Sprintf("st_%d", ts)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'student') RETURNING id`, stName).Scan(&st)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM notifications WHERE recipient_id IN ($1, $2, $3)`, l1, l2, st)
		_, _ = pool.Exec(ctx, `DELETE FROM requests WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM student_enrollments WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM course_lecturers WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2, $3)`, l1, l2, st)
	})

	_, err = pool.Exec(ctx, `INSERT INTO course_lecturers (course_id, lecturer_id) VALUES ($1, $2), ($1, $3)`, courseID, l1, l2)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO student_enrollments (course_id, student_id) VALUES ($1, $2)`, courseID, st)
	require.NoError(t, err)

	var reqID int64

	t.Run("Create directed request + notify", func(t *testing.T) {
		r, err := svc.CreateRequest(ctx, courseID, st, requests.CreateRequestRequest{
			Type:               "LEAVE_EARLY",
			Title:              "Doctor appt",
			Body:               "I need to leave early.",
			TargetedLecturerID: l1,
		})
		require.NoError(t, err)
		reqID = r.ID

		// Assert L1 is notified
		notifs, err := q.ListNotifications(ctx, db.ListNotificationsParams{
			RecipientID: l1,
			Limit:       10,
			Offset:      0,
		})
		require.NoError(t, err)
		found := false
		for _, n := range notifs {
			if n.Type == "REQUEST_CREATED" && n.ResourceID.Valid && n.ResourceID.Int64 == reqID {
				found = true
				break
			}
		}
		require.True(t, found, "L1 should be notified")

		// Assert L1 sees it, L2 does not
		l1Reqs, err := svc.ListForLecturer(ctx, l1)
		require.NoError(t, err)
		require.Len(t, l1Reqs, 1)

		l2Reqs, err := svc.ListForLecturer(ctx, l2)
		require.NoError(t, err)
		require.Len(t, l2Reqs, 0)
	})

	t.Run("L2 cannot reply", func(t *testing.T) {
		_, err := svc.ReplyRequest(ctx, reqID, l2, requests.ReplyRequestRequest{
			Decision: "APPROVED",
		})
		require.ErrorIs(t, err, requests.ErrNotTargeted)
	})

	t.Run("L1 replies + notify student", func(t *testing.T) {
		r, err := svc.ReplyRequest(ctx, reqID, l1, requests.ReplyRequestRequest{
			Decision: "APPROVED",
			Note:     "Sure.",
		})
		require.NoError(t, err)
		require.Equal(t, "APPROVED", r.Status)
		require.Equal(t, "Sure.", r.ReplyNote.String)

		// Assert student is notified
		notifs, err := q.ListNotifications(ctx, db.ListNotificationsParams{
			RecipientID: st,
			Limit:       10,
			Offset:      0,
		})
		require.NoError(t, err)
		found := false
		for _, n := range notifs {
			if n.Type == "REQUEST_REPLIED" && n.ResourceID.Valid && n.ResourceID.Int64 == reqID {
				found = true
				break
			}
		}
		require.True(t, found, "Student should be notified")
	})

	t.Run("Closed permanently (no second reply)", func(t *testing.T) {
		_, err := svc.ReplyRequest(ctx, reqID, l1, requests.ReplyRequestRequest{
			Decision: "DENIED",
		})
		require.ErrorIs(t, err, requests.ErrAlreadyClosed)
	})
}
