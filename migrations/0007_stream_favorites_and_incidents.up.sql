BEGIN;

-- stream_favorites: user's favorite and pinned streams (tenant-scope via user.company_id and stream.company_id)
CREATE TABLE stream_favorites (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    stream_id BIGINT NOT NULL,
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT stream_favorites_user_fk
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT stream_favorites_stream_fk
        FOREIGN KEY (stream_id) REFERENCES streams(id) ON DELETE CASCADE,
    CONSTRAINT stream_favorites_user_stream_uniq UNIQUE (user_id, stream_id)
);

CREATE INDEX idx_stream_favorites_user_id ON stream_favorites (user_id);
CREATE INDEX idx_stream_favorites_stream_id ON stream_favorites (stream_id);
CREATE INDEX idx_stream_favorites_user_pinned ON stream_favorites (user_id, is_pinned);

-- incidents: open/resolved per stream (tenant-scoped by company_id)
CREATE TABLE incidents (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id BIGINT NOT NULL,
    stream_id BIGINT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('open', 'resolved')),
    severity TEXT NOT NULL CHECK (severity IN ('warn', 'fail')),
    started_at TIMESTAMPTZ NOT NULL,
    last_event_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    fail_reason TEXT,
    sample_screenshot_path TEXT,
    last_check_id BIGINT,
    CONSTRAINT incidents_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT incidents_stream_fk
        FOREIGN KEY (stream_id, company_id) REFERENCES streams(id, company_id) ON DELETE CASCADE,
    CONSTRAINT incidents_last_check_fk
        FOREIGN KEY (last_check_id) REFERENCES check_results(id) ON DELETE SET NULL
);

CREATE INDEX idx_incidents_company_status ON incidents (company_id, status);
CREATE INDEX idx_incidents_company_severity ON incidents (company_id, severity);
CREATE INDEX idx_incidents_stream_status ON incidents (stream_id, status);
CREATE UNIQUE INDEX idx_incidents_one_open_per_stream
    ON incidents (stream_id, company_id) WHERE status = 'open';

COMMIT;
