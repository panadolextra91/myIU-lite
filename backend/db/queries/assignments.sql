-- name: CreateAssignment :one
INSERT INTO assignments (
    course_id, title, description, deadline, accept_late, late_threshold_days, created_by, max_score
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetAssignmentByID :one
SELECT a.*
FROM assignments a
JOIN courses c ON a.course_id = c.id
WHERE a.id = $1 AND c.deleted_at IS NULL;

-- name: ListCourseAssignments :many
SELECT a.*
FROM assignments a
JOIN courses c ON a.course_id = c.id
WHERE a.course_id = $1 AND c.deleted_at IS NULL
ORDER BY a.created_at DESC;

-- name: FinalizeAssignmentGrading :one
UPDATE assignments
SET grading_finalized_at = now()
WHERE id = $1 AND course_id = $2 AND grading_finalized_at IS NULL
RETURNING *;
