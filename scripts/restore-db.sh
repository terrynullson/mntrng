#!/usr/bin/env sh
set -eu

if [ "${1:-}" = "" ]; then
  echo "Usage: scripts/restore-db.sh <backup.sql.gz>"
  exit 1
fi

BACKUP_FILE="$1"
if [ ! -f "${BACKUP_FILE}" ]; then
  echo "Backup file not found: ${BACKUP_FILE}"
  exit 1
fi

echo "Restoring DB from ${BACKUP_FILE}"
echo "Stopping API/worker/frontend before restore"
docker compose stop api worker frontend || true

gunzip -c "${BACKUP_FILE}" | docker compose exec -T postgres psql -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-hls_monitoring}" -v ON_ERROR_STOP=1

echo "Starting services"
docker compose start api worker frontend
echo "Restore complete"

