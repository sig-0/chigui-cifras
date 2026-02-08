-- name: CreateAlert :one
-- Atomically inserts an alert only if the user has fewer than $6 active alerts.
-- Returns no rows (pgx.ErrNoRows) when the limit is reached.
INSERT INTO alerts (id, chat_id, base, direction, threshold)
SELECT $1, $2, $3, $4, $5
WHERE (SELECT COUNT(*) FROM alerts a WHERE a.chat_id = $2 AND a.triggered = false) < sqlc.arg(max_per_chat)
RETURNING id, chat_id, base, direction, threshold, triggered, created_at;

-- name: AlertsByChat :many
SELECT id, chat_id, base, direction, threshold, triggered, created_at
FROM alerts WHERE chat_id = $1 AND triggered = false
ORDER BY created_at;

-- name: DeleteAlert :exec
DELETE FROM alerts WHERE id = $1 AND chat_id = $2;

-- name: ActiveAlerts :many
SELECT id, chat_id, base, direction, threshold, triggered, created_at
FROM alerts WHERE triggered = false ORDER BY base, chat_id;

-- name: TriggerAlert :exec
UPDATE alerts SET triggered = true WHERE id = $1;
