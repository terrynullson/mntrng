BEGIN;

ALTER TABLE streams
    ADD COLUMN source_type TEXT NOT NULL DEFAULT 'HLS'
        CHECK (source_type IN ('HLS', 'EMBED')),
    ADD COLUMN source_url TEXT;

UPDATE streams
SET source_url = url
WHERE source_url IS NULL;

ALTER TABLE streams
    ALTER COLUMN source_url SET NOT NULL;

CREATE TABLE embed_whitelist (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id BIGINT NOT NULL,
    domain TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id BIGINT,
    CONSTRAINT embed_whitelist_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT embed_whitelist_created_by_user_fk
        FOREIGN KEY (created_by_user_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT embed_whitelist_company_domain_uniq UNIQUE (company_id, domain)
);

CREATE INDEX idx_embed_whitelist_company_id ON embed_whitelist (company_id);
CREATE INDEX idx_embed_whitelist_company_enabled ON embed_whitelist (company_id, enabled);

COMMIT;
