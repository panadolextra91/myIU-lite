-- name: CreateQuiz :one
INSERT INTO quizzes (
    course_id, title, pool_size, max_questions, max_grade,
    shuffle, retake_count, open_at, close_at, created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetQuizByID :one
SELECT q.*
FROM quizzes q
JOIN courses c ON q.course_id = c.id
WHERE q.id = $1 AND c.deleted_at IS NULL;

-- name: ListCourseQuizzes :many
SELECT q.*
FROM quizzes q
JOIN courses c ON q.course_id = c.id
WHERE q.course_id = $1 AND c.deleted_at IS NULL
ORDER BY q.created_at DESC;
