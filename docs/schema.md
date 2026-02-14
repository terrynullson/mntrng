# Database schema and migrations

## Apply migrations

Use PostgreSQL `psql` with `ON_ERROR_STOP` enabled.

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0001_baseline_schema.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0002_telegram_delivery_settings.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0003_preserve_company_audit_history.up.sql
```

## Roll back migrations

```bash
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

## Key constraints

- Multi-tenant isolation: every table except `companies` has `company_id NOT NULL`.
- Cross-tenant protection: child tables use composite foreign keys with `(id, company_id)`.
- Job idempotency: `check_jobs` has `UNIQUE(stream_id, planned_at)`.
- `check_results` immutability: `BEFORE UPDATE` trigger raises an exception.
- `audit_log` is append-only operational history and must persist after `companies` deletion; migration `0003_preserve_company_audit_history` removes cascade deletion for `audit_log`.
- Required indexes are present on `company_id`, `stream_id`, and `created_at` where applicable.
