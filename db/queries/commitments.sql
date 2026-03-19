-- name: UpsertCommitment :exec
INSERT INTO commitments (provider, account_id, commitment_type, period_start, period_end, total_commitment, used_commitment, on_demand_equivalent, net_savings, utilization_pct, coverage_pct, currency, synced_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider, account_id, commitment_type, period_start)
DO UPDATE SET
    period_end = excluded.period_end,
    total_commitment = excluded.total_commitment,
    used_commitment = excluded.used_commitment,
    on_demand_equivalent = excluded.on_demand_equivalent,
    net_savings = excluded.net_savings,
    utilization_pct = excluded.utilization_pct,
    coverage_pct = excluded.coverage_pct,
    currency = excluded.currency,
    synced_at = excluded.synced_at;

-- name: GetCommitmentSummary :many
SELECT commitment_type,
    SUM(total_commitment) AS total_commitment,
    SUM(used_commitment) AS used_commitment,
    SUM(on_demand_equivalent) AS on_demand_equivalent,
    SUM(net_savings) AS net_savings,
    currency
FROM commitments
WHERE provider = ? AND period_start >= ? AND period_end <= ?
GROUP BY commitment_type, currency
ORDER BY commitment_type;

-- name: GetCommitmentSummaryByAccount :many
SELECT account_id, commitment_type,
    SUM(total_commitment) AS total_commitment,
    SUM(used_commitment) AS used_commitment,
    SUM(on_demand_equivalent) AS on_demand_equivalent,
    SUM(net_savings) AS net_savings,
    currency
FROM commitments
WHERE provider = ? AND period_start >= ? AND period_end <= ?
GROUP BY account_id, commitment_type, currency
ORDER BY account_id, commitment_type;

-- name: GetCommitmentTrend :many
SELECT period_start, commitment_type,
    total_commitment,
    used_commitment,
    utilization_pct,
    coverage_pct,
    net_savings,
    currency
FROM commitments
WHERE provider = ? AND period_start >= ? AND period_end <= ?
ORDER BY period_start ASC, commitment_type;

-- name: GetAggregatedCommitmentMetrics :one
SELECT
    COALESCE(SUM(total_commitment), 0) AS total_commitment,
    COALESCE(SUM(used_commitment), 0) AS used_commitment,
    COALESCE(SUM(net_savings), 0) AS net_savings,
    CASE WHEN SUM(total_commitment) > 0
        THEN SUM(used_commitment) * 100.0 / SUM(total_commitment)
        ELSE 0
    END AS avg_utilization,
    currency
FROM commitments
WHERE provider = ? AND period_start >= ? AND period_end <= ?
GROUP BY currency;

-- name: CountCommitments :one
SELECT COUNT(*) FROM commitments WHERE provider = ?;

-- name: DeleteCommitmentsOlderThan :exec
DELETE FROM commitments WHERE period_start < ?;
