#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

BACKUP_DIR="${BACKUP_DIR:-./backups}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"
ENV_FILE="${ENV_FILE:-}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT_FILE="${BACKUP_DIR}/pg_${TIMESTAMP}.sql.gz"

# Optional: use same env and compose as prod deploy; load POSTGRES_* for pg_dump args
COMPOSE_ARGS=""
if [ -n "$ENV_FILE" ] && [ -f "$ENV_FILE" ]; then
  set -a
  . "$ENV_FILE"
  set +a
  export ENV_FILE
  COMPOSE_ARGS="--env-file $ENV_FILE -f docker-compose.yml -f docker-compose.prod.yml"
fi

mkdir -p "${BACKUP_DIR}"

echo "Creating PostgreSQL backup: ${OUT_FILE}"
if [ -n "$COMPOSE_ARGS" ]; then
  docker compose $COMPOSE_ARGS exec -T postgres pg_dump -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-hls_monitoring}" | gzip -9 > "${OUT_FILE}"
else
  docker compose exec -T postgres pg_dump -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-hls_monitoring}" | gzip -9 > "${OUT_FILE}"
fi

echo "Pruning backups older than ${RETENTION_DAYS} days"
find "${BACKUP_DIR}" -type f -name "pg_*.sql.gz" -mtime +"${RETENTION_DAYS}" -delete

echo "Backup complete"

