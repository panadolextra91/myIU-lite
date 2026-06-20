-- name: EnrollStudent :execrows
INSERT INTO student_enrollments (course_id, student_id)
VALUES ($1, $2)
ON CONFLICT (course_id, student_id) DO NOTHING;

-- name: AssignLecturer :execrows
INSERT INTO course_lecturers (course_id, lecturer_id)
VALUES ($1, $2)
ON CONFLICT (course_id, lecturer_id) DO NOTHING;

-- name: RemoveStudent :execrows
DELETE FROM student_enrollments
WHERE course_id = $1 AND student_id = $2;

-- name: UnassignLecturer :execrows
DELETE FROM course_lecturers
WHERE course_id = $1 AND lecturer_id = $2;

-- name: GetUserIDsByRole :many
SELECT id, username
FROM users
WHERE username = ANY($1::text[])
  AND role = $2
  AND is_system = FALSE
  AND deleted_at IS NULL;
