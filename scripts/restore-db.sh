#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

if [ "${1:-}" = "" ]; then
  echo "Usage: scripts/restore-db.sh <backup.sql.gz>"
  echo "On production use: ENV_FILE=.env.prod ./scripts/restore-db.sh <file>"
  exit 1
fi

BACKUP_FILE="$1"
if [ ! -f "${BACKUP_FILE}" ]; then
  echo "Backup file not found: ${BACKUP_FILE}"
  exit 1
fi

ENV_FILE="${ENV_FILE:-}"
COMPOSE_ARGS=""
if [ -n "$ENV_FILE" ] && [ -f "$ENV_FILE" ]; then
  set -a
  . "$ENV_FILE"
  set +a
  export ENV_FILE
  COMPOSE_ARGS="--env-file $ENV_FILE -f docker-compose.yml -f docker-compose.prod.yml"
fi

echo "Restoring DB from ${BACKUP_FILE}"
echo "Stopping API/worker/frontend before restore"
if [ -n "$COMPOSE_ARGS" ]; then
  docker compose $COMPOSE_ARGS stop api worker frontend scheduler || true
else
  docker compose stop api worker frontend scheduler || true
fi

if [ -n "$COMPOSE_ARGS" ]; then
  gunzip -c "${BACKUP_FILE}" | docker compose $COMPOSE_ARGS exec -T postgres psql -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-hls_monitoring}" -v ON_ERROR_STOP=1
else
  gunzip -c "${BACKUP_FILE}" | docker compose exec -T postgres psql -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-hls_monitoring}" -v ON_ERROR_STOP=1
fi

echo "Starting services"
if [ -n "$COMPOSE_ARGS" ]; then
  docker compose $COMPOSE_ARGS start api worker frontend scheduler
else
  docker compose start api worker frontend scheduler
fi
echo "Restore complete"

