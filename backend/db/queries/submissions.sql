-- name: InsertSubmissionVersion :one
INSERT INTO submissions (
    assignment_id, student_id, version, cloudinary_public_id, cloudinary_format,
    original_filename, is_late
) VALUES (
    $1, $2, 
    (COALESCE((SELECT MAX(version) FROM submissions WHERE assignment_id=$1 AND student_id=$2), 0) + 1), 
    $3, $4, $5, $6
) RETURNING *;

-- name: GetMaxSubmissionVersion :one
SELECT COALESCE(MAX(version), 0)::INT
FROM submissions
WHERE assignment_id = $1 AND student_id = $2;

-- name: GetActiveSubmission :one
SELECT s.*, a.title as assignment_title, a.course_id
FROM submissions s
JOIN assignments a ON s.assignment_id = a.id
WHERE a.id = $1 AND s.student_id = $2
ORDER BY s.version DESC
LIMIT 1;

-- name: ListSubmissionVersions :many
SELECT *
FROM submissions
WHERE assignment_id = $1 AND student_id = $2
ORDER BY version DESC;

-- name: GetSubmissionByID :one
SELECT s.*, a.title as assignment_title, a.course_id
FROM submissions s
JOIN assignments a ON s.assignment_id = a.id
JOIN courses c ON a.course_id = c.id
WHERE s.id = $1 AND c.deleted_at IS NULL;

-- name: UpsertSubmissionGrade :exec
UPDATE submissions
SET score = $2, feedback = $3, graded_at = now(), graded_by = $4
WHERE id = $1;
