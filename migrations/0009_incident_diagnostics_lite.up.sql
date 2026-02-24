BEGIN;

ALTER TABLE incidents
    ADD COLUMN diag_code TEXT,
    ADD COLUMN diag_details JSONB,
    ADD COLUMN screenshot_taken_at TIMESTAMPTZ;

COMMIT;
