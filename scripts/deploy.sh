#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

ENV_FILE="${ENV_FILE:-.env.prod}"
BRANCH="${BRANCH:-master}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:8080/api/v1/health}"

if [ ! -f "$ENV_FILE" ]; then
  echo "Missing $ENV_FILE"
  echo "Create it first (example: cp .env.example .env.prod)"
  exit 1
fi

CURRENT_COMMIT="$(git rev-parse --short HEAD)"
printf '%s\n' "$CURRENT_COMMIT" > .deploy_prev_commit

echo "[deploy] pulling latest from origin/$BRANCH"
git fetch --all --prune
git checkout "$BRANCH"
git pull --ff-only origin "$BRANCH"

NEW_COMMIT="$(git rev-parse --short HEAD)"
printf '%s\n' "$NEW_COMMIT" > .deploy_current_commit

echo "[deploy] building and starting containers"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f docker-compose.prod.yml up -d --build --remove-orphans

echo "[deploy] containers status"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f docker-compose.prod.yml ps

echo "[deploy] health check: $HEALTH_URL"
curl -fsS "$HEALTH_URL" >/dev/null
echo "[deploy] OK. commit=$NEW_COMMIT"
