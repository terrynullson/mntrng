# Operational Documentation Index

Сводный индекс операционной документации HLS Monitoring. Используйте его как точку входа для деплоя, отката, бэкапов и инцидентов.

## Документы

| Документ | Назначение |
|----------|------------|
| **prod_config_baseline.md** | Критичные production env, fail-fast guardrails, секреты, чеклист перед запуском. |
| **deploy_timeweb.md** | Пошаговый деплой на Timeweb Cloud (Ubuntu, Docker, Caddy). |
| **backup_restore.md** | Что бэкапится, как часто, где хранить, restore и restore drill. |
| **rollback_runbook.md** | Откат приложения и конфигурации, поведение при миграциях. |
| **metrics_health_alerting.md** | Health/ready, метрики API и worker, что считать тревогой, product vs system alerts. |
| **slo_sli_baseline.md** | 3–5 измеримых SLI/SLO. |
| **incident_runbook.md** | Первые шаги при инциденте, быстрая диагностика, rollback, post-incident. |

## Чеклисты (кратко)

### Deploy checklist

- [ ] Прочитан **prod_config_baseline.md** (минимум раздел 7 «Быстрый чеклист перед запуском в прод»).
- [ ] Файл `.env.prod` создан из `deploy/env.prod.example` или `deploy/env.prod.full.example`, все CHANGE_ME заменены.
- [ ] Для деплоя с откатом: `ENV_FILE=.env.prod ./scripts/deploy.sh` (записывает коммит в `.deploy_prev_commit`).
- [ ] Или простой подъём: `ENV_FILE=.env.prod ./scripts/deploy-prod.sh`.
- [ ] Проверка: `curl -fsS http://127.0.0.1:8080/api/v1/health` и `/api/v1/ready` → 200.
- [ ] Проверка UI: логин, список стримов, один стрим.
- [ ] Создан супер-админ при первом деплое: `scripts/bootstrap-superadmin.sh`.

Подробности: **deploy_timeweb.md**.

### Rollback checklist

- [ ] Определён целевой коммит (`.deploy_prev_commit` или известный стабильный).
- [ ] Выполнен `ENV_FILE=.env.prod ./scripts/rollback.sh [commit]`.
- [ ] Контейнеры подняты, health и ready — 200, базовые сценарии UI проверены.
- [ ] Если откат затрагивает схему БД — есть актуальный бэкап и понимание рисков; при необходимости см. **rollback_runbook.md** (раздел про миграции).

Подробности: **rollback_runbook.md**.

### Backup / restore checklist

- [ ] Ежедневный бэкап настроен (cron или аналог). На проде: `ENV_FILE=.env.prod BACKUP_DIR=... RETENTION_DAYS=14 ./scripts/backup-db.sh`.
- [ ] Бэкапы копируются вне сервера (другой хост / объектное хранилище).
- [ ] Restore drill выполнен хотя бы раз (см. **backup_restore.md**, раздел «Restore drill»).

Подробности: **backup_restore.md**.

### Incident triage checklist (первые шаги)

- [ ] Оценка масштаба: какие компоненты недоступны (API, worker, frontend), один тенант или все.
- [ ] Собрать состояние: `curl` к `/api/v1/health`, `/api/v1/ready`; логи `docker compose logs --since=15m api worker postgres redis`.
- [ ] Сохранить логи в файл (не редактировать): `incident-<timestamp>.log`.
- [ ] Приостановить некритичные деплои до выяснения причины.
- [ ] Действовать по **Failure scenarios** ниже и **incident_runbook.md**.

---

## Failure scenarios (troubleshooting matrix)

Краткая таблица: симптом → что проверить → куда смотреть подробнее.

| Симптом | Что проверить первым | Документ/раздел |
|---------|----------------------|------------------|
| **API недоступен** | `docker compose ps` (api), логи api, `curl .../api/v1/ready`. Часто: БД недоступна, миграции, неверный env (DATABASE_URL, CORS, cookie). | incident_runbook.md § 3.1 |
| **Worker не обрабатывает джобы** | Логи worker, `curl http://127.0.0.1:WORKER_METRICS_PORT/health`, метрики `worker_cycle_total{result="error"}`, `worker_job_finalized_total{status="failed"}`. Причины: БД, Redis, таймауты, сбой внешних стримов. | incident_runbook.md § 3.2, metrics_health_alerting.md |
| **PostgreSQL недоступен** | `docker compose ps postgres`, логи postgres, диск/память. API ready будет 503. | incident_runbook.md § 3.1, backup_restore.md (если нужен restore) |
| **Redis недоступен** | `docker compose ps redis`, логи redis и api. Сбой rate limit / сессий. | incident_runbook.md § 2–3 |
| **Frontend жив, API недоступен** | Caddy и маршрутизация на API; `curl` к API по внутреннему адресу (api:8080 или localhost). Проверить INTERNAL_API_BASE_URL, NEXT_PUBLIC_API_BASE_URL, CORS. | deploy_timeweb.md, prod_config_baseline.md |
| **Алерты не отправляются** | Telegram: TELEGRAM_BOT_TOKEN_DEFAULT, SUPER_ADMIN_TELEGRAM_CHAT_ID, настройки компании. Логи worker при отправке. Отличие: product alerts (стримы) vs системные алерты (см. metrics_health_alerting.md). | metrics_health_alerting.md, incident_runbook.md |
| **Рост очереди / застревание обработки** | Метрики worker: `worker_cycle_total`, `worker_job_finalized_total`, длительность джобов. Логи worker на таймауты и ошибки. Проверить БД и Redis, нагрузку на стримы (много failed checks). | incident_runbook.md § 3.2, metrics_health_alerting.md, slo_sli_baseline.md |

Во всех случаях при необходимости отката — **rollback_runbook.md** и наличие актуального бэкапа (**backup_restore.md**).
