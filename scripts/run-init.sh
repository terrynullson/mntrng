#!/bin/sh
# Run migrations then seed. Used in Docker init container. Idempotent: re-run exits 0.
set -e
if [ -z "$DATABASE_URL" ]; then
  echo "DATABASE_URL is required"
  exit 1
fi
for f in /migrations/0001_baseline_schema.up.sql \
         /migrations/0002_telegram_delivery_settings.up.sql \
         /migrations/0003_preserve_company_audit_history.up.sql \
         /migrations/0004_auth_and_registration.up.sql \
         /migrations/0005_indexes_admin_and_lists.up.sql \
         /migrations/0006_ai_incident_results.up.sql; do
  echo "Applying $(basename "$f")..."
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$f" || true
done
echo "Running seed..."
/app/seed || true
echo "Init done."
exit 0
