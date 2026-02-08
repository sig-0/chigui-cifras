-- name: Subscribe :exec
INSERT INTO subscriptions (chat_id, frequency, next_send_at) VALUES ($1, $2, $3)
ON CONFLICT (chat_id) DO UPDATE SET frequency = $2, next_send_at = $3;

-- name: Unsubscribe :exec
DELETE FROM subscriptions WHERE chat_id = $1;

-- name: DueSubscribers :many
SELECT chat_id, frequency FROM subscriptions WHERE next_send_at <= now() ORDER BY next_send_at;

-- name: UpdateNextSend :exec
UPDATE subscriptions SET next_send_at = $2 WHERE chat_id = $1;
