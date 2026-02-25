#!/bin/sh
# Run migrations then seed. Used in Docker init container.
# Idempotent via schema_migrations table.
set -eu

if [ -z "$DATABASE_URL" ]; then
  echo "DATABASE_URL is required"
  exit 1
fi

psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);"

for f in /migrations/0001_baseline_schema.up.sql \
         /migrations/0002_telegram_delivery_settings.up.sql \
         /migrations/0003_preserve_company_audit_history.up.sql \
         /migrations/0004_auth_and_registration.up.sql \
         /migrations/0005_indexes_admin_and_lists.up.sql \
         /migrations/0006_ai_incident_results.up.sql \
         /migrations/0007_stream_favorites_and_incidents.up.sql \
         /migrations/0008_embed_whitelist_and_stream_sources.up.sql \
         /migrations/0009_incident_diagnostics_lite.up.sql; do
  version="$(basename "$f")"
  applied="$(psql "$DATABASE_URL" -At -c "SELECT 1 FROM schema_migrations WHERE version = '$version' LIMIT 1;")"
  if [ "$applied" = "1" ]; then
    echo "Skipping $version (already applied)"
    continue
  fi

  echo "Applying $version..."
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$f"
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations (version) VALUES ('$version');"
done

echo "Running seed..."
/app/seed
echo "Init done."
