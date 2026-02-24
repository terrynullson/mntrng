BEGIN;

ALTER TABLE incidents
    DROP COLUMN IF EXISTS screenshot_taken_at,
    DROP COLUMN IF EXISTS diag_details,
    DROP COLUMN IF EXISTS diag_code;

COMMIT;
