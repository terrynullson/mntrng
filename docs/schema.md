# Database schema and migrations

## Apply baseline migration

Use PostgreSQL `psql` with `ON_ERROR_STOP` enabled.

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0001_baseline_schema.up.sql
```

## Roll back baseline migration

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0001_baseline_schema.down.sql
```

For multiple migrations, apply `*.up.sql` in lexical order and roll back using matching `*.down.sql` in reverse order.

## Key constraints

- Multi-tenant isolation: every table except `companies` has `company_id NOT NULL`.
- Cross-tenant protection: child tables use composite foreign keys with `(id, company_id)`.
- Job idempotency: `check_jobs` has `UNIQUE(stream_id, planned_at)`.
- `check_results` immutability: `BEFORE UPDATE` trigger raises an exception.
- Required indexes are present on `company_id`, `stream_id`, and `created_at` where applicable.