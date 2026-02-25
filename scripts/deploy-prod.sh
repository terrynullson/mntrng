#!/usr/bin/env sh
set -eu

ENV_FILE="${ENV_FILE:-.env.prod}"

if [ ! -f "${ENV_FILE}" ]; then
  echo "Missing ${ENV_FILE}. Create it from deploy/env.prod.example"
  exit 1
fi

echo "Deploying production stack with ${ENV_FILE}"
docker compose --env-file "${ENV_FILE}" -f docker-compose.yml -f docker-compose.prod.yml up -d --build
docker compose --env-file "${ENV_FILE}" -f docker-compose.yml -f docker-compose.prod.yml ps

echo "Deploy complete"

