BEGIN;

DROP INDEX IF EXISTS idx_embed_whitelist_company_enabled;
DROP INDEX IF EXISTS idx_embed_whitelist_company_id;
DROP TABLE IF EXISTS embed_whitelist;

ALTER TABLE streams
    DROP COLUMN IF EXISTS source_url,
    DROP COLUMN IF EXISTS source_type;

COMMIT;
