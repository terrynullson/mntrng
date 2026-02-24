BEGIN;

DROP INDEX IF EXISTS idx_incidents_one_open_per_stream;
DROP INDEX IF EXISTS idx_incidents_stream_status;
DROP INDEX IF EXISTS idx_incidents_company_severity;
DROP INDEX IF EXISTS idx_incidents_company_status;
DROP TABLE IF EXISTS incidents;

DROP INDEX IF EXISTS idx_stream_favorites_user_pinned;
DROP INDEX IF EXISTS idx_stream_favorites_stream_id;
DROP INDEX IF EXISTS idx_stream_favorites_user_id;
DROP TABLE IF EXISTS stream_favorites;

COMMIT;
