BEGIN;

CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id BIGINT,
    email TEXT NOT NULL,
    login TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('super_admin', 'company_admin', 'viewer')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT users_role_company_scope_ck
        CHECK (
            (role = 'super_admin' AND company_id IS NULL)
            OR
            (role IN ('company_admin', 'viewer') AND company_id IS NOT NULL)
        )
);

CREATE UNIQUE INDEX users_email_ci_uniq ON users ((LOWER(email)));
CREATE UNIQUE INDEX users_login_ci_uniq ON users ((LOWER(login)));
CREATE INDEX idx_users_company_id ON users (company_id);
CREATE INDEX idx_users_role_status ON users (role, status);
CREATE INDEX idx_users_created_at ON users (created_at);

CREATE TABLE registration_requests (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    company_id BIGINT NOT NULL,
    email TEXT NOT NULL,
    login TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    requested_role TEXT NOT NULL CHECK (requested_role IN ('company_admin', 'viewer')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    processed_by_user_id BIGINT,
    decision_reason TEXT,
    CONSTRAINT registration_requests_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT registration_requests_processed_by_fk
        FOREIGN KEY (processed_by_user_id) REFERENCES users(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX registration_requests_pending_email_ci_uniq
    ON registration_requests ((LOWER(email)))
    WHERE status = 'pending';

CREATE UNIQUE INDEX registration_requests_pending_login_ci_uniq
    ON registration_requests ((LOWER(login)))
    WHERE status = 'pending';

CREATE INDEX idx_registration_requests_company_id ON registration_requests (company_id);
CREATE INDEX idx_registration_requests_status_created_at ON registration_requests (status, created_at);
CREATE INDEX idx_registration_requests_processed_by_user_id ON registration_requests (processed_by_user_id);

CREATE TABLE user_telegram_links (
    user_id BIGINT PRIMARY KEY,
    telegram_user_id BIGINT NOT NULL UNIQUE,
    telegram_username TEXT,
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_telegram_links_user_fk
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_user_telegram_links_linked_at ON user_telegram_links (linked_at);

CREATE TABLE auth_sessions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    company_id BIGINT,
    access_token_hash TEXT NOT NULL UNIQUE,
    refresh_token_hash TEXT NOT NULL UNIQUE,
    access_expires_at TIMESTAMPTZ NOT NULL,
    refresh_expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT auth_sessions_user_fk
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT auth_sessions_company_fk
        FOREIGN KEY (company_id) REFERENCES companies(id) ON DELETE CASCADE,
    CONSTRAINT auth_sessions_refresh_after_access_ck
        CHECK (refresh_expires_at > access_expires_at)
);

CREATE INDEX idx_auth_sessions_user_id ON auth_sessions (user_id);
CREATE INDEX idx_auth_sessions_company_id ON auth_sessions (company_id);
CREATE INDEX idx_auth_sessions_access_expires_at ON auth_sessions (access_expires_at);
CREATE INDEX idx_auth_sessions_refresh_expires_at ON auth_sessions (refresh_expires_at);
CREATE INDEX idx_auth_sessions_active ON auth_sessions (user_id, revoked_at, refresh_expires_at);

COMMIT;
