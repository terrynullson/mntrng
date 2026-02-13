BEGIN;

DROP TRIGGER IF EXISTS check_results_immutable_update ON check_results;
DROP FUNCTION IF EXISTS prevent_check_results_update();

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS alert_state;
DROP TABLE IF EXISTS check_results;
DROP TABLE IF EXISTS check_jobs;
DROP TABLE IF EXISTS streams;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS companies;

COMMIT;