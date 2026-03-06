# Scripts

## Production / ops (shell)

| Скрипт | Назначение |
|--------|------------|
| **deploy-prod.sh** | Подъём prod-стека: `ENV_FILE=.env.prod ./scripts/deploy-prod.sh`. |
| **deploy.sh** | Деплой с записью коммита для отката (пишет `.deploy_prev_commit`, `.deploy_current_commit`). |
| **rollback.sh** | Откат на предыдущий коммит или заданный: `ENV_FILE=.env.prod ./scripts/rollback.sh [commit]`. |
| **backup-db.sh** | Бэкап PostgreSQL. На проде: `ENV_FILE=.env.prod BACKUP_DIR=... RETENTION_DAYS=14 ./scripts/backup-db.sh`. |
| **restore-db.sh** | Восстановление БД из файла: `ENV_FILE=.env.prod ./scripts/restore-db.sh <file.sql.gz>`. |
| **bootstrap-superadmin.sh** | Создание первого супер-админа. |

Подробнее: [docs/ops_index.md](../docs/ops_index.md), [docs/backup_restore.md](../docs/backup_restore.md), [docs/rollback_runbook.md](../docs/rollback_runbook.md).

## Dev/test (PowerShell)

Используются для тестовой БД и пайплайна скриншотов. Требуют PostgreSQL (psql в PATH для миграций и создания БД).

| Скрипт | Назначение |
|--------|------------|
| **ensure-test-db.ps1** | Создаёт БД `hls_monitoring_test` по env_dev (если ещё нет). |
| **run-migrations-dev.ps1** | Загружает env_dev и применяет миграции к тестовой БД. |
| **run-api-dev.ps1** | Загружает env_dev и запускает API (`go run ./cmd/api/`). |
| **backup_db.ps1** | Резервная копия БД (pg_dump). Читает DATABASE_URL из .env, создаёт `backups/hls_monitoring_YYYYMMDD_HHMMSS.sql`. |
| **rollback_migrations.ps1** | Откат миграций (0006→0001). Читает DATABASE_URL из .env. Использовать с осторожностью. |

Подробнее: [docs/screenshot_automation.md](../docs/screenshot_automation.md).
