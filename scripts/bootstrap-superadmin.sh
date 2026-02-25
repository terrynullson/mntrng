#!/usr/bin/env sh
set -eu

ENV_FILE="${ENV_FILE:-.env.prod}"

if [ $# -ne 3 ]; then
  echo "Usage: ENV_FILE=.env.prod scripts/bootstrap-superadmin.sh <email> <login> <password>"
  exit 1
fi

if [ ! -f "${ENV_FILE}" ]; then
  echo "Missing ${ENV_FILE}"
  exit 1
fi

EMAIL="$1"
LOGIN="$2"
PASSWORD="$3"

escape_sql() {
  printf "%s" "$1" | sed "s/'/''/g"
}

EMAIL_SQL="$(escape_sql "${EMAIL}")"
LOGIN_SQL="$(escape_sql "${LOGIN}")"
PASSWORD_SQL="$(escape_sql "${PASSWORD}")"

SQL=$(cat <<EOF
CREATE EXTENSION IF NOT EXISTS pgcrypto;
INSERT INTO users (company_id, email, login, password_hash, role, status)
VALUES (
  NULL,
  '${EMAIL_SQL}',
  '${LOGIN_SQL}',
  crypt('${PASSWORD_SQL}', gen_salt('bf')),
  'super_admin',
  'active'
)
ON CONFLICT (login) DO UPDATE
SET email = EXCLUDED.email,
    password_hash = EXCLUDED.password_hash,
    role = 'super_admin',
    status = 'active',
    updated_at = NOW();
EOF
)

docker compose --env-file "${ENV_FILE}" -f docker-compose.yml -f docker-compose.prod.yml exec -T postgres \
  psql -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-hls_monitoring}" -v ON_ERROR_STOP=1 -c "${SQL}"

echo "super_admin bootstrap complete for login=${LOGIN}"

