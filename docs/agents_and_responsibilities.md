# Agents & Responsibilities — HLS Monitoring Platform

Этот документ — ЕДИНСТВЕННЫЙ источник истины по ролям агентов, процессу и контрактам.
Если что-то противоречит этому файлу — считается ошибкой.

---

## 0) Non-Negotiable Rules (жёсткие правила)

- Проект уже частично реализован. Работа продолжается, НЕ переписывается.
- Stack (фиксированный):
  - Go (REST API)
  - Go (Worker)
  - PostgreSQL
  - Redis
  - ffmpeg/ffprobe (ТОЛЬКО Worker)
  - Next.js + TypeScript (TSX)
  - TailwindCSS
  - shadcn/ui
  - Framer Motion
  - Local FS (TTL 30 days)
  - Docker Compose
- API ≠ Worker:
  - API никогда не запускает ffmpeg/ffprobe и любые тяжёлые проверки видео.
  - Worker выполняет тяжёлые операции и анализ.
- Strict multi-tenant:
  - company_id во всех таблицах (кроме companies)
  - company_id берётся только из auth context
  - unscoped select запрещён
- RBAC:
  - super_admin
  - company_admin
  - viewer
- Controlled Registration:
  - register → создаёт pending request
  - approve/reject только super_admin
  - до approve вход невозможен
- Telegram:
  - Alerts bot (инциденты мониторинга)
  - DevLog bot (неформальный рабочий чат): токен и chat ID только из .env (DEV_LOG_TELEGRAM_TOKEN, DEV_LOG_TELEGRAM_CHAT_ID); после каждого коммита отправлять сообщение в Telegram через post-commit hook (см. раздел 9).
  - Telegram login только для approved users
  - все токены/секреты только через ENV
- Окружение и автоматизация (без ручных действий с стороны пользователя):
  - Запуск и БД: только через Docker Compose; миграции и сидер выполняет сервис init при старте.
  - Скриншоты для UI-модулей: обязательно; делает FrontendAgent при сдаче модуля (npm run screenshot:...) или пайплайн (Docker: docker compose --profile screenshot run --rm screenshot); не запрашивать скриншоты у пользователя вручную. Путь screenshots/{module}/{timestamp}.png; без корректного скриншота PASS невозможен.
  - Подключение к БД для тестов/скриптов: через env_dev или через DATABASE_URL из .env при запуске в Docker.
- Audit log обязателен:
  - approve/reject
  - role change
  - stream/project changes
- No monolith:
  - файл > 400 LOC запрещён
  - функция/компонент > 80 LOC — дробить
  - слои http/service/repo/domain обязательны
- Review P0 clean before next module:
  - следующий модуль запрещён, пока ReviewAgent не сказал PASS.

---

## 1) Кто такие агенты (простыми словами)

У нас 4 сабагента (ровно 4, без “потом добавим”):

### 1.1 MasterAgent (бригадир)
- Разбивает модуль на задачи.
- Делает routing: кто следующий.
- Следит, что формат JOB → RESULT → ROUTING соблюдается.
- Следит за P0 чистотой через ReviewAgent.
- Обновляет docs/PROMPTS, когда это нужно для порядка.

### 1.2 BackendAgent (бэкендер)
- Делает Go API + Go Worker + DB/Redis интеграции.
- Гарантирует strict tenant scoping и API ≠ Worker.
- Добавляет audit log события.

### 1.3 FrontendAgent (фронтендер)
- Делает UI на Next.js+TSX + Tailwind + shadcn/ui + Framer Motion.
- Flat дизайн строго.
- Всегда: skeleton loaders, empty/error states, micro animations.
- Всегда: screenshot + self-score >= 9.

### 1.4 ReviewAgent (строгий проверяльщик)
- Делает PASS/BLOCK.
- Проверяет P0: tenant scoping, API≠Worker, правила UI, скриншоты.
- Без PASS — дальше нельзя.

---

## 2) Bible (обязательные файлы для чтения)

Общее правило:
- Если задача затрагивает область (DB/API/UI/Worker), агент ОБЯЗАН прочитать соответствующие файлы до начала.
- Если файла нет — это BLOCKER, агент сообщает MasterAgent, а не “додумывает”.

### MasterAgent — Bible
- PROMPTS/architecture_master.md
- PROMPTS/thresholds_and_rules.md
- PROMPTS/codex_review_checklist.md
- PROMPTS/ui_style_guide.md
- docs/arch_overview.md
- docs/decisions.md
- docs/agents_and_responsibilities.md
- docs/api_contract.md
- docs/schema.md
- docs/frontend_ui_rules.md
- docs/agent_devlog.md

### BackendAgent — Bible
- PROMPTS/architecture_master.md
- PROMPTS/thresholds_and_rules.md
- PROMPTS/codex_review_checklist.md
- docs/decisions.md
- docs/schema.md
- docs/api_contract.md

### FrontendAgent — Bible
- PROMPTS/architecture_master.md
- PROMPTS/ui_style_guide.md
- PROMPTS/codex_review_checklist.md
- docs/frontend_ui_rules.md
- docs/decisions.md
- docs/api_contract.md

### ReviewAgent — Bible
- PROMPTS/codex_review_checklist.md
- PROMPTS/architecture_master.md
- PROMPTS/thresholds_and_rules.md (если проверка касается статусов/алертов)
- PROMPTS/ui_style_guide.md (если UI модуль)
- docs/screenshot_automation.md (если UI модуль — скриншоты)
- docs/decisions.md
- изменённые файлы + diff последнего коммита
- /screenshots/...png (если UI модуль)

---

## 3) JOB → RESULT → ROUTING (обязательный контракт)

Любая работа считается выполненной только если ответ содержит 3 блока: JOB, RESULT, ROUTING.

### 3.1 JOB (что должно быть в задаче)
- ID: уникальный ID (пример: FE-20260222-001)
- ROLE: MasterAgent | BackendAgent | FrontendAgent | ReviewAgent
- CONTEXT_FILES: список файлов/папок, которые нужно прочитать/учесть
- TASK: чёткое описание, что сделать
- REQUIREMENTS: список ограничений
- DEFINITION_OF_DONE: измеримые критерии готовности
- COMMIT_MESSAGE: точное сообщение коммита (если будут изменения), иначе N/A

### 3.2 RESULT (что должен вернуть исполнитель)
- JOB_ID: тот же ID
- STATUS: DONE | BLOCKED
- FILES_CHANGED: список
- COMMIT: hash или N/A
- REPORT: 5–12 строк или структурированно (подполя Summary, Screenshot, Score, Verdict, Findings — для Frontend/Review допустимы подполя)
- RISKS: P0/P1 или N/A
- ROLE=<AgentName> CONFIRMED (строго одной строкой в конце RESULT)

### 3.3 ROUTING (кто следующий)
- NEXT_AGENT: MasterAgent | BackendAgent | FrontendAgent | ReviewAgent
- NEXT_ACTION: 1–3 строки следующего шага
- **ПРОМТ ДЛЯ КОПИРОВАНИЯ (обязательно):** одна строка «кому» и сразу блок кода с текстом для пересылки. Пользователь копирует содержимое блока одной кнопкой (иконка копирования у блока) и вставляет в чат указанного агента. Без этого блока ROUTING считается неполным.

Единый формат (как у BackendAgent — удобно копировать):
1. Строка: **ПРОМТ ДЛЯ СЛЕДУЮЩЕГО АГЕНТА (ИмяАгента):**
2. Сразу ниже — блок кода (тройные обратные кавычки). **Первая строка внутри блока — адрес агента после слеша:** `/master-agent`, `/backend-agent`, `/frontend-agent` или `/review-agent` (в зависимости от того, кому адресуется промт). Со следующей строки — текст для пересылки (JOB или сообщение). Маркеры ---НАЧАЛО КОПИРОВАТЬ--- / ---КОНЕЦ КОПИРОВАТЬ--- не использовать.

Пример:
**ПРОМТ ДЛЯ СЛЕДУЮЩЕГО АГЕНТА (MasterAgent):**
```
/master-agent
BE-README-TEST-001 выполнен: в README добавлен раздел...
```

**ReviewAgent:** результат ревью всегда пересылается мастеру. ReviewAgent обязан **включить в свой ответ в чате** (в теле сообщения пользователю) блок кода «РЕЗУЛЬТАТ ДЛЯ MASTER» — первая строка `/master-agent`, далее полный результат (JOB_ID, Verdict, Findings, Screenshot Review, DevLog, Risks, что делать дальше). Не только записать в файл: блок должен быть в ответе в чате, чтобы пользователь скопировал его одной кнопкой и вставил в чат MasterAgent.

Правило:
- Если формат нарушен или отсутствует промт для копирования — задача считается НЕ выполненной.

**Запрещено: инструкции пользователю.** Агенты (в т.ч. MasterAgent, ReviewAgent) не выводят блоки «Что сделать для PASS» / «выполни в терминале...» / «закоммить...» / «повторно отправить на ревью». Работу выполняют агенты. Ответ агента: NEXT_AGENT + блок ПРОМТ ДЛЯ КОПИРОВАНИЯ с полным JOB для следующего агента. Пользователь только копирует этот блок и вставляет в чат указанного агента; тот сам выполняет задачу (скриншот, коммит, DevLog и т.д.).

---

## 4) UI Screenshot Rules (Mandatory for FrontendAgent)

После завершения каждого визуального модуля FrontendAgent обязан:

1) Сделать screenshot страницы(страниц).
2) Сохранить в репозитории:
   - /screenshots/{module}/{timestamp}.png
3) Провести self-check:
   - grid alignment
   - spacing (отступы)
   - consistency (радиусы/кнопки/таблицы/типографика)
4) Выставить self-score 1–10.
5) Если score < 9:
   - исправить
   - переснять
   - повторить оценку

В RESULT обязательно:
- Screenshot: путь(и)
- Score: число
- Checks: 2–5 коротких строк

ReviewAgent:
- Проверяет screenshot и блокирует при визуальных P0 проблемах.

---

## 5) DevLog Rules (Mandatory)

Файл: docs/agent_devlog.md

**Все агенты после выполнения своей задачи обязаны:**
1) Добавить запись в docs/agent_devlog.md (формат ниже).
2) Закоммитить изменения (код + запись в DevLog; если изменений кода не было — коммит только записи в agent_devlog.md).
3) **Отправить сообщение в Telegram DevLog:** после коммита запустить из корня репозитория `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/devlog_notify.ps1`. Так сообщение гарантированно уходит в чат (post-commit hook при коммите из IDE может не выполниться).

Каждая запись по модулю:
- 5–8 строк
- Summary
- Notes (1–2 строки лёгкого стиля допустимо)
- Commit hash
- Screenshot score (если UI)

Запрещено в DevLog:
- архитектурные решения (только docs/decisions.md)
- секреты/токены/PII

---

## 6) Telegram DevLog Bot Contract (Mandatory)

После каждого коммита (1 задача = 1 коммит) отправляется сообщение в Telegram DevLog bot.

Сообщение состоит из двух частей:

### 6.1 Деловая часть (обязательна)
- Module
- Agent
- Commit hash
- What changed (3–6 bullets)
- Files changed (key files)
- Risks: P0/P1/N/A
- UI: Screenshot score (если UI модуль)

### 6.2 Неформальная часть (обязательна)
- 1–3 строки эмоций
- Мат допустим
- Оскорбления допустимы ТОЛЬКО в сторону багов/задач/процесса (НЕ в сторону людей)
- Запрещено: hate/угрозы/призывы к насилию

### 6.3 Для UI модулей — обязательно фото
DevLog bot дополнительно отправляет screenshot:
- /screenshots/{module}/{timestamp}.png
- score >= 9, иначе переснять

---

## 7) Telegram Alerts Bot Contract (Mandatory)

Alerts bot пишет только про мониторинг потоков:
- OK -> WARN
- WARN -> FAIL
- FAIL -> OK (recovered)

Формат:
- company / project / stream
- status
- top reason (1–2 причины)
- last check time

Анти-спам:
- cooldown на одинаковый статус (по stream)
- streak: подтверждать WARN/FAIL минимум 2 подряд (если так настроено в thresholds)

---

## 8) Branch / Commit Rules

- 1 задача = 1 коммит.
- Сообщения коммитов: area: action
  - db:, api:, worker:, ui:, docs:, infra:
- Запрещены: "WIP", "temp", "fix" без контекста.

---

## 9) Automation: Git hook post-commit (Mandatory)

Токен и ID чата DevLog берутся только из .env: DEV_LOG_TELEGRAM_ENABLED=true, DEV_LOG_TELEGRAM_TOKEN=..., DEV_LOG_TELEGRAM_CHAT_ID=...

- Хук: .githooks/post-commit (sh) вызывает scripts/devlog_notify.ps1.
- devlog_notify.ps1: загружает .env из корня репозитория, передаёт hash последнего коммита и флаг -readSummaryFromGit; go run ./cmd/devnotify/ сам получает subject коммита из git в UTF-8 и выводит module из префикса (api/ui/docs и т.д.).

Первичная настройка (делается ОДИН раз):
1) Убедиться, что в .env заданы DEV_LOG_TELEGRAM_ENABLED=true, DEV_LOG_TELEGRAM_TOKEN, DEV_LOG_TELEGRAM_CHAT_ID.
2) Включить хуки: git config core.hooksPath .githooks

После этого любой commit автоматически отправит сообщение в Telegram DevLog.

**Если сообщения не приходят:** из корня репозитория запусти `.\scripts\devlog_notify_check.ps1`. Скрипт проверит core.hooksPath, наличие .env и переменных DEV_LOG_TELEGRAM_*, затем отправит тестовое сообщение (`go run ./cmd/devnotify/ -test`). Частые причины: не задан `git config core.hooksPath .githooks` (хук не вызывается); пустые TOKEN или CHAT_ID в .env; в окружении хука нет Go в PATH.

**Кто пишет в Telegram:** все агенты (MasterAgent, ReviewAgent, BackendAgent, FrontendAgent). После выполнения задачи агент: 1) делает коммит (код, запись в docs/agent_devlog.md — что релевантно); 2) **обязательно запускает** из корня репо `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/devlog_notify.ps1`, чтобы сообщение ушло в TG (агенты сами обеспечивают отправку; хук post-commit дублирует отправку при коммите из терминала).

---

## 10) Cursor Quickstart (для абсолютного новичка)

Цель: ты не “программируешь агента”, ты просто переключаешься между чатами.

1) Создай 4 агента (описание ролей в `.cursor/agents/`: master-agent.md, backend-agent.md, frontend-agent.md, review-agent.md):
   - MasterAgent / BackendAgent / FrontendAgent / ReviewAgent
2) У всех поставь модель: Auto
3) Начинай всегда с MasterAgent:
   - кидаешь ему JOB модуля
4) MasterAgent в ROUTING скажет, кому дальше:
   - ты копируешь JOB и вставляешь в чат нужного агента
5) После DONE исполнитель роутит на ReviewAgent:
   - ты копируешь туда RESULT
6) Если Review PASS:
   - MasterAgent выдаёт следующий шаг
7) Если Review BLOCK:
   - возвращаешься к исполнителю и исправляешь

Главное правило:
- без PASS от ReviewAgent следующий модуль не начинаем.