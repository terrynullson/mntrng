# Database schema and migrations

## Apply migrations

Use PostgreSQL `psql` with `ON_ERROR_STOP` enabled.

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0001_baseline_schema.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0002_telegram_delivery_settings.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0003_preserve_company_audit_history.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0004_auth_and_registration.up.sql
```

## Roll back migrations

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0004_auth_and_registration.down.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0003_preserve_company_audit_history.down.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0002_telegram_delivery_settings.down.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0001_baseline_schema.down.sql
```

For multiple migrations, apply `*.up.sql` in lexical order and roll back using matching `*.down.sql` in reverse order.

## Telegram delivery settings table

Migration `0002_telegram_delivery_settings` adds `telegram_delivery_settings` as a tenant-scoped company-level settings table.

- `company_id` is both `PRIMARY KEY` and `FOREIGN KEY` to `companies(id)` (`ON DELETE CASCADE`), guaranteeing one settings row per tenant company.
- Delivery flags are explicit booleans: `is_enabled` and `send_recovered`.
- `chat_id` is required (`NOT NULL`) and constrained by `CHECK (length(trim(chat_id)) > 0)`.
- Secret material is not stored directly in this table; only `bot_token_ref` (reference/alias) is stored.
- Reproducible index: `idx_telegram_delivery_settings_created_at` on `created_at`.

## Authentication and registration tables

Migration `0004_auth_and_registration` adds authentication, controlled registration, Telegram identity links, and revokable session storage.

### users

- Tenant/user identity table with RBAC role and active/disabled status.
- `role` check: `super_admin`, `company_admin`, `viewer`.
- `status` check: `active`, `disabled`.
- Company scope rule:
  - `super_admin` requires `company_id IS NULL`.
  - `company_admin/viewer` require `company_id IS NOT NULL`.
- Case-insensitive uniqueness:
  - `users_email_ci_uniq` on `LOWER(email)`.
  - `users_login_ci_uniq` on `LOWER(login)`.

### registration_requests

- Public registration queue with moderation lifecycle.
- `requested_role` check: `company_admin`, `viewer`.
- `status` check: `pending`, `approved`, `rejected`.
- Stores password hash only (`password_hash`), never plain password.
- Case-insensitive pending uniqueness:
  - `registration_requests_pending_email_ci_uniq`.
  - `registration_requests_pending_login_ci_uniq`.

### user_telegram_links

- One active Telegram link per user (`user_id` PK).
- One Telegram account cannot be linked to multiple users (`telegram_user_id` unique).
- Used for Telegram login flow after approved+active user check.

### auth_sessions

- Revokable token sessions for access/refresh model.
- Stores token hashes only (`access_token_hash`, `refresh_token_hash`), not raw tokens.
- Supports logout/revoke via `revoked_at`.
- Expiry constraints:
  - `access_expires_at` and `refresh_expires_at` required.
  - `refresh_expires_at > access_expires_at`.

## Key constraints

- Multi-tenant isolation: every table except `companies` has `company_id NOT NULL`.
- Cross-tenant protection: child tables use composite foreign keys with `(id, company_id)`.
- Job idempotency: `check_jobs` has `UNIQUE(stream_id, planned_at)`.
- `check_results` immutability: `BEFORE UPDATE` trigger raises an exception.
- `audit_log` is append-only operational history and must persist after `companies` deletion; migration `0003_preserve_company_audit_history` removes cascade deletion for `audit_log`.
- Required indexes are present on `company_id`, `stream_id`, and `created_at` where applicable.
- Auth/audit support indexes include:
  - `idx_users_company_id`, `idx_users_role_status`.
  - `idx_registration_requests_status_created_at`, `idx_registration_requests_company_id`.
  - `idx_auth_sessions_user_id`, `idx_auth_sessions_active`.
