-- name: UpsertCostRecord :exec
INSERT INTO cost_records (provider, account_id, service, region, period_start, period_end, granularity, amount, currency, synced_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider, account_id, service, region, period_start, granularity)
DO UPDATE SET
    period_end = excluded.period_end,
    amount = excluded.amount,
    currency = excluded.currency,
    synced_at = excluded.synced_at;

-- name: GetCostRecordsByProvider :many
SELECT * FROM cost_records
WHERE provider = ?
ORDER BY period_start DESC;

-- name: GetCostRecordsByAccount :many
SELECT * FROM cost_records
WHERE provider = ? AND account_id = ?
ORDER BY period_start DESC;

-- name: GetCostRecordsByDateRange :many
SELECT * FROM cost_records
WHERE provider = ? AND period_start >= ? AND period_end <= ?
ORDER BY period_start DESC;

-- name: GetCostRecordsByAccountAndDateRange :many
SELECT * FROM cost_records
WHERE provider = ? AND account_id = ? AND period_start >= ? AND period_end <= ?
ORDER BY period_start DESC;

-- name: GetCostRecordsByService :many
SELECT * FROM cost_records
WHERE provider = ? AND service = ?
ORDER BY period_start DESC;

-- name: GetTotalCostByService :many
SELECT service, SUM(amount) AS total_amount, currency
FROM cost_records
WHERE provider = ? AND period_start >= ? AND period_end <= ?
GROUP BY service, currency
ORDER BY total_amount DESC;

-- name: GetTotalCostByRegion :many
SELECT region, SUM(amount) AS total_amount, currency
FROM cost_records
WHERE provider = ? AND period_start >= ? AND period_end <= ?
GROUP BY region, currency
ORDER BY total_amount DESC;

-- name: GetMonthlyCostTrend :many
SELECT period_start, SUM(amount) AS total_amount, currency
FROM cost_records
WHERE provider = ?
GROUP BY period_start, currency
ORDER BY period_start ASC;

-- name: GetMonthlyCostTrendByService :many
SELECT period_start, SUM(amount) AS total_amount, currency
FROM cost_records
WHERE provider = ? AND service = ?
GROUP BY period_start, currency
ORDER BY period_start ASC;

-- name: GetMonthlyCostByAccount :many
SELECT account_id, period_start, SUM(amount) AS total_amount, currency
FROM cost_records
WHERE provider = ? AND period_start >= ? AND period_end <= ?
GROUP BY account_id, period_start, currency
ORDER BY account_id, period_start ASC;

-- name: CountCostRecords :one
SELECT COUNT(*) FROM cost_records;

-- name: CountCostRecordsByProvider :one
SELECT COUNT(*) FROM cost_records WHERE provider = ?;

-- name: DeleteCostRecordsOlderThan :exec
DELETE FROM cost_records WHERE period_start < ?;

-- name: GetLatestSyncedPeriod :one
SELECT MAX(period_end) AS latest_period_end
FROM cost_records
WHERE provider = ? AND account_id = ?;

-- name: GetCostByServiceForRegion :many
SELECT service, SUM(amount) AS total_amount, currency
FROM cost_records
WHERE provider = ? AND region = ? AND period_start >= ? AND period_end <= ?
GROUP BY service, currency
ORDER BY total_amount DESC;

-- name: GetDistinctServices :many
SELECT DISTINCT service FROM cost_records
WHERE provider = ?
ORDER BY service;

-- name: GetDistinctRegions :many
SELECT DISTINCT region FROM cost_records
WHERE provider = ? AND region IS NOT NULL
ORDER BY region;

-- name: GetTotalCostByAccount :many
SELECT account_id, SUM(amount) AS total_amount, currency
FROM cost_records
WHERE provider = ? AND period_start >= ? AND period_end <= ?
GROUP BY account_id, currency
ORDER BY total_amount DESC;

-- name: GetTopServicesByAccount :many
SELECT service, SUM(amount) AS total_amount, currency
FROM cost_records
WHERE provider = ? AND account_id = ? AND period_start >= ? AND period_end <= ?
GROUP BY service, currency
ORDER BY total_amount DESC
LIMIT 5;

-- name: GetDistinctAccounts :many
SELECT DISTINCT account_id FROM cost_records
WHERE provider = ?
ORDER BY account_id;

-- name: GetCostByAccountAndService :many
SELECT account_id, service, region, SUM(amount) AS total_amount, currency
FROM cost_records
WHERE provider = ? AND period_start >= ? AND period_end <= ?
GROUP BY account_id, service, region, currency
ORDER BY account_id, total_amount DESC;
