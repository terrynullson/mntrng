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
- `WORKER_JOB_TIMEOUT_SEC`, `WORKER_DB_RETRY_MAX`, `WORKER_DB_RETRY_BACKOFF_MS`
- `PLAYLIST_TIMEOUT_MS`, `SEGMENT_TIMEOUT_MS`, `SEGMENTS_SAMPLE_COUNT`, `FRESHNESS_WARN_SEC`, `FRESHNESS_FAIL_SEC`, `FREEZE_WARN_SEC`, `FREEZE_FAIL_SEC`, `EFFECTIVE_BITRATE_WARN_RATIO`, `EFFECTIVE_BITRATE_FAIL_RATIO`
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

## Worker skeleton smoke-check

1. Запустить worker:

```bash
go run ./cmd/worker
```

2. Поставить job в очередь (см. раздел Check jobs API smoke-check), затем несколько раз проверить историю:

```bash
curl -sS "http://localhost:8080/api/v1/companies/1/streams/1/check-jobs"
```

Ожидаемый lifecycle для skeleton: `queued -> running -> done` (или `failed`, если сработал timeout/error path).

В текущем checker-подшаге worker считает:
- `playlist` availability check
- `freshness` check (по `#EXT-X-PROGRAM-DATE-TIME`)
- `segments` availability check по последним `N` сегментам из playlist
- `freeze` check по `#EXT-X-PROGRAM-DATE-TIME` (оценка максимального freeze-интервала)
- `declared_bitrate` check по тегам `#EXT-X-STREAM-INF` (`BANDWIDTH` / `AVERAGE-BANDWIDTH`)
- `effective_bitrate` check по уже скачанным сегментам из окна `segments`

Формат статусов atomic-check в `checks`: только UPPERCASE (`OK` / `WARN` / `FAIL`).

Правило `segments`-статуса:
- `OK`: все выбранные `N` сегментов вернули HTTP `2xx`
- `WARN`: часть выбранных `N` сегментов недоступна
- `FAIL`: ни один выбранный сегмент не доступен, либо сегменты не извлечены из playlist

Правило `declared_bitrate`-статуса:
- `OK`: найден минимум один корректный declared bitrate (`BANDWIDTH` или `AVERAGE-BANDWIDTH`)
- `WARN`: теги `#EXT-X-STREAM-INF` отсутствуют или в них нет declared bitrate (неприменимо для части media-playlist)
- `FAIL`: теги присутствуют, но declared bitrate невалиден (нечисловой/<=0) или playlist недоступен

В `checks` сохраняются диагностические детали по `declared_bitrate` (например, `parsed_bitrate_bps`, `source`, `reason`) без секретов.

Правило `freeze`-статуса:
- `FAIL`: `max_freeze_sec >= FREEZE_FAIL_SEC` (по умолчанию `>= 5`)
- `WARN`: `max_freeze_sec >= FREEZE_WARN_SEC` и `< FREEZE_FAIL_SEC` (по умолчанию `>= 2` и `< 5`)
- `OK`: иначе

В `checks` сохраняются `freeze` (`OK/WARN/FAIL`) и `freeze_details` (`max_freeze_sec`, `reason`, `source`).

Правило `effective_bitrate`-статуса:
- формула: `calculated_bps = (sum(downloaded_segment_bytes) * 8) / sum(segment_duration_sec)` по скачанным сегментам окна
- `FAIL`: `ratio = calculated_bps / declared_bps < EFFECTIVE_BITRATE_FAIL_RATIO` (по умолчанию `< 0.4`)
- `WARN`: при отсутствии `FAIL`, если `ratio < EFFECTIVE_BITRATE_WARN_RATIO` (по умолчанию `< 0.7`) или declared bitrate недоступен (`declared_bitrate` неприменим)
- `OK`: ratio выше/равен warn-порогу

В `checks` сохраняются `effective_bitrate` (`OK/WARN/FAIL`) и `effective_bitrate_details` (`calculated_bps`, `declared_bps`, `ratio`, `reason`, `sample_count`).

Итоговая агрегация статуса: `FAIL > WARN > OK` по чекам `playlist`, `freshness`, `segments`, `freeze`, `declared_bitrate`, `effective_bitrate`.

Используемые thresholds:
- `PLAYLIST_TIMEOUT_MS` (по умолчанию `3000`)
- `SEGMENT_TIMEOUT_MS` (по умолчанию `5000`)
- `SEGMENTS_SAMPLE_COUNT` (по умолчанию `3`, допустимый диапазон `3..5`)
- `FRESHNESS_WARN_SEC` (по умолчанию `10`)
- `FRESHNESS_FAIL_SEC` (по умолчанию `30`)
- `FREEZE_WARN_SEC` (по умолчанию `2`)
- `FREEZE_FAIL_SEC` (по умолчанию `5`)
- `EFFECTIVE_BITRATE_WARN_RATIO` (по умолчанию `0.7`)
- `EFFECTIVE_BITRATE_FAIL_RATIO` (по умолчанию `0.4`)

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
