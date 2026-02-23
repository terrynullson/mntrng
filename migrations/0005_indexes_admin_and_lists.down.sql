BEGIN;

DROP INDEX IF EXISTS idx_streams_company_project;
DROP INDEX IF EXISTS idx_check_results_company_stream_created;
DROP INDEX IF EXISTS idx_users_company_id_created_at;

COMMIT;
