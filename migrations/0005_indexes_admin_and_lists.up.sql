BEGIN;

-- GET /admin/users: list with optional company_id, role, status; ORDER BY created_at DESC.
-- Supports filtered-by-company list (company_id present) with sort.
CREATE INDEX idx_users_company_id_created_at ON users (company_id, created_at DESC);

-- List check_results by company_id + stream_id; ORDER BY created_at DESC.
CREATE INDEX idx_check_results_company_stream_created ON check_results (company_id, stream_id, created_at DESC);

-- List streams by company_id + optional project_id filter.
CREATE INDEX idx_streams_company_project ON streams (company_id, project_id);

COMMIT;
