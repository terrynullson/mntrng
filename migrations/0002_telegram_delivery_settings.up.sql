BEGIN;

CREATE TABLE telegram_delivery_settings (
    company_id BIGINT PRIMARY KEY,
    is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    chat_id TEXT NOT NULL,
    send_recovered BOOLEAN NOT NULL DEFAULT FALSE,
    bot_token_ref TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT telegram_delivery_settings_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT telegram_delivery_settings_chat_id_not_empty
        CHECK (LENGTH(TRIM(chat_id)) > 0)
);

CREATE INDEX idx_telegram_delivery_settings_created_at ON telegram_delivery_settings (created_at);

COMMIT;
