# Deploy on Timeweb Cloud (Ubuntu 24.04)

## 1. Target profile

- VPS: 2 vCPU / 4 GB RAM / 50 GB NVMe
- OS: Ubuntu 24.04
- Domain: your public domain (example: `example.com`)
- TLS: Caddy automatic Let's Encrypt

## 2. DNS and network

1. Add A-records:
- `example.com` -> `<SERVER_PUBLIC_IP>`
- `www.example.com` -> `<SERVER_PUBLIC_IP>`

2. Open firewall:
- `22/tcp` (SSH)
- `80/tcp` (HTTP for ACME challenge)
- `443/tcp` (HTTPS)

## 3. Install Docker engine + compose plugin

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

## 4. Prepare project env

```bash
cp .env.example .env.prod
# или проще:
cp deploy/env.prod.example .env.prod
# полный шаблон с подсказками для первого деплоя:
cp deploy/env.prod.full.example .env.prod
```

Set at least:

```dotenv
APP_ENV=production
DOMAIN=example.com
CADDY_EMAIL=ops@example.com

POSTGRES_PASSWORD=<strong_random_password>
WORKER_METRICS_TOKEN=<strong_random_token>

AUTH_COOKIE_SECURE=true
AUTH_COOKIE_SAMESITE=strict
API_METRICS_PUBLIC=false
BOOTSTRAP_SEED_ENABLED=false
TRUST_PROXY_HEADERS=true

INTERNAL_API_BASE_URL=http://api:8080
NEXT_PUBLIC_API_BASE_URL=https://example.com
```

## 5. Start production stack

```bash
chmod +x scripts/deploy-prod.sh scripts/backup-db.sh scripts/restore-db.sh
ENV_FILE=.env.prod ./scripts/deploy-prod.sh
```

Create initial super-admin (replace values):

```bash
chmod +x scripts/bootstrap-superadmin.sh
ENV_FILE=.env.prod ./scripts/bootstrap-superadmin.sh admin@example.com root '<strong_password>'
```

## 6. Post-deploy verification

```bash
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml ps
docker compose --env-file .env.prod -f docker-compose.yml -f docker-compose.prod.yml logs -f caddy
curl -I https://example.com
curl -sS https://example.com/api/v1/health
```

Run app smoke:

```bash
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke-prod.ps1 \
  -ApiBaseUrl https://example.com \
  -Login root \
  -Password '<strong_password>' \
  -ExpectedRole super_admin
```

## 7. Backup and restore

Create backup:

```bash
chmod +x scripts/backup-db.sh scripts/restore-db.sh
BACKUP_DIR=./backups RETENTION_DAYS=14 ./scripts/backup-db.sh
```

Restore:

```bash
./scripts/restore-db.sh ./backups/pg_YYYYMMDDTHHMMSSZ.sql.gz
```

## 8. Recommended cron (daily backup at 03:30 UTC)

```bash
crontab -e
```

```cron
30 3 * * * cd /opt/hls-monitoring && BACKUP_DIR=./backups RETENTION_DAYS=14 ./scripts/backup-db.sh >> /var/log/hls-backup.log 2>&1
```
