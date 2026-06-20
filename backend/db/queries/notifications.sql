-- name: InsertNotification :one
INSERT INTO notifications (
    recipient_id, type, title, body, resource_type, resource_id, link
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: ListNotifications :many
SELECT * FROM notifications
WHERE recipient_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountUnread :one
SELECT count(*) FROM notifications
WHERE recipient_id = $1 AND read_at IS NULL;

-- name: MarkRead :exec
UPDATE notifications
SET read_at = now()
WHERE id = $1 AND recipient_id = $2 AND read_at IS NULL;
