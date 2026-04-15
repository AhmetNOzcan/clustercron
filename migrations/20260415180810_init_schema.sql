-- +goose Up
CREATE TABLE jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    schedule        TEXT NOT NULL,
    webhook_url     TEXT NOT NULL,
    http_method     TEXT NOT NULL DEFAULT 'POST',
    timeout_seconds INT  NOT NULL DEFAULT 30,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    next_fire_at    TIMESTAMPTZ,
    last_fire_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_jobs_next_fire_at
    ON jobs (next_fire_at)
    WHERE enabled = true AND next_fire_at IS NOT NULL;

CREATE TABLE job_runs (
    run_id          TEXT PRIMARY KEY,
    job_id          UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    status          TEXT NOT NULL CHECK (status IN ('running', 'success', 'failed', 'timed_out')),
    attempt         INT  NOT NULL DEFAULT 0,
    scheduled_at    TIMESTAMPTZ NOT NULL,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    worker_id       TEXT,
    http_status     INT,
    error_message   TEXT
);

CREATE INDEX idx_job_runs_job_id_started_at
    ON job_runs (job_id, started_at DESC);

CREATE INDEX idx_job_runs_status_started_at
    ON job_runs (status, started_at)
    WHERE status = 'running';


-- +goose Down
DROP TABLE IF EXISTS job_runs;
DROP TABLE IF EXISTS jobs;