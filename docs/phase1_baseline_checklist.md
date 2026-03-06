# Phase 1 Baseline Checklist

Короткий checklist для локального старта и smoke-проверки без архитектурных изменений.

## 1) Локальный старт

1. Скопировать env:
   - `cp .env.example .env` (Linux/macOS) или `Copy-Item .env.example .env` (PowerShell).
2. Заполнить минимум:
   - `POSTGRES_PASSWORD` (required, sensitive).
   - `WORKER_METRICS_TOKEN` (для production обязателен; локально можно оставить пустым).
   - при включении scheduler: `SCHEDULER_ACCESS_TOKEN`.
3. Поднять стек:
   - `docker compose up --build -d`

## 2) Обязательные env по режимам

- **Local/dev mandatory:** `POSTGRES_PASSWORD`.
- **Production mandatory:** `POSTGRES_PASSWORD`, `WORKER_METRICS_TOKEN` (strong random), `CORS_ALLOWED_ORIGINS`, корректные cookie/TLS настройки.
- **If enabled:**  
  - scheduler: `SCHEDULER_ACCESS_TOKEN`;  
  - telegram alerts: `TELEGRAM_BOT_TOKEN_DEFAULT` и `SUPER_ADMIN_TELEGRAM_CHAT_ID`;  
  - AI provider (future): `AI_INCIDENT_API_KEY` (сейчас analyzer stubbed).

## 3) Обязательные команды проверки

Из корня репозитория:

```bash
go test ./...
```

```bash
docker compose ps
```

```bash
curl -sS http://localhost:8080/api/v1/health
curl -sS http://localhost:8080/api/v1/ready
curl -sS http://localhost:9091/health
```

## 4) Проверка, что сервисы живы

- **API:** `GET /api/v1/health` -> `200`.
- **API readiness:** `GET /api/v1/ready` -> `200` при доступной БД.
- **Worker:** `GET http://localhost:${WORKER_METRICS_PORT:-9091}/health` -> `200`.
- **Frontend:** открыть `http://localhost:3000/login`.
- **Scheduler (если включен):** в логах есть `scheduler started` и `scheduler cycle`.

## 5) Минимальный smoke checklist

- [ ] `docker compose ps` показывает `api`, `worker`, `frontend` в состоянии running/healthy.
- [ ] API health/readiness отвечает без ошибок.
- [ ] Worker health отвечает без ошибок.
- [ ] Frontend login страница доступна.
- [ ] `go test ./...` проходит в чистом окружении.
- [ ] В env нет test/demo секретов.

## 6) Phase 1 complete criteria

Phase 1 считается завершенной, если:

1. В `go.mod` корректный module path репозитория.
2. В репозитории нет локальных бинарников/build artifacts.
3. `.gitignore` закрывает типичный локальный мусор.
4. Env templates явно разделяют local/dev и production usage, без двусмысленных секретов.
5. README и docs честно отражают текущее состояние AI (stub).
6. Есть воспроизводимый baseline запуска и проверки.
