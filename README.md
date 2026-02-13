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

## Остановка локального окружения

```bash
docker compose down
```

Удалить volume-данные:

```bash
docker compose down -v
```
