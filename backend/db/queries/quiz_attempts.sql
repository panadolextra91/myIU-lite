-- name: GetInProgressAttempt :one
SELECT * FROM quiz_attempts
WHERE quiz_id = $1 AND student_id = $2 AND status = 'IN_PROGRESS';

-- name: CountAttempts :one
SELECT count(*) FROM quiz_attempts
WHERE quiz_id = $1 AND student_id = $2;

-- name: StartAttempt :one
INSERT INTO quiz_attempts (
    quiz_id, student_id, status, attempt_number
) VALUES (
    $1, $2, 'IN_PROGRESS', $3
) RETURNING *;

-- name: GetAttemptByID :one
SELECT * FROM quiz_attempts
WHERE id = $1;

-- name: MarkAttemptSubmitted :execrows
UPDATE quiz_attempts
SET status = 'SUBMITTED', score = $2, submitted_at = now()
WHERE id = $1 AND status = 'IN_PROGRESS';

-- name: MarkAttemptAutoSubmitted :execrows
UPDATE quiz_attempts
SET status = 'AUTO_SUBMITTED', score = $2, submitted_at = $3
WHERE id = $1 AND status = 'IN_PROGRESS';

-- name: GetMaxScore :one
SELECT COALESCE(MAX(score), 0)::numeric AS max_score
FROM quiz_attempts
WHERE quiz_id = $1 AND student_id = $2 AND status IN ('SUBMITTED', 'AUTO_SUBMITTED');

-- name: InsertAttemptAnswer :exec
INSERT INTO quiz_attempt_answers (
    attempt_id, question_id, selected_option_ids
) VALUES (
    $1, $2, $3
);

-- name: UpdateAttemptAnswer :exec
UPDATE quiz_attempt_answers
SET selected_option_ids = $3
WHERE attempt_id = $1 AND question_id = $2;

-- name: ListAttemptAnswers :many
SELECT * FROM quiz_attempt_answers
WHERE attempt_id = $1;
