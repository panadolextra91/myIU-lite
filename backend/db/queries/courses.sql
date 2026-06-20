-- name: CreateCourse :one
INSERT INTO courses (code, name, term, start_date, end_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetCourseByID :one
SELECT * FROM courses WHERE id = $1 AND deleted_at IS NULL;

-- name: ListCourses :many
SELECT * FROM courses
WHERE deleted_at IS NULL
  AND (sqlc.narg('term')::text IS NULL OR term = sqlc.narg('term'))
  AND (sqlc.narg('search')::text IS NULL OR code ILIKE '%' || sqlc.narg('search') || '%' OR name ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountCourses :one
SELECT COUNT(*) FROM courses
WHERE deleted_at IS NULL
  AND (sqlc.narg('term')::text IS NULL OR term = sqlc.narg('term'))
  AND (sqlc.narg('search')::text IS NULL OR code ILIKE '%' || sqlc.narg('search') || '%' OR name ILIKE '%' || sqlc.narg('search') || '%');

-- name: UpdateCourse :one
UPDATE courses SET code = $2, name = $3, term = $4, start_date = $5, end_date = $6, updated_at = now()
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: SoftDeleteCourse :exec
UPDATE courses SET deleted_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: ListCourseStudents :many
SELECT u.id AS student_id, u.username, u.full_name
FROM student_enrollments e
JOIN users u ON u.id = e.student_id
WHERE e.course_id = $1 AND u.deleted_at IS NULL;

-- name: ListCourseLecturers :many
SELECT u.id AS lecturer_id, u.username, u.full_name
FROM course_lecturers l
JOIN users u ON u.id = l.lecturer_id
WHERE l.course_id = $1 AND u.deleted_at IS NULL;
