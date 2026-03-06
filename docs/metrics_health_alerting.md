# Metrics, Health & Alerting Baseline

Операционный минимум: что смотреть, что считать тревогой, как отличать product alerts от system-health.

## Health и Ready

| Endpoint | Сервис | Назначение |
|----------|--------|------------|
| `GET /api/v1/health` | API | Лёгкая проверка: процесс жив, отвечает. Не проверяет БД. |
| `GET /api/v1/ready` | API | Готовность к трафику: проверяется подключение к PostgreSQL (`db.Ping`). При недоступности БД — 503. |
| `GET /health` (порт WORKER_METRICS_PORT, по умолчанию 9091) | Worker | Лёгкая проверка: процесс worker жив. Без проверки БД/очередей. |

Использование:

- Load balancer / reverse proxy (Caddy): использовать **ready** для маршрутизации к API, не health.
- Мониторинг доступности: опрашивать **health** и **ready** по таймауту (например 5–10 с). Два подряд 5xx или таймаут — повод для алерта.

## Метрики

### API

- **Endpoint**: `GET /api/v1/metrics` (Prometheus format).
- **Доступ**: в production при `API_METRICS_PUBLIC=false` доступ только для аутентифицированных пользователей с ролью `super_admin`. Не открывать публично.
- Содержимое: стандартные Go/Prometheus метрики (goroutines, memory, http request duration и т.д.), плюс метрики, зарегистрированные в приложении.

### Worker

- **Endpoint**: `GET /metrics` на порту worker (по умолчанию 9091), только localhost в prod.
- **Аутентификация**: в production обязателен `WORKER_METRICS_TOKEN`; запрос с заголовком `Authorization: Bearer <token>`.
- Ключевые метрики для операционного мониторинга:
  - `worker_cycle_total{result="ok"|"error"}` — циклы воркера;
  - `worker_cycle_total{result="error"}` — рост означает сбои цикла (БД, Redis, таймауты);
  - `worker_job_finalized_total{status="ok"|"failed"}` — завершённые джобы;
  - `worker_job_finalized_total{status="failed"}` — рост без компенсирующего роста ok — повод разобраться;
  - `worker_job_duration_seconds` — латенция обработки джобов;
  - `worker_retention_cleanup_total{result="ok"|"error"}` — очистка по retention.

Что считать тревогой (baseline):

- **API**: ready перестаёт отдавать 200 (или таймаут) — API или БД недоступны.
- **Worker**: устойчивый рост `worker_cycle_total{result="error"}`; доля failed джобов резко растёт; worker health перестаёт отвечать 200.

## Product alerts vs system-health

- **Product alerts (Telegram и т.п.)**: уведомления о качестве стримов (падение, фриз, черный кадр, битрейт). Настраиваются через `TELEGRAM_BOT_TOKEN_DEFAULT`, `ALERT_*`, компании/проекты. Это не замена системному мониторингу.
- **System-health**: доступность API/worker/БД/Redis, рост ошибок в метриках, падение health/ready. Рекомендуется:
  - опрашивать health/ready и при необходимости worker `/metrics` из отдельной системы мониторинга (Prometheus + Alertmanager, Uptime Robot, и т.д.);
  - не полагаться только на Telegram-алерты приложения для инфраструктурных сбоев.

Разделение:

- Сбой доставки продуктовых алертов (нет сообщений в Telegram при падении стрима) — проблема конфигурации алертов или Telegram API.
- Сбой доступности API/worker/БД — инфраструктурный инцидент; реагировать по runbook (incident_runbook, failure playbooks).

## Минимальный набор сигналов для мониторинга

1. **API ready** — HTTP 200 с разумным таймаутом (например 5 с). Иначе: алерт «API/DB недоступны».
2. **Worker health** — HTTP 200 на `http://127.0.0.1:WORKER_METRICS_PORT/health`. Иначе: алерт «Worker не отвечает».
3. **Worker cycle errors** — темп прироста `worker_cycle_total{result="error"}`. Резкий рост или стабильно ненулевой темп — расследование.
4. **Worker job failures** — доля или темп `worker_job_finalized_total{status="failed"}`. Существенный рост — расследование.
5. **PostgreSQL** — доступность (через API ready или отдельный проверяющий запрос к БД).
6. **Redis** — используется API для rate limiting и сессий; при недоступности Redis часть запросов может падать. При наличии метрик по Redis — следить за доступностью.

Реагирование: по `docs/incident_runbook.md` и `docs/ops_index.md` (failure playbooks).
