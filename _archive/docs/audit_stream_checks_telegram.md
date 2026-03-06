# Аудит: проверки потоков / история / Telegram / ручные проверки

**Дата:** 2026-02-22  
**Репозиторий:** https://github.com/terrynullson/mntrng  
**Фокус:** 4 блока — периодические проверки, хранение/ретеншн, ручная проверка из ЛК, Telegram alerts.

---

## A) Периодические проверки («каждые ~30 минут»)

### DONE

- **Сервис scheduler** — отдельный бинарник `cmd/scheduler/main.go`, без `internal/`, только stdlib.
  - Раз в `SCHEDULER_INTERVAL_MIN` минут (по умолчанию 30): GET `/api/v1/companies` → для каждой компании GET `/api/v1/companies/{id}/streams?is_active=true` → для каждого активного потока POST `/api/v1/companies/{id}/streams/{stream_id}/check-jobs` с телом `{"planned_at": "<now RFC3339>"}`.
  - Аутентификация: заголовок `Authorization: Bearer <SCHEDULER_ACCESS_TOKEN>`.
  - Обработка ответов: 202 — успех, 409 — пропуск (идемпотентность), иначе — лог + счётчик failed.
  - Graceful shutdown по SIGINT/SIGTERM; при `SCHEDULER_ENABLED=false` процесс не создаёт джобы, только ждёт сигнала.
- **Инфраструктура:** `Dockerfile.scheduler` (Alpine, без БД/Redis/ffmpeg), в `docker-compose.yml` сервис `scheduler` с `depends_on: api (condition: service_healthy)`, переменные `SCHEDULER_API_BASE_URL`, `SCHEDULER_ACCESS_TOKEN`, `SCHEDULER_INTERVAL_MIN`, `SCHEDULER_ENABLED`.
- **Создание check_jobs:** только через API. Обработчик `handleEnqueueCheckJob` в `internal/http/api/handlers_check_job.go` вызывает `checkJobService.EnqueueCheckJob`; запись в `check_jobs` создаётся в API (сервис/репо), Worker только забирает джобы из очереди (Redis/БД по текущей реализации).

**Ссылки:**  
`cmd/scheduler/main.go` (runOnce, fetchCompanies, fetchStreams, enqueueCheckJob),  
`internal/http/api/handlers_check_job.go` (handleEnqueueCheckJob),  
`internal/http/api/router.go` (POST …/check-jobs),  
`docker-compose.yml` (сервис scheduler).

### NOT DONE / MISSING

- **Ничего критичного.** Для работы «каждые 30 минут» нужно:
  1. Задать в `.env` **`SCHEDULER_ACCESS_TOKEN`** (Bearer-токен пользователя с доступом к списку компаний и потоков, например super_admin после логина).
  2. Не выставлять `SCHEDULER_ENABLED=false` (по умолчанию `true`).
  3. Запускать стек с сервисом `scheduler` (`docker compose up -d`).

Если `SCHEDULER_ACCESS_TOKEN` пустой, процесс scheduler при старте завершается с `log.Fatal("SCHEDULER_ACCESS_TOKEN is required")` — контейнер будет падать. Это единственная «не включённая по умолчанию» вещь.

### HOW TO VERIFY

1. В `.env`: `SCHEDULER_ACCESS_TOKEN=<access_token super_admin>`, `SCHEDULER_ENABLED=true`, `SCHEDULER_INTERVAL_MIN=30` (или 1 для теста).
2. `docker compose up -d`; дождаться healthy API.
3. В логах scheduler: `scheduler started: api=... interval=30m0s`; через интервал — `scheduler cycle: enqueued=... skipped=... failed=... companies=...`.
4. В БД: появление новых строк в `check_jobs` с `status='queued'` и `planned_at` примерно каждые 30 минут для активных потоков.

---

## B) Хранение истории результатов и ретеншн (TTL 30 days)

### DONE

- **Запись истории (check_results):** Worker после выполнения проверки вызывает `persistCheckResultWithRetry` → `persistCheckResult` → `checkResultRepo.PersistCheckResult`. В репозитории выполняется **INSERT** в `check_results` (company_id, job_id, stream_id, status, checks).
  - **Ссылки:**  
  `internal/service/worker/job_flow.go` (стр. 39–45: persistCheckResultWithRetry после processJob),  
  `internal/service/worker/worker_alert_state.go` (persistCheckResult, вызов checkResultRepo.PersistCheckResult),  
  `internal/repo/postgres/worker_check_result_repo.go` (INSERT INTO check_results ... ON CONFLICT (job_id) DO NOTHING).

- **Ретеншн (TTL 30 дней):** В Worker запускается периодическая очистка по расписанию.
  - Конфиг: `RETENTION_TTL_DAYS` (по умолчанию 30), `RETENTION_CLEANUP_INTERVAL_MIN` (по умолчанию 60), `RETENTION_CLEANUP_BATCH_SIZE` (по умолчанию 100). Задаётся в `cmd/worker/main.go`.
  - Логика: `RunRetentionCleanup` в `internal/service/worker/worker_retention_finalize.go` — по каждой компании выбор кандидатов с `created_at < cutoff` (cutoff = now - TTL), удаление файлов скриншотов по `screenshot_path`, затем `DeleteStaleCheckResult` в репозитории.
  - Запуск: из `internal/service/worker/app.go` по тикеру `cleanupTicker` (интервал `retentionCleanupInterval`); при старте один раз вызывается `runRetentionCleanupWithRetry`, затем периодически.
  - **Ссылки:**  
  `docs/retention_cleanup.md`,  
  `internal/service/worker/worker_retention_finalize.go`,  
  `internal/repo/postgres/worker_retention_repo.go` (ListRetentionCandidates, DeleteStaleCheckResult),  
  `cmd/worker/main.go` (retentionTTLDays, retentionCleanupInterval),  
  `internal/service/worker/app.go` (retentionCleanup ticker).

### NOT DONE / MISSING

- Нет. История пишется в цикле Worker (INSERT в check_results); ретеншн по TTL 30 дней реализован и включён в Worker.

### HOW TO VERIFY

1. Убедиться, что Worker обрабатывает джобы: в логах worker — `worker stored check_result: job_id=...`, `worker finalized job as done`.
2. В БД: `SELECT id, company_id, stream_id, status, created_at FROM check_results ORDER BY created_at DESC LIMIT 10;` — появление новых строк после проверок.
3. Ретеншн: в логах worker — `worker retention cleanup heartbeat`, `worker retention cleanup: company_id=... affected_rows=...`. Для быстрой проверки можно временно выставить `RETENTION_TTL_DAYS=1` и `RETENTION_CLEANUP_INTERVAL_MIN=1` и убедиться, что старые строки удаляются.

---

## C) Ручная проверка из личного кабинета (кнопка / эндпоинт)

### DONE

- **API:**  
  - POST `/api/v1/companies/{company_id}/streams/{stream_id}/check-jobs` — создание check job с телом `{"planned_at": "..."}` (используется scheduler’ом).  
  - POST `/api/v1/companies/{company_id}/streams/{stream_id}/check` — «ручной» триггер: без тела, сервер подставляет `planned_at=now`, ответ 202 и созданная job.  
  - Оба требуют авторизации и прав доступа к tenant (company_id).
  - **Ссылки:**  
  `internal/http/api/handlers_check_job.go` (handleEnqueueCheckJob, handleTriggerStreamCheck),  
  `internal/http/api/router.go` (streamParts[1] == "check" → HandleTriggerStreamCheck, streamParts[1] == checkJobsCollectionPath → HandleEnqueueCheckJob).

- **Frontend:** На странице потоков (Streams) для каждого потока есть кнопка «Run check» (для не-viewer). Она вызывает POST `/companies/{scopeCompanyId}/streams/${stream.id}/check` (без тела) и при успехе показывает сообщение о постановке job в очередь.
  - **Ссылки:**  
  `web/app/streams/page.tsx` (handleRunCheck, apiRequest POST `.../check`, кнопка с aria-label «Run check for stream …»).

### NOT DONE / MISSING

- Нет. Ручная проверка из ЛК реализована: эндпоинт POST …/check и кнопка «Run check» на странице Streams.

### HOW TO VERIFY

1. Залогиниться как company_admin или super_admin, открыть раздел Streams.
2. Нажать «Run check» у любого потока. Должно появиться сообщение вида «Check job #… queued for stream #…».
3. В логах API — обработка POST; в БД — новая запись в `check_jobs` со status `queued`; после обработки Worker’ом — запись в `check_results`.

---

## D) Telegram alerts (настройки + реальная отправка из Worker)

### DONE

- **Настройки (API и БД):**  
  - Таблица `telegram_delivery_settings` (миграция `migrations/0002_telegram_delivery_settings.up.sql`): company_id, is_enabled, chat_id, send_recovered, bot_token_ref.  
  - API: GET/PATCH `/api/v1/companies/{company_id}/telegram-delivery-settings` — чтение/обновление настроек (company_admin, super_admin; viewer — доступ запрещён).  
  - **Ссылки:**  
  `internal/http/api/handlers_telegram_settings.go`,  
  `internal/repo/postgres/api_telegram_settings_repo.go`,  
  `internal/http/api/router.go` (telegramDeliverySettingsPath),  
  `migrations/0002_telegram_delivery_settings.up.sql`.

- **Реальная отправка из Worker:**  
  - В пайплайне после сохранения результата и применения alert state вызывается `processTelegramDelivery(ctx, job, evaluation, alertDecision)` (`internal/service/worker/job_flow.go`, стр. 60).  
  - В `internal/service/worker/worker_telegram.go`: загрузка настроек по company_id из БД (`loadTelegramDeliverySettings` → `telegramSettingsRepo.LoadTelegramDeliverySettings`), проверка is_enabled, chat_id, send_recovered; разрешение токена бота (default или по ref из настроек); сборка текста сообщения и вызов `sendTelegramMessageWithRetry` → `transport.NewClient().SendMessage` (пакет `internal/telegram`).  
  - Worker читает настройки из той же таблицы `telegram_delivery_settings` через `internal/repo/postgres/worker_telegram_repo.go` (LoadTelegramDeliverySettings).  
  - **Ссылки:**  
  `internal/service/worker/job_flow.go` (processTelegramDelivery после applyAlertStateWithRetry и logAlertDecision),  
  `internal/service/worker/worker_telegram.go` (processTelegramDelivery, loadTelegramDeliverySettings, sendTelegramMessageWithRetry, buildTelegramMessage),  
  `internal/repo/postgres/worker_telegram_repo.go`,  
  `internal/telegram` (HTTP-клиент отправки в Telegram Bot API).

### NOT DONE / MISSING

- Нет. Telegram alerts реализованы: настройки в ЛК (Settings → Telegram) и в БД, отправка выполняется в Worker при переходе статуса (warn/fail/recovered) с учётом cooldown, streak и send_recovered. Если настройки не найдены, is_enabled=false или chat_id пустой — отправка пропускается с логом (skipped).

### HOW TO VERIFY

1. В ЛК: Settings → включить Telegram, указать chat_id и при необходимости bot token ref; сохранить.
2. В `.env` Worker: задать `TELEGRAM_BOT_TOKEN_DEFAULT` (или соответствующий `TELEGRAM_BOT_TOKEN_<REF>`) для используемого бота.
3. Довести поток до перехода OK→WARN или WARN→FAIL (или FAIL→OK при включённом send_recovered).
4. В логах worker: строки `worker telegram delivery: ... delivery_result=sent ... reason=ok` при успешной отправке или `delivery_result=skipped`/`failed` с причиной.

---

## Итоговая сводка

| Блок | Статус | Кто создаёт check_jobs | Где хранится история | Примечание |
|------|--------|-------------------------|------------------------|------------|
| A) Периодические проверки | Реализовано | Scheduler (HTTP → API) | check_results (Worker INSERT) | Нужен SCHEDULER_ACCESS_TOKEN в .env |
| B) История и ретеншн | Реализовано | — | check_results, TTL 30 дней в Worker | INSERT в worker_check_result_repo; cleanup в worker_retention_finalize |
| C) Ручная проверка из ЛК | Реализовано | Frontend → API POST …/check | check_results (Worker INSERT) | Кнопка «Run check» на странице Streams |
| D) Telegram alerts | Реализовано | — | Настройки в telegram_delivery_settings | Worker вызывает processTelegramDelivery после persist и alert state |

---

## Вердикт

**Работает ли сейчас автопроверка каждые 30 минут?**  
**Да**, при условии что в окружении задан **`SCHEDULER_ACCESS_TOKEN`** (Bearer-токен пользователя с доступом к компаниям и потокам) и сервис scheduler запущен (по умолчанию он в составе `docker compose up -d`). Без токена контейнер scheduler падает при старте.

**Что минимально нужно добавить/включить, чтобы стало «да» (если сейчас «нет»):**

1. Задать в `.env` переменную **`SCHEDULER_ACCESS_TOKEN`** — выдать токен один раз (логин super_admin → скопировать access_token из ответа логина или из cookie) и прописать в `.env`; перезапустить scheduler (или весь стек).
2. Убедиться, что **scheduler** не отключён: не ставить `SCHEDULER_ENABLED=false`.
3. Убедиться, что в системе есть **хотя бы одна компания и хотя бы один активный поток** (`is_active=true`), иначе цикл scheduler будет с enqueued=0 (логи при этом в порядке).

Дополнительно для алертов в Telegram: в ЛК включить и настроить Telegram delivery для компании и задать токен бота в `.env` Worker (`TELEGRAM_BOT_TOKEN_DEFAULT` или ref из настроек).
