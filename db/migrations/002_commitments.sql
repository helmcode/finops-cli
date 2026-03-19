-- Migration 002: Add commitments table for tracking Savings Plans and Reserved Instances

CREATE TABLE IF NOT EXISTS commitments (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    provider               TEXT    NOT NULL,
    account_id             TEXT    NOT NULL,
    commitment_type        TEXT    NOT NULL, -- 'savings_plan', 'reserved_instance'
    period_start           TEXT    NOT NULL,
    period_end             TEXT    NOT NULL,
    total_commitment       REAL    NOT NULL DEFAULT 0,
    used_commitment        REAL    NOT NULL DEFAULT 0,
    on_demand_equivalent   REAL    NOT NULL DEFAULT 0,
    net_savings            REAL    NOT NULL DEFAULT 0,
    utilization_pct        REAL    NOT NULL DEFAULT 0,
    coverage_pct           REAL    NOT NULL DEFAULT 0,
    currency               TEXT    NOT NULL DEFAULT 'USD',
    synced_at              TEXT    NOT NULL,
    UNIQUE(provider, account_id, commitment_type, period_start)
);
