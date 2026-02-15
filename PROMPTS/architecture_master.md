A. One-liner

Multi-tenant HLS Monitoring Platform: Go (API+Worker) + PostgreSQL + Redis + ffmpeg/ffprobe + Next.js (TS/TSX) + local FS + Docker Compose.

B. Цель продукта (что строим)

Мониторинг HLS: m3u8/segments доступность, freshness, declared/effective bitrate, freeze, black frame → OK/WARN/FAIL

Управление потоками Company → Project → Stream

Плеер в админке (HLS) + статус рядом

Telegram alerts + anti-spam

Результаты/скриншоты 30 дней + аналитика

Multi-tenant изоляция

Расширяемость (SRT позже, AI позже)

C. Стек (фиксированный, не менять)

Backend API: Go (REST)

Worker: Go (очередь задач, ffmpeg/ffprobe)

DB: PostgreSQL

Queue: Redis

Frontend: Next.js + TypeScript (TSX)

Storage: локальная FS (VPS), TTL 30 дней

Deploy: Docker Compose (api, worker, postgres, redis, frontend)

D. Негласные “границы” (самое важное)

API НЕ запускает ffmpeg/ffprobe. Любые тяжёлые проверки только Worker.

API делает: CRUD, выдачу статусов/истории, постановку задач в очередь, RBAC/tenant checks.

Worker делает: проверки потоков, скриншоты, анализ, алерты, cleanup.

Multi-tenant строго: никакой выборки без company\_id, никакой “общей” админки без явного режима.

E. Data model invariants (инварианты данных)

company\_id везде (кроме companies)

audit log на изменения сущностей (streams/projects/…)

alert\_state на поток (streak/cooldown/последний алерт)

результаты проверок immutable (не правим задним числом)

F. Хранилище скриншотов

Путь на FS формируется детерминированно: storage/{company\_id}/{stream\_id}/{check\_id}.jpg

TTL 30 дней: cleanup job в Worker

API отдаёт ссылки/метаданные, чтение файла через API (или прокси), но правила доступа по tenant обязательны

G. Набор модулей (чтобы мастер шёл по шагам)

Repo bootstrap + Docker Compose

DB schema + migrations

API contracts (OpenAPI/README endpoints)

CRUD компаний/проектов/стримов + audit

Worker jobs: enqueue + processing skeleton

HLS checkers (m3u8/segments/freshness/bitrate/freeze/blackframe)

Alerts Telegram + anti-spam

Retention cleanup

Frontend admin shell + таблицы/фильтры

Player page + status

Analytics pages

AI (позже)

H. Стандарты разработки

1 шаг = 1 коммит (узкая задача)

Нет TODO, нет “пока так”

Изменения API → обновить контракт/док

Все thresholds — из thresholds\_and\_rules.md, без магических чисел



I. UI Canon (Frontend) — обязательный архитектурный стандарт



UI админки обязан быть:

\- flat

\- modern

\- professional

\- admin-first (плотный, информативный, без декоративности)



Источник истины:

\- PROMPTS/ui\_style\_guide.md

\- docs/frontend\_ui\_rules.md



FrontendAgent обязан читать эти файлы перед любой UI-задачей.



UI стек (обязательный):

\- TailwindCSS

\- shadcn/ui (Button, Badge, Table, Input, Dropdown, Dialog, Tabs)

\- Framer Motion для микро-анимаций



Анимационные правила:

\- длительность 120–360ms

\- easing: easeOut

\- без bounce / elastic / spring exaggeration

\- анимации не должны мешать чтению таблиц



Компонентные требования:

\- Таблицы с фильтрами, поиском и пагинацией

\- StatusBadge (OK/WARN/FAIL/INFO) как единый компонент

\- Skeleton loaders вместо спиннеров

\- Empty state и Error state обязательны



Запрещено:

\- Неоморфизм

\- Glassmorphism

\- Тяжёлые градиенты

\- Декоративные эффекты без бизнес-смысла

\- Несогласованные UI-паттерны



Definition of Done (UI-задачи):

\- Есть loading / empty / error states

\- Используются общие UI-примитивы

\- Нет inline-стилей

\- Нет магических цветов

\- Один шаг = один коммит



J. Code Organization (mandatory) — запрет монофайлов

Цель: поддерживаемая структура кода. Любая реализация должна быть разложена по слоям/пакетам.
Правило: “один файл на весь модуль” запрещён.

Ограничения:
- Любой новый файл > 400 строк = требуется разбиение (обоснование только через ADR).
- Любая функция > 80 строк = требуется разбиение (вынести подфункции/пакеты).
- В cmd/* только wiring (инициализация, роутинг, DI), без бизнес-логики.
- Бизнес-логика и доступ к данным разделены.

Структура Go (baseline):

/cmd/api/main.go
/cmd/worker/main.go

/internal/config        (ENV, конфиг)
/internal/http          (router, handlers, middleware)
/internal/service       (use-cases, бизнес-логика)
/internal/repo          (DB queries, транзакции)
/internal/domain        (types, enums, status models)
/internal/queue         (enqueue/dequeue abstraction)
/internal/ffmpeg        (wrapper/runner for ffmpeg/ffprobe)
/internal/telegram      (client + templates)
/internal/storage       (FS paths, read/write)
/internal/observability (logging, metrics placeholders)
/migrations

Запрещено:
- handlers напрямую ходят в БД
- repo вызывает внешние сервисы (telegram/ffmpeg)
- worker использует http handlers

K. Security Canon (mandatory)

Authentication and authorization are mandatory for API access.

- Public endpoints only:
  - `GET /api/v1/health`
  - controlled registration/auth public endpoints
- All other API endpoints must pass:
  - auth middleware
  - role guard
  - tenant guard

RBAC baseline:
- `super_admin`: cross-company by explicit policy.
- `company_admin`: read/write only inside own company.
- `viewer`: read-only inside own company.

Tenant identity source:
- `company_id` for access control must come from auth context.
- Route/query values are validation inputs only and must be checked against auth tenant context.

Controlled registration:
- Public signup creates `registration_requests` with `pending`.
- Only super_admin can approve/reject and assign role/company.
- Pending/rejected/disabled identities cannot authenticate.

Telegram auth policy:
- Telegram payload signatures must be verified by backend.
- Telegram login/link allowed only for approved flow + active users.
- No token/secret leakage in logs.
