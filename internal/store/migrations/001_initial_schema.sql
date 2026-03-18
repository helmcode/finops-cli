-- Migration 001: Initial schema

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

CREATE TABLE IF NOT EXISTS config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
