# HLS Monitoring Platform Bootstrap

Минимальный инфраструктурный каркас репозитория и Docker Compose окружения.

## Состав сервисов

- `api` (Go)
- `worker` (Go)
- `postgres` (PostgreSQL)
- `redis` (Redis)
- `frontend` (Next.js + TypeScript)

## Frontend (сборка и запуск)

Из каталога `web/`: `npm install` — установка зависимостей; `npm run dev` — режим разработки (dev-сервер с hot reload); `npm run build` — продакшен-сборка. После сборки запуск: `npm run start`.

## Реализованные возможности

- **HLS-мониторинг:** Worker выполняет проверки (playlist, segments, freshness, freeze, blackframe, declared_bitrate, effective_bitrate), статусы OK/WARN/FAIL, сохранение результатов и скриншотов.
- **Потоки и проекты:** tenant-scoped CRUD компаний, проектов, потоков; постановка check-jobs в очередь, история проверок и результатов.
- **Админка:** controlled registration (pending → approve/reject только super_admin), список пользователей и заявок, смена ролей и статусов, audit log.
- **Telegram:** алерты при переходах OK→WARN, WARN→FAIL и recovered (cooldown, streak); настройки доставки по компании (chat_id, send_recovered); DevLog в Telegram после каждого коммита (post-commit hook); «настроение» в сообщении — из `.devlog_mood.txt` в корне (опционально).
- **Аналитика:** отображение состояний потоков, тренды, фильтрация по FAIL/WARN в UI.
- **Плеер HLS:** просмотр потока в браузере, тёмная тема.
- **AI по инцидентам:** при WARN/FAIL Worker после сохранения результата вызывает AI (cause/summary), результат сохраняется в БД; API отдаёт его по GET для job (только чтение).
- **Тесты:** юнит-тесты API handlers (streams, check-jobs, check-results, ai-incident — 200/404/401/403) и Worker (job flow, persist, alert state); `go test ./...`.

## Документация

- [docs/api_contract.md](docs/api_contract.md) — контракт REST API (эндпоинты, форматы, коды ошибок, tenant scope).
- [docs/schema.md](docs/schema.md) — схема БД и порядок применения/отката миграций.
- [docs/telegram_alerts_contract.md](docs/telegram_alerts_contract.md) — контракт Telegram Alerts (Worker): переходы статуса, формат сообщения, антиспам (cooldown/streak).
- [docs/retention_cleanup.md](docs/retention_cleanup.md) — retention cleanup в Worker: TTL, батчи, tenant scope, конфиг ENV.
- [docs/ai_incident_contract.md](docs/ai_incident_contract.md) — целевой контракт интеграции AI по инцидентам (B6): триггер WARN/FAIL, вход/выход, on-demand.
- [docs/decisions.md](docs/decisions.md) — архитектурные решения (ADR).
- [docs/agents_and_responsibilities.md](docs/agents_and_responsibilities.md) — процесс, роли агентов, JOB→RESULT→ROUTING, источник истины по правилам.
- [docs/screenshot_automation.md](docs/screenshot_automation.md) — автоматизация скриншотов для UI-модулей (Docker, скрипты).
- [docs/agent_devlog.md](docs/agent_devlog.md) — журнал работ агентов по модулям, формат записей и ограничения.

Переменные DevLog (`DEV_LOG_*`), retention (`RETENTION_*`) и Telegram Alerts (`TELEGRAM_*`, `ALERT_*`) описаны в [.env.example](.env.example) и в подразделах README ниже.

## Тесты

Запуск всех Go-тестов из корня репозитория:

```bash
go test ./...
```

Тесты, требующие БД, используют `DATABASE_URL` из окружения (например из `.env` или `env_dev`); при отсутствии подключения такие тесты можно пропускать.

## Переменные окружения

1. Создать локальный env-файл:

```bash
cp .env.example .env
```

2. При необходимости изменить значения в `.env`:

- `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_PORT`
- `API_PORT`, `REDIS_ADDR`, `AUTH_ACCESS_TTL_MIN`, `AUTH_REFRESH_TTL_DAYS`, `AUTH_TELEGRAM_MAX_AGE_SEC`, `SUPER_ADMIN_TELEGRAM_CHAT_ID`, `WORKER_HEARTBEAT_SEC`
- `WORKER_JOB_TIMEOUT_SEC`, `WORKER_DB_RETRY_MAX`, `WORKER_DB_RETRY_BACKOFF_MS`
- `PLAYLIST_TIMEOUT_MS`, `SEGMENT_TIMEOUT_MS`, `SEGMENTS_SAMPLE_COUNT`, `FRESHNESS_WARN_SEC`, `FRESHNESS_FAIL_SEC`, `FREEZE_WARN_SEC`, `FREEZE_FAIL_SEC`, `BLACKFRAME_WARN_RATIO`, `BLACKFRAME_FAIL_RATIO`, `EFFECTIVE_BITRATE_WARN_RATIO`, `EFFECTIVE_BITRATE_FAIL_RATIO`, `ALERT_FAIL_STREAK`, `ALERT_COOLDOWN_MIN`, `ALERT_SEND_RECOVERED`, `TELEGRAM_HTTP_TIMEOUT_MS`, `TELEGRAM_SEND_RETRY_MAX`, `TELEGRAM_SEND_RETRY_BACKOFF_MS`, `TELEGRAM_BOT_TOKEN_DEFAULT`, `RETENTION_TTL_DAYS`, `RETENTION_CLEANUP_INTERVAL_MIN`, `RETENTION_CLEANUP_BATCH_SIZE`
- `FRONTEND_PORT`, `NEXT_PUBLIC_API_BASE_URL`

Файл `.env` не добавляется в git (трекается только `.env.example`).

## Запуск в Docker (без ручных шагов)

Миграции и тестовый пользователь для скриншотов применяются автоматически при старте (сервис `init`).

```bash
docker compose up --build -d
```

После старта: API на 8080, frontend на 3000, БД с миграциями и сидером (логин `test_screenshot_admin` / `TestScreenshot1`).

### Быстрый старт

Выполните `docker compose up --build -d`, откройте http://localhost:3000 и войдите под пользователем `test_screenshot_admin` / `TestScreenshot1`.

Проверка API:

```bash
curl http://localhost:8080/api/v1/health
```

### Скриншот страницы Settings (одна команда)

Поднять стек и один раз выполнить контейнер скриншота (Playwright):

```bash
docker compose up --build -d
docker compose --profile screenshot run --rm screenshot
```

Скриншот сохранится в `screenshots/telegram-delivery-settings/<timestamp>.png` (репозиторий смонтирован в контейнер). При необходимости скрипт сделает `git add` и коммит.

### Telegram DevLog после коммита

В `.env` задать: `DEV_LOG_TELEGRAM_ENABLED=true`, `DEV_LOG_TELEGRAM_TOKEN`, `DEV_LOG_TELEGRAM_CHAT_ID`. Один раз включить хуки:

```bash
git config core.hooksPath .githooks
```

После каждого коммита будет отправляться сообщение в Telegram (скрипт `scripts/devlog_notify.ps1`).

## Деплой (запуск в production)

Для оценки или продакшена поднимите стек через Docker Compose из корня репозитория:

```bash
docker compose up --build -d
```

**Обязательные переменные окружения** (в `.env` или в среде контейнеров): `DATABASE_URL`, `API_PORT` (по умолчанию 8080), для frontend — `NEXT_PUBLIC_API_BASE_URL` (URL API с точки зрения браузера). Остальные переменные — см. [.env.example](.env.example) и разделы README выше.

**Порты:** API — 8080, frontend — 3000. При необходимости пробросьте их в `docker-compose.yml` на хосте.

**Volumes:** для персистентности БД используйте volume для PostgreSQL (в стандартном `docker-compose.yml` уже задан). Локальные скриншоты и миграции — по текущей конфигурации Compose.

**Проверка после старта:** `GET /api/v1/health` возвращает 200 и JSON `{"status":"ok","service":"api",...}`. Этого достаточно для liveness и readiness в оркестраторе (Kubernetes, Docker Swarm); отдельный readiness endpoint с проверкой БД/Redis не обязателен для базового деплоя.

## Authentication + controlled registration smoke-check

All API endpoints except health and public auth endpoints are protected by bearer auth.

1. Apply migrations (если не используете Docker; при `docker compose up` миграции выполняет сервис `init`):

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0001_baseline_schema.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0002_telegram_delivery_settings.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0003_preserve_company_audit_history.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0004_auth_and_registration.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0005_indexes_admin_and_lists.up.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/0006_ai_incident_results.up.sql
```

2. Create bootstrap super-admin user (local smoke):

```bash
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -c "INSERT INTO users (company_id, email, login, password_hash, role, status) VALUES (NULL, 'root@example.com', 'root', '\$2a\$10\$N5qmS5BHUDsoD3/9rXHpPequqyXHEWq2w.gAldS7zKKy10zG/T4qC', 'super_admin', 'active');"
```

Password for this hash: `Admin12345`.

3. Login and export access token:

```bash
ACCESS_TOKEN=$(curl -sS -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"login_or_email":"root","password":"Admin12345"}' | jq -r '.access_token')
```

4. Public controlled registration request:

```bash
curl -sS -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"company_id":1,"email":"viewer@example.com","login":"viewer01","password":"Viewer12345","requested_role":"viewer"}'
```

5. Super-admin moderation:

```bash
curl -sS http://localhost:8080/api/v1/admin/registration-requests \
  -H "Authorization: Bearer $ACCESS_TOKEN"

curl -sS -X POST http://localhost:8080/api/v1/admin/registration-requests/1/approve \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"company_id":1,"role":"viewer"}'
```

## Companies CRUD smoke-check

All requests below require header `Authorization: Bearer $ACCESS_TOKEN`.

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

Use `-H "Authorization: Bearer $ACCESS_TOKEN"` for all calls.

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

Use `-H "Authorization: Bearer $ACCESS_TOKEN"` for all calls.

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

Use `-H "Authorization: Bearer $ACCESS_TOKEN"` for all calls.

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
- `blackframe` check по кадрам через `ffmpeg blackframe` (яркость/доля тёмных кадров)
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

Правило `blackframe`-статуса:
- `FAIL`: `dark_frame_ratio >= BLACKFRAME_FAIL_RATIO` (по умолчанию `>= 0.98`)
- `WARN`: `dark_frame_ratio >= BLACKFRAME_WARN_RATIO` и `< BLACKFRAME_FAIL_RATIO` (по умолчанию `>= 0.9` и `< 0.98`)
- `OK`: иначе

При невозможности анализа worker возвращает детерминированный `WARN` с явным `reason` в `blackframe_details`.
Ожидаемый набор fallback-reason:
- `playlist_unavailable`
- `segments_not_available`
- `no_downloaded_segments`
- `blackframe_analysis_failed`
- `blackframe_analyzer_not_available`

В `checks` сохраняются `blackframe` (`OK/WARN/FAIL`) и `blackframe_details` (`dark_frame_ratio`, `analyzed_frames`, `reason`, `source`).

Правило `effective_bitrate`-статуса:
- формула: `calculated_bps = (sum(downloaded_segment_bytes) * 8) / sum(segment_duration_sec)` по скачанным сегментам окна
- `FAIL`: `ratio = calculated_bps / declared_bps < EFFECTIVE_BITRATE_FAIL_RATIO` (по умолчанию `< 0.4`)
- `WARN`: при отсутствии `FAIL`, если `ratio < EFFECTIVE_BITRATE_WARN_RATIO` (по умолчанию `< 0.7`) или declared bitrate недоступен (`declared_bitrate` неприменим)
- `OK`: ratio выше/равен warn-порогу

В `checks` сохраняются `effective_bitrate` (`OK/WARN/FAIL`) и `effective_bitrate_details` (`calculated_bps`, `declared_bps`, `ratio`, `reason`, `sample_count`).

Итоговая агрегация статуса: `FAIL > WARN > OK` по чекам `playlist`, `freshness`, `segments`, `freeze`, `blackframe`, `declared_bitrate`, `effective_bitrate`.

Alert anti-spam decision engine (`alert_state`):
- fail alert отправляется только после `ALERT_FAIL_STREAK` подряд `FAIL` (по умолчанию `2`)
- после решения `should_send=true` включается cooldown на `ALERT_COOLDOWN_MIN` минут (по умолчанию `10`)
- recovered decision активен только для перехода `FAIL -> OK` и только если `ALERT_SEND_RECOVERED=true` (по умолчанию `false`), также с учетом cooldown
- Telegram transport вызывается только при `should_send=true` и читает tenant-scoped настройки из `telegram_delivery_settings` по `company_id`
- recovered отправка дополнительно требует `send_recovered=true` в `telegram_delivery_settings`
- resolver токена:
  - если `bot_token_ref` пустой -> используется `TELEGRAM_BOT_TOKEN_DEFAULT`
  - если `bot_token_ref` задан -> используется env `TELEGRAM_BOT_TOKEN_<REF_NORMALIZED>` (`REF_NORMALIZED`: uppercase + неалфанумерные символы заменены `_`)
- токены и секреты не логируются
- retention cleanup выполняется только в Worker: каждые `RETENTION_CLEANUP_INTERVAL_MIN` минут удаляются `check_results` старше `RETENTION_TTL_DAYS` и связанные screenshot-файлы батчами по `RETENTION_CLEANUP_BATCH_SIZE` в tenant-scope (`company_id`)
- ошибки удаления файлов логируются и учитываются в `errors_count`, но не останавливают cleanup цикла

Используемые thresholds:
- `PLAYLIST_TIMEOUT_MS` (по умолчанию `3000`)
- `SEGMENT_TIMEOUT_MS` (по умолчанию `5000`)
- `SEGMENTS_SAMPLE_COUNT` (по умолчанию `3`, допустимый диапазон `3..5`)
- `FRESHNESS_WARN_SEC` (по умолчанию `10`)
- `FRESHNESS_FAIL_SEC` (по умолчанию `30`)
- `FREEZE_WARN_SEC` (по умолчанию `2`)
- `FREEZE_FAIL_SEC` (по умолчанию `5`)
- `BLACKFRAME_WARN_RATIO` (по умолчанию `0.9`)
- `BLACKFRAME_FAIL_RATIO` (по умолчанию `0.98`)
- `EFFECTIVE_BITRATE_WARN_RATIO` (по умолчанию `0.7`)
- `EFFECTIVE_BITRATE_FAIL_RATIO` (по умолчанию `0.4`)
- `ALERT_FAIL_STREAK` (по умолчанию `2`)
- `ALERT_COOLDOWN_MIN` (по умолчанию `10`)
- `ALERT_SEND_RECOVERED` (по умолчанию `false`)
- `TELEGRAM_HTTP_TIMEOUT_MS` (по умолчанию `5000`)
- `TELEGRAM_SEND_RETRY_MAX` (по умолчанию `2`)
- `TELEGRAM_SEND_RETRY_BACKOFF_MS` (по умолчанию `500`)
- `TELEGRAM_BOT_TOKEN_DEFAULT` (по умолчанию пусто)
- `RETENTION_TTL_DAYS` (по умолчанию `30`)
- `RETENTION_CLEANUP_INTERVAL_MIN` (по умолчанию `60`)
- `RETENTION_CLEANUP_BATCH_SIZE` (по умолчанию `100`)

## Check results API smoke-check

Use `-H "Authorization: Bearer $ACCESS_TOKEN"` for all calls.

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

## Dev log Telegram completion notifier

`cmd/devnotify` is an isolated completion notifier and does not reuse the worker stream-alert path.

Required env vars:
- `DEV_LOG_TELEGRAM_ENABLED` (`false` by default)
- `DEV_LOG_TELEGRAM_TOKEN`
- `DEV_LOG_TELEGRAM_CHAT_ID`

Usage example:

```bash
go run ./cmd/devnotify \
  -module "phase-c-step" \
  -agent "BackendAgent" \
  -commit "<commit_hash>" \
  -summary "Закрыт шаг рефакторинга" \
  -summary "Тесты прошли" \
  -thought "Было жарко, но дожали" \
  -thought "Дальше можно идти спокойно" \
  -mood "Огонь"
```

Message format:

```text
[MODULE ЗАВЕРШЕНО]
Агент: <AgentName>
Коммит: <hash>
Статус: УСПЕХ
Сводка:
- ...
- ...
Настроение: <value>
Мысли:
- ...
- ...
```

Notes:
- `Summary` / `Mood` / `Thoughts` are Russian by default.
- `-thought` is optional and can be repeated up to 2 lines.
- Payload is validated by safety guardrails (no personal insults, hate/discrimination, secrets/tokens, PII, architecture decisions).
- Send errors are logged without secrets and do not affect API/Worker runtime flows.
