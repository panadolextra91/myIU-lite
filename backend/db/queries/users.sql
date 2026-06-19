-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdatePasswordAndStamp :exec
UPDATE users SET password_hash = $2, password_changed_at = now(), must_change_password = false, updated_at = now() WHERE id = $1 AND deleted_at IS NULL;
