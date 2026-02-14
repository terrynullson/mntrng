BEGIN;

ALTER TABLE audit_log
    DROP CONSTRAINT IF EXISTS audit_log_company_fk;

COMMIT;
