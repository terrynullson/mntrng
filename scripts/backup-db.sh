#!/usr/bin/env sh
set -eu

BACKUP_DIR="${BACKUP_DIR:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_FILE="${BACKUP_DIR}/pg_${TIMESTAMP}.sql.gz"

mkdir -p "${BACKUP_DIR}"

echo "Creating PostgreSQL backup: ${OUT_FILE}"
docker compose exec -T postgres pg_dump -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-hls_monitoring}" | gzip -9 > "${OUT_FILE}"

echo "Pruning backups older than ${RETENTION_DAYS} days"
find "${BACKUP_DIR}" -type f -name "pg_*.sql.gz" -mtime +"${RETENTION_DAYS}" -delete

echo "Backup complete"

