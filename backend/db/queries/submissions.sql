-- name: InsertSubmissionVersion :one
INSERT INTO submissions (
    assignment_id, student_id, version, cloudinary_public_id, cloudinary_format,
    original_filename, is_late
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetMaxSubmissionVersion :one
SELECT COALESCE(MAX(version), 0)::INT
FROM submissions
WHERE assignment_id = $1 AND student_id = $2;

-- name: GetActiveSubmission :one
SELECT *
FROM submissions
WHERE assignment_id = $1 AND student_id = $2
ORDER BY version DESC
LIMIT 1;

-- name: ListSubmissionVersions :many
SELECT *
FROM submissions
WHERE assignment_id = $1 AND student_id = $2
ORDER BY version DESC;

-- name: GetSubmissionByID :one
SELECT s.*
FROM submissions s
JOIN assignments a ON s.assignment_id = a.id
JOIN courses c ON a.course_id = c.id
WHERE s.id = $1 AND c.deleted_at IS NULL;
