-- name: InsertQuestion :one
INSERT INTO quiz_questions (
    quiz_id, prompt, question_type
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: InsertOption :one
INSERT INTO quiz_question_options (
    question_id, text, is_correct
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: ListQuestionsForQuiz :many
SELECT * FROM quiz_questions
WHERE quiz_id = $1
ORDER BY id ASC;

-- name: ListOptionsForQuestion :many
SELECT * FROM quiz_question_options
WHERE question_id = $1
ORDER BY id ASC;

-- name: CountQuizQuestions :one
SELECT count(*) FROM quiz_questions
WHERE quiz_id = $1;
