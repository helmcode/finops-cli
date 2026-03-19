-- FinOps CLI database schema

CREATE TABLE IF NOT EXISTS cost_records (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    provider     TEXT    NOT NULL,
    account_id   TEXT    NOT NULL,
    service      TEXT    NOT NULL,
    region       TEXT,
    period_start TEXT    NOT NULL,
    period_end   TEXT    NOT NULL,
    granularity  TEXT    NOT NULL DEFAULT 'MONTHLY',
    amount       REAL    NOT NULL,
    currency     TEXT    NOT NULL DEFAULT 'USD',
    synced_at    TEXT    NOT NULL,
    UNIQUE(provider, account_id, service, region, period_start, granularity)
);

CREATE TABLE IF NOT EXISTS resources (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    provider      TEXT    NOT NULL,
    account_id    TEXT    NOT NULL,
    service       TEXT    NOT NULL,
    resource_id   TEXT    NOT NULL,
    resource_type TEXT    NOT NULL,
    name          TEXT,
    region        TEXT,
    spec          TEXT,
    tags          TEXT,
    state         TEXT,
    discovered_at TEXT    NOT NULL,
    UNIQUE(provider, account_id, resource_id)
);

CREATE TABLE IF NOT EXISTS sync_history (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    provider        TEXT    NOT NULL,
    account_id      TEXT    NOT NULL,
    region          TEXT,
    period_start    TEXT    NOT NULL,
    period_end      TEXT    NOT NULL,
    cost_records    INTEGER,
    resources_found INTEGER,
    started_at      TEXT    NOT NULL,
    completed_at    TEXT
);

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

CREATE TABLE IF NOT EXISTS config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
