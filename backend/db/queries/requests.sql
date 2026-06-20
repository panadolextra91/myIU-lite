-- name: InsertRequest :one
INSERT INTO requests (course_id, student_id, targeted_lecturer_id, type, title, body, status)
VALUES ($1, $2, $3, $4, $5, $6, 'PENDING')
RETURNING *;

-- name: ReplyRequest :one
UPDATE requests
SET status = $3,
    reply_note = $4,
    replied_at = now()
WHERE id = $1
  AND targeted_lecturer_id = $2
  AND status = 'PENDING'
RETURNING *;

-- name: ListLecturerRequests :many
SELECT *
FROM requests
WHERE targeted_lecturer_id = $1
ORDER BY created_at DESC;

-- name: ListStudentRequests :many
SELECT *
FROM requests
WHERE student_id = $1
ORDER BY created_at DESC;

-- name: GetRequestByID :one
SELECT r.*
FROM requests r
JOIN courses c ON r.course_id = c.id
WHERE r.id = $1 AND c.deleted_at IS NULL;
