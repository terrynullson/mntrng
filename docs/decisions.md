# Architecture Decision Records

## ADR-0001: Worker-only ffmpeg/ffprobe execution

### Context
Проверки HLS (парсинг сегментов, вычисление effective bitrate, freeze/black frame) ресурсоёмкие и потенциально долгие. Архитектурная граница фиксирована: API обслуживает HTTP-контракты и не должен выполнять тяжёлые медиа-операции.

### Decision
`ffmpeg` и `ffprobe` запускаются только в Worker-процессе. API выполняет только CRUD, tenant/RBAC-проверки, чтение статусов и постановку задач в очередь/таблицу `check_jobs`.

### Consequences
- API остаётся предсказуемым по latency и не блокируется вычислительными задачами.
- Горизонтальное масштабирование API и Worker возможно независимо.
- Все медиа-зависимости и таймауты изолируются в Worker-контуре.

### Alternatives considered
- Запуск `ffmpeg/ffprobe` в API-процессе: отклонено из-за деградации отклика API и нарушения архитектурной границы `API != Worker`.
- Выделенный отдельный media-сервис на baseline: отклонено как преждевременное усложнение.

## ADR-0002: Strict multi-tenant isolation via company_id and DB constraints

### Context
Платформа обслуживает несколько компаний в единой БД. Критическое требование P0: отсутствие межтенантных утечек данных и невозможность записи/чтения без tenant scope.

### Decision
Во всех таблицах, кроме `companies`, обязателен `company_id NOT NULL`. Связи между сущностями строятся через составные внешние ключи с `company_id` (например, `(stream_id, company_id)`), а API и Worker обязаны фильтровать запросы по `company_id`.

### Consequences
- Изоляция поддерживается одновременно на уровне приложения и БД.
- Ошибки в приложении (неполный WHERE) компенсируются ограничениями БД и FK-проверками.
- Сложность SQL возрастает из-за составных ключей, но это осознанная цена за безопасность tenant-границ.

### Alternatives considered
- Изоляция только в коде без DB-ограничений: отклонено как небезопасное решение.
- Отдельная БД на tenant: отклонено для baseline из-за высокой операционной стоимости.

## ADR-0003: Local filesystem storage for screenshots

### Context
Скриншоты используются для диагностики проверок и должны храниться с TTL 30 дней. На baseline приоритет: простота эксплуатации в single-VPS/docker-compose окружении.

### Decision
Скриншоты хранятся в локальной FS по детерминированному пути `storage/{company_id}/{stream_id}/{check_id}.jpg`. В БД сохраняется путь (`check_results.screenshot_path`), а доступ к файлам проходит через API с tenant-проверкой.

### Consequences
- Минимальная инфраструктурная сложность и быстрый старт без S3-совместимого хранилища.
- Требуется контроль диска и регулярный cleanup устаревших файлов.
- При миграции на объектное хранилище потребуется адаптация слоя доступа, но формат `screenshot_path` это допускает.

### Alternatives considered
- S3/MinIO с пресайн-ссылками: отклонено для baseline как избыточное усложнение.
- Хранение скриншотов в PostgreSQL (BYTEA): отклонено из-за роста БД и ухудшения I/O.

## ADR-0004: Redis-backed queueing with PostgreSQL job state

### Context
Проверки должны выполняться асинхронно и устойчиво к временному падению воркера. Нужны быстрое распределение задач и постоянный журнал состояний.

### Decision
Для доставки задач используется Redis-очередь, а жизненный цикл задачи фиксируется в PostgreSQL (`check_jobs`: `queued/running/done/failed`). Идемпотентность достигается ограничением `UNIQUE(stream_id, planned_at)`.

### Consequences
- Быстрое enqueue/dequeue и разделение ответственности между транспортом (Redis) и источником истины (PostgreSQL).
- Возможна безопасная переобработка после сбоев за счёт статусов и идемпотентного ключа.
- Появляется необходимость согласования Redis-сообщений и DB-статусов при ошибках выполнения.

### Alternatives considered
- Только PostgreSQL как очередь (`FOR UPDATE SKIP LOCKED`): отклонено на baseline в пользу более простой очередной модели с Redis.
- Брокеры Kafka/RabbitMQ: отклонено как избыточные для текущего масштаба.

## ADR-0005: 30-day retention via worker cleanup job

### Context
Исторические результаты и скриншоты нужны для аналитики, но не должны расти бесконтрольно. В архитектуре зафиксирован TTL 30 дней.

### Decision
Worker выполняет периодический cleanup: удаляет результаты проверок старше 30 дней и соответствующие файлы скриншотов в FS. API не выполняет retention-задачи и не удаляет данные пакетно.

### Consequences
- Контролируемый рост хранилища БД и диска.
- Поведение согласовано с границей `API != Worker`: тяжёлые фоновые операции остаются в Worker.
- По истечении TTL старые данные недоступны для аналитики; это принимаемое ограничение baseline.

### Alternatives considered
- Бессрочное хранение: отклонено из-за неограниченного роста затрат.
- Cleanup в API по пользовательскому запросу: отклонено из-за нагрузки на API и нестабильного SLA.

## ADR-0006: Mandatory tenant-scoped API (company_id in routes and queries)

### Context
Даже при корректной схеме БД межтенантные утечки возможны через неверно спроектированные API-маршруты или запросы без tenant-фильтра. Требуется единый контракт tenant-scoping для всех read/write операций.

### Decision
Любой маршрут, работающий с tenant-данными, обязан быть company-scoped: `company_id` передаётся в route/контексте запроса и обязателен для всех запросов в БД. Шаблон запроса: фильтрация и join выполняются с `company_id` на каждом шаге, без исключений и без "global list" режима по умолчанию.

### Consequences
- Явная и проверяемая tenant-граница в API-контрактах и SQL.
- Проще аудитировать код на соответствие P0-требованию "нет select без tenant scope".
- Усложняется дизайн некоторых эндпоинтов (нельзя короткие unscoped URL), но это снижает риск утечек данных.

### Alternatives considered
- Tenant scope только из JWT без явного параметра в маршруте: отклонено как менее прозрачное и сложнее проверяемое решение.
- Поддержка unscoped admin-endpoints в baseline: отклонено, так как противоречит strict multi-tenant модели.

## ADR-0007: Tenant-scoped Telegram alert delivery in Worker

### Context
Нужна отправка уведомлений в Telegram при деградации/падении потоков с anti-spam (streak/cooldown/recovered). Критично сохранить strict multi-tenant модель: любые операции уведомлений должны быть tenant-scoped и не допускать unscoped fan-out. Также архитектурная граница фиксирована: `API != Worker`.

### Decision
Отправка Telegram-уведомлений выполняется только в Worker. Любая попытка доставки должна быть tenant-scoped по `company_id` и привязана к `stream_id`.

Решение "слать/не слать" принимает anti-spam движок на основе таблицы `alert_state`:
- streak: отправлять alert только после K подряд FAIL
- cooldown: после отправки молчать T минут
- recovered (опционально): при переходе FAIL->OK отправлять "recovered" с тем же cooldown

Telegram конфигурация и секреты (bot token и т.п.) хранятся в ENV/секретах Worker. В логах и audit payload запрещено выводить токены/секреты.

### Consequences
- Соблюдается граница `API != Worker`: API не выполняет доставки и не блокируется внешними вызовами.
- Multi-tenant соблюдён: каждая доставка и состояние anti-spam однозначно привязаны к `company_id`.
- Поведение уведомлений становится детерминированным и аудируемым через `alert_state` и результаты доставок.
- Можно вводить per-company лимиты/настройки без unscoped режима.

### Alternatives considered
- Отправка Telegram из API: отклонено из-за нарушения `API != Worker` и риска деградации SLA.
- Unscoped доставка без company scope: отклонено как нарушение strict multi-tenant.

## ADR-0008: Preserve company audit history after company deletion

### Context
`audit_log` stores operational and security-relevant history. For incident analysis and compliance, deleting a company must not erase historical audit entries for that company.

### Decision
`audit_log.company_id` remains `NOT NULL` and indexed, but the foreign key cascade `audit_log.company_id -> companies(id) ON DELETE CASCADE` is removed by migration `0003_preserve_company_audit_history`. API writes company create/update/delete audit records before commit, and audit history remains after `DELETE /companies/{company_id}`.

### Consequences
- Company audit trail is preserved after tenant deletion.
- Company lifecycle operations keep an append-only audit timeline for investigations.
- `audit_log.company_id` may reference a deleted company id by design; this is accepted to preserve history.

### Alternatives considered
- Keep `ON DELETE CASCADE`: rejected because it deletes audit history and breaks forensic traceability.
- Keep FK with `RESTRICT/NO ACTION`: rejected because company deletion would be blocked.
- `ON DELETE SET NULL`: rejected because `company_id` must stay `NOT NULL`.

## ADR-0009: Mandatory authentication and RBAC with tenant scope from auth context

### Context
Platform endpoints now expose tenant CRUD, check lifecycle operations, and privileged administration actions. P0 security requires deterministic protection against unauthenticated access, role abuse, and tenant escapes.

### Decision
All API endpoints are authenticated by default except explicitly public endpoints (`/api/v1/health`, public auth endpoints). Access control is enforced by RBAC roles:
- `super_admin`: cross-company access by policy.
- `company_admin`: write/read only inside own company scope.
- `viewer`: read-only inside own company scope.

Tenant scope is derived from authenticated user context (`company_id` in auth context) and validated against route tenant (`/companies/{company_id}/...`). Route/query input is never trusted as the source of tenant identity.

### Consequences
- Unauthorized requests are consistently rejected with typed JSON envelope (`unauthorized`/`forbidden`/`tenant_scope_required`).
- Tenant escape attempts are blocked before handler execution.
- Existing APIs remain tenant-scoped while gaining explicit auth+role guard.

### Alternatives considered
- Optional auth per endpoint: rejected due high risk of accidental unprotected routes.
- Tenant scope only from route/query without auth context binding: rejected as unsafe for strict multi-tenant.
- Role checks only inside business handlers: rejected in favor of centralized middleware guard.

## ADR-0010: Controlled registration and Telegram auth policy (approved + active only)

### Context
Direct self-service account creation is incompatible with baseline tenant governance. Registration must be moderated by super-admins, while Telegram-based login/linking must be allowed only for approved and active users.

### Decision
Public registration creates `registration_requests` with `pending` status. User accounts are created/activated only via super-admin approval workflow:
- list pending requests,
- approve (assign `company_id` + role, create active user),
- reject.

Audit log entries are mandatory for approve/reject and role-change actions.

Telegram integration policy:
- super-admin notification is sent when a new registration request is created (best-effort, no secret leakage),
- Telegram login payload is signature-verified server-side,
- Telegram login is allowed only for approved flow users with `active` status and existing `user_telegram_links`,
- pending/rejected/disabled users cannot authenticate via password or Telegram.

### Consequences
- Access onboarding is moderated and traceable.
- Registration decisions and role escalations are auditable.
- Telegram auth remains bound to controlled account lifecycle and active status.

### Alternatives considered
- Auto-approve registration: rejected as incompatible with controlled onboarding.
- Telegram auth without signature verification: rejected due impersonation risk.
- Allow pending/rejected/disabled users to login: rejected by security policy.

## ADR-0011: Agent DevLog protocol and Telegram Dev Notifications boundary

### Context
The delivery workflow requires a lightweight execution journal for completed steps and a simple notification signal for completion events. Without explicit boundaries, DevLog or dev-notification channels can drift into architecture decision-making, task planning, or runtime control.

### Decision
A dedicated DevLog protocol is mandatory in `docs/agent_devlog.md` with a compact entry format and explicit constraints:
- maximum 12 lines per entry,
- execution summary only (non-normative for architecture),
- no architecture decisions and no new task initiation.

Notes in DevLog entries may use emotional/conversational tone, but must not include insults toward addressees, hate/discrimination, secrets/tokens, or PII.

Telegram Dev Notifications are restricted to completion-notify usage only:
- notify stage/task completion,
- do not discuss/approve architecture,
- do not trigger business logic or workflow control.

All architecture decisions remain ADR-only. Dev-notification secrets/tokens are stored only in ENV and must never be logged.

### Consequences
- Completion history is standardized and easy to audit.
- Architectural governance remains centralized in ADR, avoiding split sources of truth.
- Dev Telegram channel stays operationally simple and low-risk.
- Secret handling remains consistent with security baseline (no token leakage in logs).
- Team communication in DevLog Notes can stay expressive without weakening safety and governance boundaries.

### Alternatives considered
- Use DevLog as an architecture decision source: rejected due governance ambiguity and decision drift.
- Use Telegram dev-bot as interactive planning/control channel: rejected due boundary violations and security risk.
- Skip DevLog and rely only on commit history: rejected because commit logs do not enforce a uniform, compact task journal protocol.
