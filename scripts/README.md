# Dev/test scripts (PowerShell)

Используются для тестовой БД и пайплайна скриншотов. Требуют PostgreSQL (psql в PATH для миграций и создания БД).

| Скрипт | Назначение |
|--------|------------|
| **ensure-test-db.ps1** | Создаёт БД `hls_monitoring_test` по env_dev (если ещё нет). |
| **run-migrations-dev.ps1** | Загружает env_dev и применяет миграции к тестовой БД. |
| **run-api-dev.ps1** | Загружает env_dev и запускает API (`go run ./cmd/api/`). |

Подробнее: [docs/screenshot_automation.md](../docs/screenshot_automation.md).
