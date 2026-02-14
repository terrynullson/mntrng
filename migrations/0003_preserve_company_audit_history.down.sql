BEGIN;

DELETE FROM audit_log AS al
WHERE NOT EXISTS (
    SELECT 1
    FROM companies AS c
    WHERE c.id = al.company_id
);

ALTER TABLE audit_log
    DROP CONSTRAINT IF EXISTS audit_log_company_fk;

ALTER TABLE audit_log
    ADD CONSTRAINT audit_log_company_fk
    FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE;

COMMIT;
