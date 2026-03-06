#!/usr/bin/env sh
set -eu

ROOT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$ROOT_DIR"

ENV_FILE="${ENV_FILE:-.env.prod}"
TARGET_COMMIT="${1:-}"

if [ -z "$TARGET_COMMIT" ]; then
  if [ -f .deploy_prev_commit ]; then
    TARGET_COMMIT="$(cat .deploy_prev_commit)"
  else
    echo "No target commit provided and .deploy_prev_commit not found"
    echo "Usage: scripts/rollback.sh <commit>"
    exit 1
  fi
fi

if [ ! -f "$ENV_FILE" ]; then
  echo "Missing $ENV_FILE"
  exit 1
fi

echo "[rollback] switching to commit $TARGET_COMMIT"
git fetch --all --prune
git checkout --detach "$TARGET_COMMIT"
printf '%s\n' "$TARGET_COMMIT" > .deploy_current_commit

echo "[rollback] rebuilding and starting containers"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f docker-compose.prod.yml up -d --build --remove-orphans

echo "[rollback] containers status"
docker compose --env-file "$ENV_FILE" -f docker-compose.yml -f docker-compose.prod.yml ps

echo "[rollback] done. current commit=$TARGET_COMMIT"
echo "[rollback] tip: return to branch later: git checkout master"
