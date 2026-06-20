-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdatePasswordAndStamp :exec
UPDATE users SET password_hash = $2, password_changed_at = now(), must_change_password = false, updated_at = now() WHERE id = $1 AND deleted_at IS NULL;

-- name: GetSystemUserID :one
SELECT id FROM users WHERE is_system = TRUE LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (username, password_hash, role, full_name, date_of_birth, must_change_password)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: ResetUserPassword :exec
UPDATE users SET password_hash = $2, must_change_password = true, password_changed_at = now(), updated_at = now() WHERE id = $1 AND deleted_at IS NULL AND is_system = FALSE;

-- name: ListUsers :many
SELECT * FROM users
WHERE deleted_at IS NULL AND is_system = FALSE
  AND (sqlc.narg('role')::user_role IS NULL OR role = sqlc.narg('role'))
  AND (sqlc.narg('search')::text IS NULL OR username ILIKE '%' || sqlc.narg('search') || '%' OR full_name ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY created_at DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: CountUsers :one
SELECT COUNT(*) FROM users
WHERE deleted_at IS NULL AND is_system = FALSE
  AND (sqlc.narg('role')::user_role IS NULL OR role = sqlc.narg('role'))
  AND (sqlc.narg('search')::text IS NULL OR username ILIKE '%' || sqlc.narg('search') || '%' OR full_name ILIKE '%' || sqlc.narg('search') || '%');

-- name: GetActiveUsernames :many
SELECT username FROM users WHERE username = ANY($1::text[]) AND deleted_at IS NULL AND is_system = FALSE;
