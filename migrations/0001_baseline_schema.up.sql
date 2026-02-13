BEGIN;

CREATE TABLE companies (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE projects (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT projects_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT projects_company_name_key UNIQUE (company_id, name),
    CONSTRAINT projects_id_company_key UNIQUE (id, company_id)
);

CREATE TABLE streams (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id BIGINT NOT NULL,
    project_id BIGINT NOT NULL,
    name TEXT NOT NULL,
    url TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT streams_project_fk
        FOREIGN KEY (project_id, company_id) REFERENCES projects(id, company_id) ON DELETE CASCADE,
    CONSTRAINT streams_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT streams_company_project_name_key UNIQUE (company_id, project_id, name),
    CONSTRAINT streams_id_company_key UNIQUE (id, company_id)
);

CREATE TABLE check_jobs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id BIGINT NOT NULL,
    stream_id BIGINT NOT NULL,
    planned_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'done', 'failed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    error_message TEXT,
    CONSTRAINT check_jobs_stream_fk
        FOREIGN KEY (stream_id, company_id) REFERENCES streams(id, company_id) ON DELETE CASCADE,
    CONSTRAINT check_jobs_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT check_jobs_stream_planned_unique UNIQUE (stream_id, planned_at),
    CONSTRAINT check_jobs_id_company_key UNIQUE (id, company_id)
);

CREATE TABLE check_results (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id BIGINT NOT NULL,
    job_id BIGINT NOT NULL,
    stream_id BIGINT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ok', 'warn', 'fail')),
    checks JSONB NOT NULL DEFAULT '{}'::JSONB,
    screenshot_path TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT check_results_job_fk
        FOREIGN KEY (job_id, company_id) REFERENCES check_jobs(id, company_id) ON DELETE CASCADE,
    CONSTRAINT check_results_stream_fk
        FOREIGN KEY (stream_id, company_id) REFERENCES streams(id, company_id) ON DELETE CASCADE,
    CONSTRAINT check_results_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT check_results_job_unique UNIQUE (job_id)
);

CREATE TABLE alert_state (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id BIGINT NOT NULL,
    stream_id BIGINT NOT NULL,
    fail_streak INTEGER NOT NULL DEFAULT 0 CHECK (fail_streak >= 0),
    cooldown_until TIMESTAMPTZ,
    last_alert_at TIMESTAMPTZ,
    last_status TEXT CHECK (last_status IN ('ok', 'warn', 'fail')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT alert_state_stream_fk
        FOREIGN KEY (stream_id, company_id) REFERENCES streams(id, company_id) ON DELETE CASCADE,
    CONSTRAINT alert_state_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT alert_state_stream_unique UNIQUE (stream_id)
);

CREATE TABLE audit_log (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id BIGINT NOT NULL,
    actor_type TEXT NOT NULL,
    actor_id TEXT,
    entity_type TEXT NOT NULL,
    entity_id BIGINT NOT NULL,
    action TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT audit_log_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE
);

CREATE OR REPLACE FUNCTION prevent_check_results_update()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'check_results rows are immutable';
END;
$$;

CREATE TRIGGER check_results_immutable_update
BEFORE UPDATE ON check_results
FOR EACH ROW
EXECUTE FUNCTION prevent_check_results_update();

CREATE INDEX idx_projects_company_id ON projects (company_id);
CREATE INDEX idx_projects_created_at ON projects (created_at);

CREATE INDEX idx_streams_company_id ON streams (company_id);
CREATE INDEX idx_streams_created_at ON streams (created_at);

CREATE INDEX idx_check_jobs_company_id ON check_jobs (company_id);
CREATE INDEX idx_check_jobs_stream_id ON check_jobs (stream_id);
CREATE INDEX idx_check_jobs_created_at ON check_jobs (created_at);

CREATE INDEX idx_check_results_company_id ON check_results (company_id);
CREATE INDEX idx_check_results_stream_id ON check_results (stream_id);
CREATE INDEX idx_check_results_created_at ON check_results (created_at);

CREATE INDEX idx_alert_state_company_id ON alert_state (company_id);
CREATE INDEX idx_alert_state_stream_id ON alert_state (stream_id);
CREATE INDEX idx_alert_state_created_at ON alert_state (created_at);

CREATE INDEX idx_audit_log_company_id ON audit_log (company_id);
CREATE INDEX idx_audit_log_created_at ON audit_log (created_at);

COMMIT;