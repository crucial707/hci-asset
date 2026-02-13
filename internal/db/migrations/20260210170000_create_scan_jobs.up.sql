CREATE TABLE IF NOT EXISTS scan_jobs (
    id SERIAL PRIMARY KEY,
    target TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'running',
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ NULL,
    error TEXT NULL,
    assets JSONB NULL
);

CREATE INDEX IF NOT EXISTS idx_scan_jobs_started_at ON scan_jobs (started_at DESC);
