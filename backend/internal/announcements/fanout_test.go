package announcements_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/panadolextra91/myiu-lite/backend/internal/announcements"
	"github.com/panadolextra91/myiu-lite/backend/internal/shared/db"
	"github.com/stretchr/testify/require"
	"fmt"
	"time"
)

func setupTestDB(t *testing.T) (*pgxpool.Pool, *announcements.Service, *db.Queries) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)

	q := db.New(pool)
	repo := announcements.NewRepository(q)
	svc := announcements.NewService(repo, pool)

	return pool, svc, q
}

func TestAnnouncementFanout(t *testing.T) {
	pool, svc, q := setupTestDB(t)
	defer pool.Close()

	ctx := context.Background()

	// Create course
	ts := time.Now().UnixNano()
	var courseID int64
	err := pool.QueryRow(ctx, fmt.Sprintf(`INSERT INTO courses (code, name, term, start_date, end_date) VALUES ('ANNC_%d', 'Annc', 'Fall', now(), now() + interval '1 month') RETURNING id`, ts)).Scan(&courseID)
	require.NoError(t, err)

	// Create users
	var lecturerID, st1, st2, st3 int64
	lName := fmt.Sprintf("lect_%d", ts)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'lecturer') RETURNING id`, lName).Scan(&lecturerID)
	require.NoError(t, err)

	s1Name := fmt.Sprintf("st1_%d", ts)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'student') RETURNING id`, s1Name).Scan(&st1)
	require.NoError(t, err)

	s2Name := fmt.Sprintf("st2_%d", ts)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'student') RETURNING id`, s2Name).Scan(&st2)
	require.NoError(t, err)

	s3Name := fmt.Sprintf("st3_%d", ts)
	err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'student') RETURNING id`, s3Name).Scan(&st3)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM notifications WHERE recipient_id IN ($1, $2, $3, $4)`, lecturerID, st1, st2, st3)
		_, _ = pool.Exec(ctx, `DELETE FROM announcement_recipients WHERE student_id IN ($1, $2, $3)`, st1, st2, st3)
		_, _ = pool.Exec(ctx, `DELETE FROM announcements WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM student_enrollments WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM course_lecturers WHERE course_id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM courses WHERE id = $1`, courseID)
		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1, $2, $3, $4)`, lecturerID, st1, st2, st3)
	})

	// Enroll users
	_, err = pool.Exec(ctx, `INSERT INTO course_lecturers (course_id, lecturer_id) VALUES ($1, $2)`, courseID, lecturerID)
	require.NoError(t, err)

	for _, st := range []int64{st1, st2, st3} {
		_, err = pool.Exec(ctx, `INSERT INTO student_enrollments (course_id, student_id) VALUES ($1, $2)`, courseID, st)
		require.NoError(t, err)
	}

	t.Run("ALL_STUDENTS fan-out", func(t *testing.T) {
		req := announcements.CreateAnnouncementRequest{
			Title:        "All title",
			Body:         "All body",
			AudienceType: "ALL_STUDENTS",
		}
		ann, err := svc.CreateAnnouncement(ctx, courseID, lecturerID, req)
		require.NoError(t, err)

		// Verify notifications for all students
		for _, st := range []int64{st1, st2, st3} {
			notifs, err := q.ListNotifications(ctx, db.ListNotificationsParams{
				RecipientID: st,
				Limit:       10,
				Offset:      0,
			})
			require.NoError(t, err)
			found := false
			for _, n := range notifs {
				if n.Type == "ANNOUNCEMENT" && n.Title == "All title" && n.ResourceID.Valid && n.ResourceID.Int64 == ann.ID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected notification for student %d", st)
			}
		}

		// Student 1 can view
		studentList, err := svc.ListForStudent(ctx, courseID, st1)
		require.NoError(t, err)
		if len(studentList) != 1 || studentList[0].ID != ann.ID {
			t.Errorf("expected student 1 to see announcement, got %d", len(studentList))
		}
	})

	t.Run("SPECIFIC_STUDENTS fan-out", func(t *testing.T) {
		req := announcements.CreateAnnouncementRequest{
			Title:        "Specific title",
			Body:         "Specific body",
			AudienceType: "SPECIFIC_STUDENTS",
			StudentIDs:   []int64{st1, st2},
		}
		ann, err := svc.CreateAnnouncement(ctx, courseID, lecturerID, req)
		require.NoError(t, err)

		// Check st1 and st2 received it
		for _, st := range []int64{st1, st2} {
			notifs, err := q.ListNotifications(ctx, db.ListNotificationsParams{
				RecipientID: st,
				Limit:       10,
				Offset:      0,
			})
			require.NoError(t, err)
			found := false
			for _, n := range notifs {
				if n.Type == "ANNOUNCEMENT" && n.Title == "Specific title" && n.ResourceID.Valid && n.ResourceID.Int64 == ann.ID {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected notification for specific student %d", st)
			}
		}

		// Check st3 did NOT receive it
		notifs3, err := q.ListNotifications(ctx, db.ListNotificationsParams{
			RecipientID: st3,
			Limit:       10,
			Offset:      0,
		})
		require.NoError(t, err)
		for _, n := range notifs3 {
			if n.Type == "ANNOUNCEMENT" && n.Title == "Specific title" && n.ResourceID.Valid && n.ResourceID.Int64 == ann.ID {
				t.Errorf("student 3 should not receive specific notification")
			}
		}

		// Student 1 can view
		st1List, err := svc.ListForStudent(ctx, courseID, st1)
		require.NoError(t, err)
		found1 := false
		for _, a := range st1List {
			if a.ID == ann.ID {
				found1 = true
			}
		}
		if !found1 {
			t.Errorf("expected st1 to see specific announcement")
		}

		// Student 3 cannot view
		st3List, err := svc.ListForStudent(ctx, courseID, st3)
		require.NoError(t, err)
		for _, a := range st3List {
			if a.ID == ann.ID {
				t.Errorf("student 3 should not see specific announcement")
			}
		}
	})

	t.Run("atomicity - invalid specific student", func(t *testing.T) {
		req := announcements.CreateAnnouncementRequest{
			Title:        "Fail title",
			Body:         "Fail body",
			AudienceType: "SPECIFIC_STUDENTS",
			StudentIDs:   []int64{st1, 99999}, // 99999 is invalid/unenrolled
		}
		_, err := svc.CreateAnnouncement(ctx, courseID, lecturerID, req)
		if err == nil {
			t.Fatalf("expected error for invalid student ID")
		}

		// Verify announcement was NOT created
		anns, err := svc.ListForCourse(ctx, courseID, lecturerID)
		require.NoError(t, err)
		for _, a := range anns {
			if a.Title == "Fail title" {
				t.Errorf("announcement should have been rolled back")
			}
		}
	})

	t.Run("IDOR - specific student visibility", func(t *testing.T) {
		req := announcements.CreateAnnouncementRequest{
			Title:        "Specific IDOR title",
			Body:         "Specific IDOR body",
			AudienceType: "SPECIFIC_STUDENTS",
			StudentIDs:   []int64{st1},
		}
		ann, err := svc.CreateAnnouncement(ctx, courseID, lecturerID, req)
		require.NoError(t, err)

		// st1 can GetByID
		_, err = svc.GetByID(ctx, courseID, ann.ID, st1, db.UserRoleStudent)
		require.NoError(t, err)

		// st2 cannot GetByID
		_, err = svc.GetByID(ctx, courseID, ann.ID, st2, db.UserRoleStudent)
		require.ErrorIs(t, err, announcements.ErrNotFound)

		// un-enrolled user cannot GetByID
		var stOther int64
		stOtherName := fmt.Sprintf("st_other_%d", ts)
		err = pool.QueryRow(ctx, `INSERT INTO users (username, password_hash, role) VALUES ($1, 'hash', 'student') RETURNING id`, stOtherName).Scan(&stOther)
		require.NoError(t, err)

		_, err = svc.GetByID(ctx, courseID, ann.ID, stOther, db.UserRoleStudent)
		require.Error(t, err)

		_, _ = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, stOther)
	})
}
