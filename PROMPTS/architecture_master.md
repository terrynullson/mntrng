# Architecture Master — HLS Monitoring Platform (Cursor Orchestration)

## A) One-liner
Multi-tenant HLS Monitoring SaaS: Go (API + Worker) + PostgreSQL + Redis + ffmpeg/ffprobe (Worker only) + Next.js (TSX) + local FS (TTL 30 days) + Docker Compose.

---

## B) Цель продукта (что строим)

### B1) Мониторинг HLS потоков (ядро)
Проверки (Worker):
- доступность m3u8/segments
- freshness (насколько “живой” поток)
- declared vs effective bitrate
- freeze detection
- black frame detection
Итоговый статус: OK / WARN / FAIL.

### B2) Управление потоками (админка)
Иерархия:
Company → Project → Stream

Нужно:
- добавлять/редактировать/удалять потоки
- каждый заведённый поток автоматически под мониторингом
- история статусов и инцидентов

### B3) Плеер в админке (обязательно “крутой”)
- кастомизированный HLS player
- удобно переключать потоки
- аккуратный вывод метаданных (не мешает)

### B4) Статистика и аналитика
- аналитическая страница: состояния, тренды, инциденты, частота FAIL/WARN
- таблицы, фильтры, статусы

### B5) Telegram
- Alerts bot: сообщения об инцидентах мониторинга
- DevLog bot: сообщения о работе команды/агентов (с неформальным стилем)
- anti-spam обязателен

### B6) AI (дёшево)
AI подключается только по событию (WARN/FAIL):
- берём метрики + 1–2 кадра
- получаем более точную причину и краткую сводку
- без постоянной аренды дорогих мощностей

---

## C) Стек (фиксированный, не менять)

Backend API: Go (REST)  
Worker: Go (jobs, ffmpeg/ffprobe)  
DB: PostgreSQL  
Queue/state: Redis  
Frontend: Next.js + TypeScript (TSX)  
UI: TailwindCSS + shadcn/ui + Framer Motion  
Storage: Local FS (TTL 30 days)  
Deploy: Docker Compose (api, worker, postgres, redis, frontend)

---

## D) Главные границы (самое важное)

### D1) API ≠ Worker
- API НЕ запускает ffmpeg/ffprobe
- API делает:
  - auth, RBAC, tenant guards
  - CRUD companies/projects/streams
  - отдачу статусов/истории
  - enqueue jobs
- Worker делает:
  - все проверки HLS
  - любые ffmpeg/ffprobe операции
  - скриншоты
  - отправку Alerts в Telegram
  - cleanup TTL

### D2) Strict multi-tenant
- company_id должен быть везде (кроме таблицы companies)
- company_id для доступа берётся только из auth context
- любая выборка без tenant scope = P0 дефект

---

## E) Инварианты данных (обязательно)

- company_id везде (кроме companies)
- audit log на изменения сущностей:
  - stream/project changes
  - role changes
  - approve/reject registration
- alert_state на поток:
  - streak/cooldown/last_sent
- результаты проверок immutable (не переписывать прошлое задним числом)

---

## F) Хранилище скриншотов (Local FS, TTL 30 days)

- Детерминированный путь:
  storage/{company_id}/{stream_id}/{check_id}.jpg  (или аналогичный, единый)
- TTL 30 дней:
  cleanup job — ответственность Worker
- API отдаёт метаданные/прокси-доступ строго по tenant guard

---

## G) Security Canon (обязательно)

- Без auth UI недоступен.
- Public endpoints:
  - GET /api/v1/health
  - controlled registration/auth public endpoints
- Остальные endpoints:
  - auth middleware
  - role guard
  - tenant guard

RBAC:
- super_admin: cross-company по явной политике
- company_admin: read/write внутри своей company
- viewer: read-only внутри своей company

Controlled registration:
- signup создаёт registration_requests со статусом pending
- approve/reject только super_admin
- pending/rejected/disabled не могут аутентифицироваться

Telegram:
- токены через ENV
- без утечек токенов в логах

---

## H) UI Canon (Frontend) — обязательный стандарт

UI должен быть:
- flat
- modern
- professional
- admin-first (плотный, информативный)

Источник истины:
- PROMPTS/ui_style_guide.md
- docs/frontend_ui_rules.md

Обязательно:
- skeleton loaders (не спиннеры)
- empty state и error state
- micro animations 120–360ms, easeOut
- sidebar collapsible + topbar user menu
- Streams table + filters + status badges
- Player page
- Analytics page
- Users page (super_admin)
- Requests page (approve/reject)
- Telegram link page

Запрещено:
- glassmorphism
- neomorphism
- тяжёлые градиенты
- декоративщина без смысла

---

## I) Code Organization (No monolith)

Ограничения:
- новый файл > 400 строк — дробить
- функция/компонент > 80 строк — дробить
- в cmd/* только wiring

Go структура (baseline):
/cmd/api/main.go
/cmd/worker/main.go

/internal/config
/internal/http
/internal/service
/internal/repo
/internal/domain
/internal/queue
/internal/ffmpeg
/internal/telegram
/internal/storage
/internal/observability
/migrations

Запрещено:
- handlers напрямую ходят в БД
- repo вызывает внешние сервисы (telegram/ffmpeg)
- worker использует http handlers

---

## J) Telegram Contracts (Mandatory)

### J1) Alerts bot
События:
- OK -> WARN
- WARN -> FAIL
- FAIL -> OK (recovered)

Содержимое:
- company/project/stream
- status
- причины (1–2)
- время проверки

Анти-спам:
- cooldown + (опционально) streak

### J2) DevLog bot
Триггер: каждый commit (1 задача = 1 коммит)

Сообщение:
1) по делу: module/agent/commit/что сделал/файлы/риски/score
2) эмоции: 1–3 строки, мат допустим, оскорблять людей нельзя

UI:
- обязательно прикладывать screenshot из /screenshots/{module}/{timestamp}.png

---

## K) Cursor Orchestration (Mandatory)

Процесс: JOB → RESULT → ROUTING
Единый источник процесса:
- docs/agents_and_responsibilities.md

Правило:
- следующий модуль только после ReviewAgent PASS (P0 clean)
