# HLS Monitoring Platform Bootstrap

Минимальный инфраструктурный каркас репозитория и Docker Compose окружения.

## Состав сервисов

- `api` (Go)
- `worker` (Go)
- `postgres` (PostgreSQL)
- `redis` (Redis)
- `frontend` (Next.js + TypeScript)

## Переменные окружения

1. Создать локальный env-файл:

```bash
cp .env.example .env
```

2. При необходимости изменить значения в `.env`:

- `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_PORT`
- `API_PORT`, `REDIS_ADDR`, `WORKER_HEARTBEAT_SEC`
- `FRONTEND_PORT`, `NEXT_PUBLIC_API_BASE_URL`

Файл `.env` не добавляется в git (трекается только `.env.example`).

## Запуск локального окружения

```bash
docker compose up --build -d
```

Проверка API:

```bash
curl http://localhost:8080/api/v1/health
```

## Companies CRUD smoke-check

1. Применить baseline-миграцию в PostgreSQL:

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0001_baseline_schema.up.sql
```

2. Проверить Companies API:

```bash
# create
curl -sS -X POST http://localhost:8080/api/v1/companies \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme Media"}'

# list
curl -sS http://localhost:8080/api/v1/companies

# get by id
curl -sS http://localhost:8080/api/v1/companies/1

# patch
curl -sS -X PATCH http://localhost:8080/api/v1/companies/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"Acme Media Group"}'

# delete
curl -sS -X DELETE http://localhost:8080/api/v1/companies/1 -i
```

## Projects CRUD smoke-check

Требуется существующий `company_id` (например, `1`).

```bash
# create
curl -sS -X POST http://localhost:8080/api/v1/companies/1/projects \
  -H "Content-Type: application/json" \
  -d '{"name":"News Channel"}'

# list
curl -sS http://localhost:8080/api/v1/companies/1/projects

# get by id
curl -sS http://localhost:8080/api/v1/companies/1/projects/1

# patch
curl -sS -X PATCH http://localhost:8080/api/v1/companies/1/projects/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"Updated Project Name"}'

# delete
curl -sS -X DELETE http://localhost:8080/api/v1/companies/1/projects/1 -i
```

## Streams CRUD smoke-check

Требуются существующие `company_id` и `project_id` (например, `1` и `1`).

```bash
# create in project
curl -sS -X POST http://localhost:8080/api/v1/companies/1/projects/1/streams \
  -H "Content-Type: application/json" \
  -d '{"name":"Primary HLS","url":"https://cdn.example.com/live/index.m3u8","is_active":true}'

# list by company (optional filters: project_id, is_active)
curl -sS "http://localhost:8080/api/v1/companies/1/streams?project_id=1&is_active=true"

# get by id
curl -sS http://localhost:8080/api/v1/companies/1/streams/1

# patch
curl -sS -X PATCH http://localhost:8080/api/v1/companies/1/streams/1 \
  -H "Content-Type: application/json" \
  -d '{"name":"Backup HLS","is_active":false}'

# delete
curl -sS -X DELETE http://localhost:8080/api/v1/companies/1/streams/1 -i
```

## Check jobs API smoke-check

Требуется существующий `company_id` и `stream_id` (например, `1` и `1`).

```bash
# enqueue
curl -sS -X POST http://localhost:8080/api/v1/companies/1/streams/1/check-jobs \
  -H "Content-Type: application/json" \
  -d '{"planned_at":"2026-02-13T10:05:00Z"}'

# get status by job id
curl -sS http://localhost:8080/api/v1/companies/1/check-jobs/1

# history for stream (optional filters: status, from, to)
curl -sS "http://localhost:8080/api/v1/companies/1/streams/1/check-jobs?status=queued&from=2026-02-13T00:00:00Z&to=2026-02-14T00:00:00Z"
```

## Check results API smoke-check

Требуется существующий `company_id`, `stream_id`, `result_id` и `job_id`.

```bash
# get result by id
curl -sS http://localhost:8080/api/v1/companies/1/check-results/1

# list results by stream (optional filters: status, from, to)
curl -sS "http://localhost:8080/api/v1/companies/1/streams/1/check-results?status=FAIL&from=2026-02-13T00:00:00Z&to=2026-02-14T00:00:00Z"

# get result by job id
curl -sS http://localhost:8080/api/v1/companies/1/check-jobs/1/result
```

## Остановка локального окружения

```bash
docker compose down
```

Удалить volume-данные:

```bash
docker compose down -v
```
