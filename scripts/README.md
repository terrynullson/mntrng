# Dev/test scripts (PowerShell)

Используются для тестовой БД и пайплайна скриншотов. Требуют PostgreSQL (psql в PATH для миграций и создания БД).

| Скрипт | Назначение |
|--------|------------|
| **ensure-test-db.ps1** | Создаёт БД `hls_monitoring_test` по env_dev (если ещё нет). |
| **run-migrations-dev.ps1** | Загружает env_dev и применяет миграции к тестовой БД. |
| **run-api-dev.ps1** | Загружает env_dev и запускает API (`go run ./cmd/api/`). |
| **backup_db.ps1** | Резервная копия БД (pg_dump). Читает DATABASE_URL из .env, создаёт `backups/hls_monitoring_YYYYMMDD_HHMMSS.sql`. |
| **rollback_migrations.ps1** | Откат миграций (0006→0001). Читает DATABASE_URL из .env. Использовать с осторожностью. |

Подробнее: [docs/screenshot_automation.md](../docs/screenshot_automation.md).
