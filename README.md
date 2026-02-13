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

## Остановка локального окружения

```bash
docker compose down
```

Удалить volume-данные:

```bash
docker compose down -v
```
