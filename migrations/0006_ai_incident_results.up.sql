BEGIN;

CREATE TABLE ai_incident_results (
    job_id BIGINT PRIMARY KEY,
    company_id BIGINT NOT NULL,
    stream_id BIGINT NOT NULL,
    cause TEXT,
    summary TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ai_incident_results_check_result_fk
        FOREIGN KEY (job_id) REFERENCES check_results(job_id) ON DELETE CASCADE,
    CONSTRAINT ai_incident_results_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE
);

CREATE INDEX idx_ai_incident_results_company_id ON ai_incident_results (company_id);
CREATE INDEX idx_ai_incident_results_stream_id ON ai_incident_results (stream_id);

COMMIT;
