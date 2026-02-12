CREATE TABLE IF NOT EXISTS scan_schedules (
    id SERIAL PRIMARY KEY,
    target TEXT NOT NULL,
    cron_expr TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
