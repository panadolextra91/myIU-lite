-- name: CreateGradeScheme :one
INSERT INTO grade_schemes (course_id, created_by)
VALUES ($1, $2)
RETURNING *;

-- name: GetSchemeByCourse :one
SELECT gs.*
FROM grade_schemes gs
JOIN courses c ON gs.course_id = c.id
WHERE gs.course_id = $1 AND c.deleted_at IS NULL;

-- name: InsertGradeComponent :one
INSERT INTO grade_components (scheme_id, parent_id, name, weight, source_type, auto_kind)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListSchemeComponents :many
SELECT gc.*
FROM grade_components gc
JOIN grade_schemes gs ON gc.scheme_id = gs.id
WHERE gs.course_id = $1
ORDER BY gc.parent_id NULLS FIRST, gc.id;

-- name: UpsertGradeScore :exec
INSERT INTO grade_scores (component_id, student_id, score)
VALUES ($1, $2, $3)
ON CONFLICT (component_id, student_id)
DO UPDATE SET score = EXCLUDED.score, updated_at = now();

-- name: ListScoresForStudent :many
SELECT s.*
FROM grade_scores s
JOIN grade_components c ON s.component_id = c.id
JOIN grade_schemes gs ON c.scheme_id = gs.id
WHERE gs.course_id = $1 AND s.student_id = $2;

-- name: ComputeQuizAverage :one
WITH eligible AS (
    SELECT q.id, q.max_grade
    FROM quizzes q
    JOIN courses c ON q.course_id = c.id
    WHERE q.course_id = $1 AND c.deleted_at IS NULL
      AND q.close_at IS NOT NULL AND q.close_at <= now()
      AND q.max_grade IS NOT NULL AND q.max_grade > 0
), per_quiz AS (
    SELECT e.id,
           COALESCE(
             (SELECT MAX(a.score) FROM quiz_attempts a
              WHERE a.quiz_id = e.id AND a.student_id = $2
                AND a.status IN ('SUBMITTED','AUTO_SUBMITTED')),
             0) / e.max_grade * 100 AS normalized
    FROM eligible e
)
SELECT COALESCE(AVG(normalized), 0)::numeric AS quiz_average,
       COUNT(*)::int AS eligible_count
FROM per_quiz;

-- name: ComputeAssignmentAverage :one
WITH eligible AS (
    SELECT a.id, a.max_score
    FROM assignments a
    JOIN courses c ON a.course_id = c.id
    WHERE a.course_id = $1 AND c.deleted_at IS NULL
      AND a.grading_finalized_at IS NOT NULL
      AND a.max_score IS NOT NULL AND a.max_score > 0
), per_assignment AS (
    SELECT e.id,
           COALESCE(
             (SELECT s.score FROM submissions s
              WHERE s.assignment_id = e.id AND s.student_id = $2
              ORDER BY s.version DESC LIMIT 1),
             0) / e.max_score * 100 AS normalized
    FROM eligible e
)
SELECT COALESCE(AVG(normalized), 0)::numeric AS assignment_average,
       COUNT(*)::int AS eligible_count
FROM per_assignment;

-- name: CountSchemeScores :one
SELECT COUNT(*)
FROM grade_scores s
JOIN grade_components c ON s.component_id = c.id
WHERE c.scheme_id = $1;

-- name: CountSchemePublications :one
SELECT COUNT(*)
FROM grade_publications p
JOIN grade_components c ON p.component_id = c.id
WHERE c.scheme_id = $1;

-- name: DeleteSchemeIfEmpty :execrows
DELETE FROM grade_schemes gs
WHERE gs.id = $1 AND gs.course_id = $2
  AND NOT EXISTS (
      SELECT 1 FROM grade_components c
      JOIN grade_scores s ON s.component_id = c.id
      WHERE c.scheme_id = gs.id
  )
  AND NOT EXISTS (
      SELECT 1 FROM grade_components c
      JOIN grade_publications p ON p.component_id = c.id
      WHERE c.scheme_id = gs.id
  );

-- name: DeleteSchemeComponents :exec
DELETE FROM grade_components WHERE scheme_id = $1;
