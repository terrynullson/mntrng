## Production config baseline

Этот документ описывает **минимально достаточный baseline** для безопасного конфига в проде.
Он не заменяет `.env.example` и `deploy/env.prod*.example`, а **собирает в одном месте то, что критично для эксплуатации**.

Основные источники правды:
- `.env.example` — dev/local шаблон с пометками по чувствительности.
- `deploy/env.prod.example` — минимальный prod-шаблон.
- `deploy/env.prod.full.example` — расширенный prod-шаблон с безопасными дефолтами.
- `internal/config/runtime_safety.go`, `internal/bootstrap/workerapp/bootstrap.go` — runtime‑проверки.

Везде ниже предполагается `APP_ENV=production`.

### 1. Обязательные переменные для продакшена

- **APP_ENV**  
  - **Назначение**: включает production-ветку поведения и runtime‑проверки.  
  - **Требование**: `APP_ENV=production`.  
  - **Fail-fast**: используется в `config.IsProduction()` и валидациях.

- **DOMAIN, CADDY_EMAIL**  
  - **Назначение**: внешний хост и e‑mail для Let's Encrypt (Caddy).  
  - **Требование**: задать валидный домен и почту (`deploy/env.prod*.example`).  
  - **Fail-fast**: ошибки от Caddy/ACME при неправильных значениях.

- **POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_PORT**  
  - **Тип**: **секретные** (кроме имени БД/порта).  
  - **Требование**: `POSTGRES_PASSWORD` **обязан** быть длинным случайным паролем.  
  - **Fail-fast**: неверные креды приводят к падению API/worker при попытке подключения.

- **INTERNAL_API_BASE_URL, NEXT_PUBLIC_API_BASE_URL**  
  - **Назначение**: маршрутизация фронтенда и Caddy.  
  - **Требование**: в проде `INTERNAL_API_BASE_URL=http://api:8080`, `NEXT_PUBLIC_API_BASE_URL=https://<DOMAIN>`.  
  - **Fail-fast**: ошибки маршрутизации проявятся как 4xx/5xx/timeout на фронте.

- **WORKER_METRICS_TOKEN**  
  - **Тип**: **секретный токен**, минимум 32 символа.  
  - **Требование**: **обязан** быть задан в проде.  
  - **Fail-fast**: `workerapp.RuntimeConfig.Validate()` вернёт ошибку, если `APP_ENV=production` и токен пуст.

### 2. Security guard‑rails (что уже принудительно проверяется)

Эти проверки живут в `config.ValidateAPIRuntimeSafety()` и вызываются при старте API (`cmd/api/main.go`):

- **API_METRICS_PUBLIC**  
  - **Требование** в проде: `false`.  
  - **Fail-fast**: при `APP_ENV=production` и `API_METRICS_PUBLIC=true` стартап API завершится ошибкой.
  - **Смысл**: `/api/v1/metrics` никогда не должен быть публичным без аутентификации.

- **AUTH_COOKIE_SECURE**  
  - **Требование** в проде: `true`.  
  - **Fail-fast**: при `false` API не запустится.

- **AUTH_COOKIE_SAMESITE**  
  - **Требование**: значение `none` **запрещено** в проде.  
  - **Fail-fast**: при `AUTH_COOKIE_SAMESITE=none` API не запустится.

- **BOOTSTRAP_SEED_ENABLED**  
  - **Требование** в проде: `false`.  
  - **Fail-fast**: при `true` API не запустится (защита от случайного сидирования боевой БД).

- **CORS_ALLOWED_ORIGINS**  
  - **Требование**: в проде для доменов, отличных от `localhost`/`127.0.0.1`, **обязан** использоваться `https://`.  
  - **Fail-fast**: при `http://something` (не localhost) API не запустится.

Для worker:

- **WORKER_METRICS_TOKEN**  
  - При `APP_ENV=production` пустое значение приводит к ошибке `WORKER_METRICS_TOKEN is required in production`.  
  - Без токена `/metrics` остаётся публичным только в non‑prod окружениях.

### 3. Чувствительные секреты и токены

Ниже — значения, которые **не должны попадать в репозиторий, логи и issue‑трекер**:

- **POSTGRES_PASSWORD** — пароль БД.  
- **TELEGRAM_BOT_TOKEN_DEFAULT, DEV_LOG_TELEGRAM_TOKEN** — токены ботов Telegram.  
- **SUPER_ADMIN_TELEGRAM_CHAT_ID, DEV_LOG_TELEGRAM_CHAT_ID** — идентификаторы приватных чатов.  
- **AI_INCIDENT_API_KEY** — ключ внешнего AI‑провайдера (когда будет подключён реальный сервис).  
- **SCHEDULER_ACCESS_TOKEN** — access‑токен для `/api/v1/auth/login`, используется планировщиком.  
- **WORKER_METRICS_TOKEN** — токен доступа к worker `/metrics`.  

Рекомендации:
- **никогда** не хранить эти значения в `.env.example` / `deploy/env.prod*.example` в явном виде;
- использовать `CHANGE_ME_*` плейсхолдеры только в примерах;
- в проде подставлять значения через секрет‑хранилище платформы (Vault / cloud secrets / Ansible vars и т.п.), а `.env.prod` держать только на хосте.

### 4. Telegram‑алерты и их ограничения

Переменные:
- **ALERT_FAIL_STREAK, ALERT_COOLDOWN_MIN, ALERT_SEND_RECOVERED** — поведение продуктовых алертов.  
- **TELEGRAM_BOT_TOKEN_DEFAULT, SUPER_ADMIN_TELEGRAM_CHAT_ID** — куда слать продуктовые уведомления.

Это **product‑level алерты** (качество стримов, инциденты по контенту), а не системный мониторинг.

- При пустых токене/чате алерты просто не будут отправляться, но система продолжит работать.  
- Для эксплуатации продакшена рекомендуется:
  - завести отдельного бота/чат для **операционных** алертов (через внешнюю систему мониторинга);
  - не смешивать product‑события и состояние инфраструктуры в один поток.

### 5. AI / diagnostics / OTEL

- **AI_INCIDENT_ENABLED**  
  - По умолчанию `false`.  
  - При `true` сейчас включается **stub‑анализатор** (нет внешних вызовов AI‑провайдера), см. `internal/ai`.  
  - Безопасный дефолт: никаких скрытых внешних запросов.

- **AI_INCIDENT_API_KEY**  
  - Резерв под будущего реального AI‑провайдера.  
  - В текущей версии не используется, но должен рассматриваться как полноценный секрет.

- **OTEL_EXPORTER_OTLP_ENDPOINT**  
  - Опционально включает экспорт трейсинга/метрик через OTEL.  
  - Рекомендуется держать пустым, пока нет согласованной observability‑платформы.

### 6. Scheduler

- **SCHEDULER_ENABLED**  
  - Дефолт в prod‑шаблонах — `false`.  
  - Включайте только если есть:
    - понятная потребность в автоматических ре‑чеках;
    - процесс управления и ротации **SCHEDULER_ACCESS_TOKEN**.

- **SCHEDULER_ACCESS_TOKEN**  
  - По смыслу — обычный `access_token` из `/api/v1/auth/login` с ролью, достаточно привилегированной для планировщика.  
  - При утечке даёт полный доступ в рамках роли.  
  - При `SCHEDULER_ENABLED=true` должен считаться обязательным и храниться как секрет.

### 7. Быстрый чеклист перед запуском в прод

Минимальный чек перед первым прод‑стартом:

1. **Базовые значения**
   - `APP_ENV=production`
   - `DOMAIN`, `CADDY_EMAIL` заданы и совпадают с DNS.
2. **База данных**
   - `POSTGRES_DB`, `POSTGRES_USER` согласованы с `docker-compose.yml`.  
   - `POSTGRES_PASSWORD` — длинный случайный пароль, не использовался в тесте/деве.
3. **Security‑флаги**
   - `BOOTSTRAP_SEED_ENABLED=false`
   - `API_METRICS_PUBLIC=false`
   - `AUTH_COOKIE_SECURE=true`
   - `AUTH_COOKIE_SAMESITE=strict`
   - `CORS_ALLOWED_ORIGINS=https://<DOMAIN>`
4. **Worker metrics**
   - `WORKER_METRICS_PORT=9091` (или свой, но только localhost в прод‑compose).  
   - `WORKER_METRICS_TOKEN` — установлен, хранится как секрет.
5. **Telegram / алерты (если используются)**
   - `TELEGRAM_BOT_TOKEN_DEFAULT`, `SUPER_ADMIN_TELEGRAM_CHAT_ID` заданы.  
   - Есть отдельный канал для операционных алертов (через внешнюю мониторинг‑систему).
6. **Scheduler (по умолчанию выключен)**
   - Если `SCHEDULER_ENABLED=true`, то:
     - `SCHEDULER_ACCESS_TOKEN` установлен;
     - токен завязан на контролируемый сервисный аккаунт;
     - задокументирован способ ротации.

