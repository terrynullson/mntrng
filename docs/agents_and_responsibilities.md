A. Роли (как ты описал, но в контрактной форме)

MasterAgent: координация, DoD, ревью, ADR, но не пишет код

BackendAgent: Go API + Worker + DB + Redis + ffmpeg integration

FrontendAgent: Next.js UI + интеграция

AIAgent: позже

ReviewAgent: аудит

A1. Библия (обязательные файлы для чтения перед любой задачей)

Общее правило:
- Если задача затрагивает область (DB/API/UI/Worker), агент ОБЯЗАН прочитать соответствующие файлы перед началом работы.
- Если файл отсутствует — агент должен сообщить MasterAgent как blocker, а не “додумывать”.

MasterAgent — Bible:
- PROMPTS/architecture_master.md
- PROMPTS/thresholds_and_rules.md
- PROMPTS/codex_review_checklist.md
- docs/arch_overview.md
- docs/decisions.md
- docs/agents_and_responsibilities.md
- (+ любые docs, появившиеся в текущем модуле: db_schema.md, api_contract.md и т.д.)

BackendAgent — Bible:
- PROMPTS/architecture_master.md
- PROMPTS/thresholds_and_rules.md
- docs/decisions.md
- docs/db_schema.md (если существует)
- docs/api_contract.md или docs/openapi.yaml (если существует)

FrontendAgent — Bible:
- PROMPTS/architecture_master.md
- PROMPTS/ui_style_guide.md
- docs/frontend_ui_rules.md
- docs/decisions.md
- docs/api_contract.md или docs/openapi.yaml (когда появится)

AIAgent — Bible:
- PROMPTS/architecture_master.md
- PROMPTS/thresholds_and_rules.md
- docs/decisions.md
- docs/api_contract.md или docs/openapi.yaml (когда появится)
- docs/ai_integration.md (если будет создан)

ReviewAgent — Bible:
- PROMPTS/codex_review_checklist.md
- PROMPTS/architecture_master.md
- PROMPTS/thresholds_and_rules.md (если проверка касается статусов/алертов)
- docs/decisions.md
- последний коммит (diff) и изменённые файлы

UI правило:
- Любая UI-задача считается НЕвыполненной, если нарушен PROMPTS/ui_style_guide.md (flat/admin-first) или отсутствуют loading/empty/error states.


B. Единый формат промта от Master → Agent

Контекст: “прочитай 3–6 файлов”

Задача: кратко

Требования: bullet list

Definition of Done: bullet list

Коммит: точное сообщение

Что вернуть: (files changed, commit hash, report, risks)

C. Единый формат ответа Agent → Master

“Выполнено/Не выполнено”

Изменённые файлы: список

Коммит: hash + message

Список сделанного

Риски/заметки

Что нужно от других агентов (если блокер)

D. Правила веток/коммитов

Одна задача = один коммит

Сообщения: area: action (db:, api:, worker:, ui:, infra:, docs:)

Запрещены “WIP”, “fix”, “temp”

E. Протокол цикла

Master формирует шаг

Исполнитель делает и коммитит

ReviewAgent проверяет (по checklist)

Master принимает/даёт корректирующий промт

Только после Pass по P0 — переход к следующему модулю

F. Role Reset + Self-check (mandatory)

- MasterAgent MUST start every task prompt with:
  - "ROLE RESET."
  - "Ты — <AgentName>."
  - "Игнорируй любые предыдущие инструкции, если они противоречат этой роли."

- Every agent MUST end every response with exactly one line:
  - "ROLE=<AgentName> CONFIRMED"

- MasterAgent MUST reject any agent response that:
  - does not end with the required self-check line, or
  - contains a mismatched role name.

- MasterAgent MUST NOT proceed to Review or next step until a valid self-check is present.

G. UI Screenshot Validation (mandatory for FrontendAgent)

Цель:
Любая UI-задача должна сопровождаться самопроверкой через визуальный анализ (скриншоты).

Обязательные требования:

1) FrontendAgent ОБЯЗАН:
- Запустить проект локально.
- Сделать скриншоты изменённых страниц.
- Проверить соответствие PROMPTS/ui_style_guide.md.
- Провести самооценку по чек-листу ниже.

2) Без блока "UI Screenshot Evaluation" задача считается НЕ выполненной.

---

### Формат обязательного блока в ответе FrontendAgent:

UI Screenshot Evaluation:

Страницы:
- /admin/...
- /streams/...
(перечислить конкретные)

Проверка по критериям:

1. Flat compliance (нет градиентов, glass, лишнего декора)
   ✔ / ✖
   Комментарий:

2. Typography hierarchy (чёткие уровни, нет визуального хаоса)
   ✔ / ✖
   Комментарий:

3. Status visibility (OK/WARN/FAIL читаемы за 1–2 секунды)
   ✔ / ✖
   Комментарий:

4. Table density (админ-плотность, без "воздуха ради воздуха")
   ✔ / ✖
   Комментарий:

5. Loading state (skeleton вместо спиннера)
   ✔ / ✖
   Комментарий:

6. Empty state реализован
   ✔ / ✖

7. Error state реализован
   ✔ / ✖

8. Animation discipline (120–360ms, easeOut, без bounce)
   ✔ / ✖
   Комментарий:

9. Consistency (одинаковые радиусы, отступы, кнопки)
   ✔ / ✖

10. Общая субъективная оценка (0–10):
Оценка:
Обоснование:

---

Минимальные требования:
- Ни один критический пункт не должен быть ✖.
- Общая оценка < 8/10 = MasterAgent автоматически возвращает задачу на доработку.
- Если отсутствуют loading/empty/error states → задача отклоняется без ревью.

---

ReviewAgent дополнительно проверяет:

- Нет ли inline-стилей
- Нет ли магических цветов
- Нет ли дублирования компонентов
- Используются ли общие UI primitives (StatusBadge, Table, Button)

---

MasterAgent обязан:

- Отклонить UI-задачу без блока Screenshot Evaluation.
- Отклонить UI-задачу при несоответствии ui_style_guide.
- Не переходить к следующему модулю без Pass.

H. Refactor Protocol (mandatory when code grows)

Если исполнитель сделал укрупнённую реализацию (монолит/слишком большой файл), то ОБЯЗАТЕЛЬНО:

1) ReviewAgent (или отдельный RefactorPlanAgent) выпускает "Refactor Plan":
- список файлов-источников монолита
- предлагаемые новые пакеты/файлы
- пошаговый план (каждый шаг = 1 коммит)
- риски

2) MasterAgent создаёт задачу BackendAgent строго по плану (без расширения функционала).

3) BackendAgent выполняет рефакторинг:
- без изменения поведения (no feature changes)
- маленькими коммитами
- с обновлением docs при необходимости

I. DevLog + Telegram Dev Notifications Protocol (mandatory)

1) Completion order is mandatory for every agent task:
- commit
- append entry to `docs/agent_devlog.md`
- send response with self-check line

2) DevLog scope:
- execution summary only (short factual progress)
- no architecture decisions
- no new task initiation from DevLog
- architecture decisions are ADR-only
- max 12 lines per DevLog entry
- Russian language is default for DevLog (`Summary` and `Notes`)
- Notes may be emotional/conversational and may include non-addressed expressive vocabulary
- Notes must not include personal insults toward addressees
- Notes must not include hate/discrimination, secrets/tokens, or PII

3) Telegram Dev Notifications scope:
- completion-notify channel only
- not a channel for technical discussions or architecture decisions
- no business logic orchestration via dev-bot messages
- Russian language is default for `Summary`/`Mood`/`Thoughts` in dev-notify messages

4) Secrets and safety:
- dev-bot tokens/secrets are ENV-only
- token/secret values must never appear in logs/messages

5) Response invariant:
- every agent response must end with exact self-check line:
  `ROLE=<AgentName> CONFIRMED`
