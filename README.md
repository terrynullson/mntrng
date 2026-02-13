# HLS Monitoring Platform Bootstrap

Минимальный инфраструктурный каркас репозитория и Docker Compose окружения.

## Состав сервисов

- `api` (Go)
- `worker` (Go)
- `postgres` (PostgreSQL)
- `redis` (Redis)
- `frontend` (Next.js + TypeScript)

## Запуск локального окружения

```bash
docker compose up --build -d
```

Проверка API:

```bash
curl http://localhost:8080/healthz
```

## Остановка локального окружения

```bash
docker compose down
```

Удалить volume-данные:

```bash
docker compose down -v
```