-- name: WriteAuditLog :exec
INSERT INTO audit_log (actor_id, action, target_type, target_id, affected_count, metadata)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListAuditLogs :many
SELECT * FROM audit_log
WHERE (sqlc.narg('actor_id')::bigint IS NULL OR actor_id = sqlc.narg('actor_id'))
  AND (sqlc.narg('action')::text   IS NULL OR action = sqlc.narg('action'))
  AND (sqlc.narg('from')::timestamptz IS NULL OR created_at >= sqlc.narg('from'))
  AND (sqlc.narg('to')::timestamptz   IS NULL OR created_at <= sqlc.narg('to'))
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_log
WHERE (sqlc.narg('actor_id')::bigint IS NULL OR actor_id = sqlc.narg('actor_id'))
  AND (sqlc.narg('action')::text   IS NULL OR action = sqlc.narg('action'))
  AND (sqlc.narg('from')::timestamptz IS NULL OR created_at >= sqlc.narg('from'))
  AND (sqlc.narg('to')::timestamptz   IS NULL OR created_at <= sqlc.narg('to'));
