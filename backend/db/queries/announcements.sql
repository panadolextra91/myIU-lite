-- name: InsertAnnouncement :one
INSERT INTO announcements (course_id, author_id, title, body, audience_type)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: InsertAnnouncementRecipient :exec
INSERT INTO announcement_recipients (announcement_id, student_id)
VALUES ($1, $2);

-- name: ListCourseAnnouncements :many
SELECT a.*
FROM announcements a
JOIN courses c ON a.course_id = c.id
WHERE a.course_id = $1 AND c.deleted_at IS NULL
ORDER BY a.created_at DESC;

-- name: GetAnnouncementByID :one
SELECT a.*
FROM announcements a
JOIN courses c ON a.course_id = c.id
WHERE a.id = $1 AND c.deleted_at IS NULL;

-- name: ListAnnouncementRecipients :many
SELECT student_id
FROM announcement_recipients
WHERE announcement_id = $1;

-- name: ListAnnouncementsForStudent :many
SELECT a.*
FROM announcements a
JOIN courses c ON a.course_id = c.id
JOIN student_enrollments se ON se.course_id = a.course_id AND se.student_id = $2
LEFT JOIN announcement_recipients ar ON ar.announcement_id = a.id AND ar.student_id = $2
WHERE a.course_id = $1 AND c.deleted_at IS NULL
  AND (a.audience_type = 'ALL_STUDENTS' OR ar.student_id IS NOT NULL)
ORDER BY a.created_at DESC;
