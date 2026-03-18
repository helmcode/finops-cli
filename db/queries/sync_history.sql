-- name: InsertSyncHistory :one
INSERT INTO sync_history (provider, account_id, region, period_start, period_end, cost_records, resources_found, started_at, completed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id;

-- name: UpdateSyncHistoryCompleted :exec
UPDATE sync_history
SET cost_records = ?, resources_found = ?, completed_at = ?
WHERE id = ?;

-- name: GetLatestSync :one
SELECT * FROM sync_history
WHERE provider = ? AND account_id = ?
ORDER BY started_at DESC
LIMIT 1;

-- name: GetLatestSyncByProvider :one
SELECT * FROM sync_history
WHERE provider = ?
ORDER BY started_at DESC
LIMIT 1;

-- name: GetSyncHistory :many
SELECT * FROM sync_history
ORDER BY started_at DESC;

-- name: GetSyncHistoryByProvider :many
SELECT * FROM sync_history
WHERE provider = ?
ORDER BY started_at DESC;

-- name: CountSyncHistory :one
SELECT COUNT(*) FROM sync_history;
