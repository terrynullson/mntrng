BEGIN;

DROP INDEX IF EXISTS idx_ai_incident_results_stream_id;
DROP INDEX IF EXISTS idx_ai_incident_results_company_id;
DROP TABLE IF EXISTS ai_incident_results;

COMMIT;
