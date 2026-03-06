# ENV + Runtime аудит (docker compose)

Проверка на поднятом стеке: scheduler автопроверок, запись в check_results, ручная проверка из UI, Telegram alerts.

---

## Обязательные ENV (значения/форматы)

| Переменная | Где | Формат/значение | Обязательность |
|------------|-----|------------------|----------------|
| `SCHEDULER_API_BASE_URL` | scheduler | URL без trailing slash, в compose: `http://api:8080` | обязательна при SCHEDULER_ENABLED=true |
| `SCHEDULER_ACCESS_TOKEN` | scheduler | Bearer-токен (access_token пользователя с доступом к companies/streams, напр. super_admin) | обязательна при SCHEDULER_ENABLED=true |
| `SCHEDULER_INTERVAL_MIN` | scheduler | целое ≥1, по умолчанию 30 | опционально |
| `SCHEDULER_ENABLED` | scheduler | `true` / `false` (или 1/0, yes/no). По умолчанию true | опционально |
| `DATABASE_URL` | api, worker, init | `postgres://user:pass@host:port/db?sslmode=...` | обязательна |
| `REDIS_ADDR` | api, worker | `host:port`, напр. `redis:6379` | обязательна для очереди джобов |
| `TELEGRAM_BOT_TOKEN_DEFAULT` | worker | Токен бота Telegram (BotFather). Пустая строка = не шлём, если в настройках компании не указан ref | для отправки алертов |
| `TELEGRAM_BOT_TOKEN_<REF>` | worker | Если в telegram_delivery_settings задан bot_token_ref, то ENV `TELEGRAM_BOT_TOKEN_<normalized_ref>` (буквы/цифры/подчёркивания, верхний регистр) | опционально при использовании ref |
| `RETENTION_TTL_DAYS` | worker | целое ≥1, по умолчанию 30 | опционально |
| `RETENTION_CLEANUP_INTERVAL_MIN` | worker | целое ≥1, по умолчанию 60 | опционально |
| `RETENTION_CLEANUP_BATCH_SIZE` | worker | целое ≥1, по умолчанию 100 | опционально |
| `ALERT_SEND_RECOVERED` | worker | true/false, по умолчанию false | опционально |
| `ALERT_FAIL_STREAK` | worker | целое ≥1, по умолчанию 2 | опционально |
| `ALERT_COOLDOWN_MIN` | worker | целое ≥1, по умолчанию 10 | опционально |

---

## CHECKLIST → EXPECTED OUTPUT → COMMON FAILS → FIX COMMANDS

### 1. Все сервисы подняты

**Команда:**  
`docker compose ps`

**EXPECTED OUTPUT:**  
api, worker, scheduler, postgres, redis, frontend — состояние `running`. init — `exited (0)` допустимо. У api/worker/scheduler не должно быть `Restarting` или `Exit 1`.

**COMMON FAILS:**  
scheduler в состоянии `Exit 1` или постоянно перезапускается.

**FIX:**  
Проверить логи: `docker compose logs scheduler`. Если `SCHEDULER_ACCESS_TOKEN is required` или `SCHEDULER_API_BASE_URL is required` — задать в `.env` и перезапустить:  
`docker compose up -d scheduler`

---

### 2. Scheduler стартовал и не падает

**Команда:**  
`docker compose logs scheduler 2>&1 | tail -20`

**EXPECTED OUTPUT:**  
Строка вида:  
`scheduler started: api=http://api:8080 interval=30m0s`  
(или другой интервал). При `SCHEDULER_ENABLED=false`:  
`scheduler disabled by SCHEDULER_ENABLED; holding process`

**COMMON FAILS:**  
`log.Fatal ... SCHEDULER_ACCESS_TOKEN is required` — процесс завершился; контейнер в restart loop.

**FIX:**  
В `.env`: `SCHEDULER_ACCESS_TOKEN=<access_token>`. Взять токен: логин в API (POST /api/v1/auth/login), скопировать `access_token` из ответа. Затем:  
`docker compose up -d scheduler`

---

### 3. Scheduler реально кладёт jobs каждые N минут

**Команда (подождать 1–2 цикла по интервалу, затем логи):**  
`docker compose logs scheduler 2>&1 | grep "scheduler cycle"`

**EXPECTED OUTPUT:**  
Периодические строки не реже чем раз в SCHEDULER_INTERVAL_MIN минут, например:  
`scheduler cycle: enqueued=2 skipped=0 failed=0 companies=1`  
При наличии активных потоков `enqueued` или `skipped` > 0. Если потоков нет: `enqueued=0 skipped=0 failed=0 companies=1` — норма.

**COMMON FAILS:**  
- Только одна строка после старта, дальше тишина — проверить, что контейнер не падает: `docker compose ps scheduler`.  
- `failed` > 0 — смотреть выше в логах: `scheduler fetch companies` или `scheduler enqueue ... status=401/403` (токен неверный или нет прав).  
- `scheduler fetch companies: ... connection refused` — API не готов или неверный SCHEDULER_API_BASE_URL.

**FIX:**  
Токен: перевыпустить access_token (логин), обновить SCHEDULER_ACCESS_TOKEN, перезапустить scheduler. URL: в compose должен быть `SCHEDULER_API_BASE_URL=http://api:8080`.

---

### 4. В БД появляются новые check_jobs (за последние 60 минут)

**Команда (подставить свои POSTGRES_* или DATABASE_URL):**  
```bash
docker compose exec postgres psql -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-hls_monitoring}" -c "
SELECT id, company_id, stream_id, status, planned_at, created_at
FROM check_jobs
WHERE created_at > NOW() - INTERVAL '60 minutes'
ORDER BY created_at DESC
LIMIT 20;
"
```

**EXPECTED OUTPUT:**  
Таблица с строками: `status` = `queued` (ещё не взят), `running` (в работе), `done` или `failed` (обработаны). Для автопроверок `planned_at` и `created_at` примерно каждые N минут при наличии активных потоков.

**COMMON FAILS:**  
Пустой результат при работающем scheduler — нет компаний или активных потоков; или scheduler не доходит до API (см. п. 3).

**FIX:**  
Создать компанию и поток с `is_active=true` через API/UI. Проверить логи scheduler (п. 3).

---

### 5. Worker забирает джобы (логи)

**Команда:**  
`docker compose logs worker 2>&1 | grep -E "worker claimed job|worker finalized job" | tail -30`

**EXPECTED OUTPUT:**  
Пары строк на каждый job:  
`worker claimed job: id=... company_id=... stream_id=... planned_at=...`  
`worker finalized job as done: id=... company_id=...`  
или `worker finalized job as failed: id=... company_id=... reason=...`

**COMMON FAILS:**  
Нет строк "claimed" — Worker не видит джобов (очередь Redis, или все джобы уже обработаны). Нет "finalized job as done" при наличии "claimed" — падает на процессе проверки (логи выше: processErr/persistErr/alertErr).

**FIX:**  
Проверить REDIS_ADDR у api и worker одинаковый. Посмотреть полные логи: `docker compose logs worker`.

---

### 6. Worker реально пишет check_results

**Команда (логи):**  
`docker compose logs worker 2>&1 | grep "worker stored check_result" | tail -20`

**EXPECTED OUTPUT:**  
Строки вида:  
`worker stored check_result: job_id=... company_id=... stream_id=... status=ok|warn|fail checks=...`

**Команда (БД):**  
```bash
docker compose exec postgres psql -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-hls_monitoring}" -c "
SELECT id, company_id, job_id, stream_id, status, created_at
FROM check_results
WHERE created_at > NOW() - INTERVAL '60 minutes'
ORDER BY created_at DESC
LIMIT 20;
"
```

**EXPECTED OUTPUT:**  
Таблица с новыми строками; `created_at` после успешного "worker finalized job as done".

**COMMON FAILS:**  
Есть "finalized job as done", но нет "stored check_result" — не должно быть (persist идёт до finalize). Нет строк в check_results при наличии done-джобов — проверить логи на "persist retry" или ошибки БД.

**FIX:**  
Проверить DATABASE_URL worker, доступность Postgres: `docker compose exec worker wget -q -O - http://localhost:9091/health` (worker health), логи БД/репо не показываются отдельно — смотреть `docker compose logs worker`.

---

### 7. Ручная проверка из UI

**Действие:**  
Залогиниться в ЛК → Streams → нажать «Run check» у любого потока (роль не viewer).

**EXPECTED OUTPUT:**  
Сообщение в UI вида «Check job #N queued for stream #M». В БД в течение секунд появляется новая строка в check_jobs с `status=queued`, затем worker забирает и появляется check_result.

**COMMON FAILS:**  
Кнопка не реагирует / ошибка сети — проверить NEXT_PUBLIC_API_BASE_URL и что API доступен из браузера. 401/403 — сессия истекла или нет прав (company_admin/super_admin).

**FIX:**  
Перелогиниться. Проверить: `curl -s -o /dev/null -w "%{http_code}" -X POST -H "Authorization: Bearer <token>" "http://localhost:8080/api/v1/companies/1/streams/1/check"` — ожидается 202.

---

### 8. API readiness и health

**Команда:**  
`curl -s http://localhost:8080/api/v1/health`  
`curl -s http://localhost:8080/api/v1/ready`

**EXPECTED OUTPUT:**  
health: `{"status":"ok","service":"api",...}` и HTTP 200. ready: `{"ready":true}` 200 при доступной БД, иначе 503 `{"ready":false}`.

**COMMON FAILS:**  
Connection refused — API не слушает порт или не проброшен. ready 503 — БД недоступна для API.

**FIX:**  
`docker compose ps api`; при необходимости `docker compose up -d postgres api` и подождать healthy.

---

### 9. Telegram: отправка или причина скипа (логи)

**Команда:**  
`docker compose logs worker 2>&1 | grep "worker telegram delivery" | tail -30`

**EXPECTED OUTPUT (отправлено):**  
`worker telegram delivery: company_id=... stream_id=... event_type=... should_send=true delivery_result=sent reason=ok`

**EXPECTED OUTPUT (скип — нормально):**  
- `delivery_result=skipped reason=decision_false` — по логике алертов не шлём (cooldown, streak, нет перехода).  
- `delivery_result=skipped reason=settings_not_found` — для company нет строки в telegram_delivery_settings.  
- `delivery_result=skipped reason=settings_disabled` — is_enabled=false.  
- `delivery_result=skipped reason=chat_id_missing` — chat_id пустой.  
- `delivery_result=skipped reason=recovered_disabled_for_company` — событие recovered, а send_recovered=false.  
- `delivery_result=failed reason=token_resolve_error` или `reason=send_error` — нет/неверный токен бота или ошибка Telegram API.

**COMMON FAILS:**  
Всегда скип из‑за settings_not_found — в ЛК не настроен Telegram для компании. Всегда token_resolve_error — не задан TELEGRAM_BOT_TOKEN_DEFAULT (или TELEGRAM_BOT_TOKEN_<REF> при использовании ref).

**FIX:**  
Настройки: ЛК → Settings → Telegram: включить, указать chat_id, при необходимости bot_token_ref и задать соответствующий ENV в worker. В `.env` для worker: `TELEGRAM_BOT_TOKEN_DEFAULT=<bot_token>`. Перезапуск: `docker compose up -d worker`.

---

### 10. Retention cleanup (TTL 30 дней)

**Команда:**  
`docker compose logs worker 2>&1 | grep "worker retention cleanup" | tail -15`

**EXPECTED OUTPUT:**  
Периодически:  
`worker retention cleanup heartbeat: <timestamp>`  
и при наличии старых данных:  
`worker retention cleanup: company_id=... affected_rows=... deleted_files=... errors_count=...`

**COMMON FAILS:**  
Нет heartbeat — worker не стартовал или падает до тикера. Ошибки в логах — смотреть "worker retention cleanup failed".

**FIX:**  
Проверить `docker compose logs worker` с начала; убедиться, что RETENTION_TTL_DAYS и RETENTION_CLEANUP_INTERVAL_MIN заданы по желанию (по умолчанию 30 и 60).

---

### 11. Scheduler: один цикл «на сейчас» (быстрая проверка)

**Команда (интервал 1 мин для теста):**  
В `.env`: `SCHEDULER_INTERVAL_MIN=1`. Перезапуск: `docker compose up -d scheduler`. Подождать 2–3 минуты.

`docker compose logs scheduler 2>&1 | grep "scheduler cycle"`

**EXPECTED OUTPUT:**  
Несколько строк с интервалом ~1 мин, например:  
`scheduler cycle: enqueued=1 skipped=0 failed=0 companies=1`

**COMMON FAILS:**  
Как в п. 3.

**FIX:**  
Вернуть `SCHEDULER_INTERVAL_MIN=30` после проверки.

---

### 12. Сводка по check_jobs и check_results за 60 минут

**Команда:**  
```bash
docker compose exec postgres psql -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-hls_monitoring}" -c "
SELECT
  (SELECT COUNT(*) FROM check_jobs WHERE created_at > NOW() - INTERVAL '60 minutes') AS jobs_last_60min,
  (SELECT COUNT(*) FROM check_results WHERE created_at > NOW() - INTERVAL '60 minutes') AS results_last_60min;
"
```

**EXPECTED OUTPUT:**  
При работающих scheduler и worker: `jobs_last_60min` ≥ 1 (при наличии активных потоков), `results_last_60min` близко к количеству обработанных джобов (done/failed).

**COMMON FAILS:**  
jobs_last_60min=0 — scheduler не создаёт или нет потоков. results_last_60min=0 при jobs > 0 — worker не обрабатывает или не пишет (см. п. 5–6).

**FIX:**  
По пунктам 2–3 (scheduler) и 5–6 (worker).

---

## Минимальный прогон на сервере (копируй и запускай)

```bash
# 1. Сервисы
docker compose ps

# 2. Scheduler старт
docker compose logs scheduler 2>&1 | tail -5

# 3. Циклы scheduler (подождать интервал)
docker compose logs scheduler 2>&1 | grep "scheduler cycle"

# 4. Джобы за 60 мин
docker compose exec postgres psql -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-hls_monitoring}" -c "SELECT id, company_id, stream_id, status, created_at FROM check_jobs WHERE created_at > NOW() - INTERVAL '60 minutes' ORDER BY created_at DESC LIMIT 10;"

# 5. Результаты за 60 мин
docker compose exec postgres psql -U "${POSTGRES_USER:-app}" -d "${POSTGRES_DB:-hls_monitoring}" -c "SELECT id, company_id, job_id, stream_id, status, created_at FROM check_results WHERE created_at > NOW() - INTERVAL '60 minutes' ORDER BY created_at DESC LIMIT 10;"

# 6. Worker пишет check_result
docker compose logs worker 2>&1 | grep "worker stored check_result" | tail -5

# 7. Telegram delivery
docker compose logs worker 2>&1 | grep "worker telegram delivery" | tail -10
```

После прогона: если п. 2 — "scheduler started", п. 3 — периодические "scheduler cycle", п. 4 — есть строки при активных потоках, п. 5–6 — есть строки и "stored check_result", то автопроверки и запись истории работают. П. 7 интерпретировать по reason (sent vs skipped/ failed и причина).
