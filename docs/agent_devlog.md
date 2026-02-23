# Agent DevLog Protocol

## После своей работы (обязательно для агентов)

Это касается **всех агентов**: MasterAgent, ReviewAgent, BackendAgent, FrontendAgent.

После выполнения задачи каждый агент обязан:
1. **Запись в этот файл** — добавить одну запись по формату ниже (DATE, MODULE, Agent, Commit, Summary, Notes; до 12 строк).
2. **Коммит** — закоммитить изменения (код + эта запись). Если изменений кода не было — закоммитить только запись в этом файле.
3. **Telegram DevLog** — сообщение уйдёт автоматически после коммита (post-commit hook). Ручной вызов devnotify не нужен; нужны включённые хуки (`git config core.hooksPath .githooks`) и переменные в `.env` (DEV_LOG_TELEGRAM_*).

Итог: запись в файл + коммит → и в файле DevLog, и в Telegram.

Полный контракт и настройка хука: `docs/agents_and_responsibilities.md`, разделы 5–6 и 9.

---

## Entry format (mandatory)

[DATE] [MODULE]
Agent: <AgentName>
Commit: <hash>
Summary:
- ...
- ...
- ...
Notes:
1-3 lines short comment (разговорный/эмоциональный тон допускается в рамках guardrails).

## Constraints (mandatory)

- Maximum 12 lines per entry.
- DevLog must not record architecture decisions.
- DevLog must not initiate new tasks.
- Architecture decisions are recorded only in ADR (`docs/decisions.md`).
- Notes may use emotional or conversational tone.
- Russian language is default for `Summary` and `Notes`.
- Non-addressed expressive vocabulary is allowed.
- Notes must not contain insults toward addressees.
- Notes must not contain hate speech or discrimination.
- Notes must not contain secrets, tokens, or PII.

## Example

[2026-02-15] [api-auth]
Agent: BackendAgent
Commit: 1ef5ae9
Summary:
- Добавлен auth middleware.
- Добавлен workflow controlled registration.
- Добавлен RBAC tenant guard.
Notes:
Смоук-тесты прошли, регрессий в runtime не найдено.

[2026-02-15] [api-auth-baseline]
Agent: BackendAgent
Commit: 268cbc7eea7ddb6df73bc02d20b4cea46472a4f8
Summary:
- Добавлены baseline security-gate тесты для login и RBAC edge-cases.
- Синхронизированы schema docs по auth/session tenant-scope ограничениям.
- Подтверждён pass полного test suite для принятого шага.
Notes:
Backfill после принятия шага: completion-notify отправлен через `cmd/devnotify`.

[2026-02-15] [ui-auth-shell-baseline]
Agent: FrontendAgent
Commit: 270ef36d53623be0e69cee011a6df7d0af27091f
Summary:
- Закрыт baseline security-gate: публичные страницы `/login` и `/register`.
- Подтверждён protected shell с `/me` bootstrap, role-aware navigation и logout в topbar.
- Страницы Requests/Users/Streams/Settings приведены к read-only baseline без мутаций.
Notes:
Сборка web прошла, шаг закрыт без изменений backend/runtime API; дрейф по протоколу подчищен.

[2026-02-16] [ui-rbac-mutations-phase2]
Agent: FrontendAgent
Commit: f0136015d007a5a2bb638c8b267881512894d44c
Summary:
- Восстановлены approve/reject действия в `/admin/requests` только для super_admin.
- Возвращены role-management controls в `/admin/users` для super_admin; остальные роли оставлены в read-only.
- В `/streams` возвращён Run check с RBAC-гейтами, в `/settings` восстановлен Telegram link/reconnect flow.
Notes:
Protected shell и auth guard не трогались; сборка web прошла, шаг закрыт без изменений backend/runtime API.

[2026-02-16] [api-admin-users-phase3]
Agent: BackendAgent
Commit: 69c013b9a7df98f52b5173e62160edaad05d60d8
Summary:
- Добавлены `GET /api/v1/admin/users` с фильтрами `company_id/role/status/limit` и safe cap.
- Добавлен `PATCH /api/v1/admin/users/{user_id}/status` с валидацией `active|disabled`.
- Добавлен audit `status_change` с payload (`user_id`, `old_status`, `new_status`, `actor_user_id`).
Notes:
Закрыли админский user-management шаг для Secure Admin UI v2, без дрейфа по RBAC/tenant инвариантам.

[2026-02-16] [ui-admin-users-phase-next]
Agent: FrontendAgent
Commit: 497b277a3538a55e2b0089ed46be31b599577ff9
Summary:
- Переведена страница `/admin/users` на `GET /api/v1/admin/users` с фильтрами `company_id`, `role`, `status`, `limit`.
- Добавлена мутация статуса через `PATCH /api/v1/admin/users/{user_id}/status` и обновление строки таблицы без перезагрузки.
- Усилен RBAC в UI: `super_admin` управляет role/status, `company_admin` и `viewer` работают только в read-only режиме.
Notes:
Protected-shell и auth flow сохранены; сборка web проходит, визуальный канон admin-first удержан.

[2026-02-16] [api-telegram-auth-hardening]
Agent: BackendAgent
Commit: 6e655f9c4c8ce52e8dd3eeb6a8733e72a95a099f
Summary:
- Усилен hardening Telegram auth/link: reason-коды валидации подписи whitelist-only, без утечки секретов.
- Добавлены тесты login/link: signature success/fail, deny для unlinked/disabled, happy path active+approved flow.
- Подтверждён guard `/api/v1/auth/telegram/link`: без auth-контекста доступ запрещён middleware.
Notes:
Политика controlled registration сохранена, шаг закрыт без изменений API surface и SQL семантики.

[2026-02-22] [telegram-delivery-settings]
Agent: BackendAgent / FrontendAgent
Commit: bf978c6 (api), d99ed92 + b83781b (ui), eddadd8 (screenshot)
Summary:
- API: GET/PATCH company telegram delivery settings (tenant-scoped, company_admin+).
- UI: секция «Telegram Alerts (Company)» на /settings (skeleton, empty/error, micro animation).
- Скриншот: screenshots/telegram-delivery-settings/20260222213043.png, score 9; Dockerfile.api на Go 1.24, run-init.sh доработан для стабильного init.
Notes:
Модуль закрыт. ReviewAgent PASS по скриншоту секции /settings (eddadd8).

[2026-02-22] [analytics]
Agent: FrontendAgent
Commit: b0cf9ad
Summary:
- Страница /analytics: flat, skeleton/empty/error, микроанимации 200–280 ms easeOut.
- Скриншот: screenshots/analytics/20260222213536.png, score 9; скрипт npm run screenshot:analytics.
Notes:
Модуль закрыт. ReviewAgent RV-ANALYTICS-001: PASS.

[2026-02-22] [streams]
Agent: FrontendAgent
Commit: 3338eb4
Summary:
- Страница /streams: flat, skeleton/empty/error, микроанимации 200–280 ms easeOut; скрипт npm run screenshot:streams.
- Скриншот: screenshots/streams/20260222215347.png, score 9.
Notes:
Модуль закрыт. ReviewAgent RV-STREAMS-001: PASS.

[2026-02-22] [stream-detail]
Agent: FrontendAgent
Commit: 833aabc
Summary:
- Страница /streams/[streamId] (Stream Player): flat, skeleton/empty/error, микроанимации 200–280 ms easeOut; скрипт npm run screenshot:stream-detail.
- Скриншот: screenshots/stream-detail/20260222221909.png, score 9.
Notes:
Модуль закрыт. ReviewAgent RV-STREAM-DETAIL-001: PASS.

[2026-02-22] [admin-requests]
Agent: FrontendAgent
Commit: 69439d4
Summary:
- Страница /admin/requests: flat, skeleton/empty/error, read-only для не–super_admin, микроанимации 200–280 ms easeOut; сидер test_super_admin, скрипт npm run screenshot:admin-requests.
- Скриншот: screenshots/admin-requests/20260222223722.png, score 9.
Notes:
Модуль закрыт. ReviewAgent RV-ADMIN-REQUESTS-001: PASS.

[2026-02-22] [admin-users]
Agent: FrontendAgent
Commit: fb9f235
Summary:
- Страница /admin/users: flat, skeleton/empty/error, read-only для не–super_admin, микроанимации 200–280 ms easeOut; скрипт npm run screenshot:admin-users.
- Скриншот: screenshots/admin-users/20260222224250.png, score 9.
Notes:
Модуль закрыт. ReviewAgent RV-ADMIN-USERS-001: PASS.

[2026-02-22] [companies]
Agent: FrontendAgent
Commit: e8375ac
Summary:
- Страница /companies: flat, skeleton/empty/error, «Access denied» для не–super_admin, микроанимации 200–280 ms easeOut; скрипт npm run screenshot:companies.
- Скриншот: screenshots/companies/20260222224756.png, score 9.
Notes:
Модуль закрыт. ReviewAgent RV-COMPANIES-001: PASS.

[2026-02-22] [overview]
Agent: FrontendAgent
Commit: c779bd4 (page), отдельный коммит со скриншотом 20260222231707.png
Summary:
- Главная / (Overview): skeleton при !isReady, error «Auth context is unavailable», карточки Account и Quick navigation, микроанимации 200–280 ms easeOut; скрипт npm run screenshot:overview.
- Скриншот: screenshots/overview/20260222231707.png, score 9.
Notes:
Модуль закрыт. ReviewAgent RV-OVERVIEW-002/003: PASS после добавления .png в репо.

[2026-02-23] [login]
Agent: FrontendAgent
Commit: 0eec6a4 (polish+script), 829d91b (screenshot)
Summary:
- Страница /login: flat, микроанимации auth-card и error (120–360 ms easeOut); скрипт npm run screenshot:login.
- Скриншот: screenshots/login/20260223065518.png, score 9.
Notes:
Модуль закрыт. ReviewAgent RV-LOGIN-001: PASS.

[2026-02-23] [register]
Agent: FrontendAgent
Commit: 1d54bc8 (polish+script), 18daa03 + коммит пересъёмки (screenshot 20260223075110)
Summary:
- Страница /register: flat, микроанимации auth-card, form, pending-card и error (120–360 ms easeOut); скрипт npm run screenshot:register (ожидание h1 перед снимком).
- Скриншот: screenshots/register/20260223075110.png, score 9.
Notes:
Модуль закрыт. ReviewAgent RV-REGISTER-002: PASS.

[2026-02-23] [p0-backend-worker]
Agent: ReviewAgent
Commit: N/A
Summary:
- P0-аудит backend/worker (RV-P0-BACKEND-001): вердикт PASS. Tenant scope, API≠Worker, идемпотентность, таймауты, миграции, отсутствие секретов в логах проверены.
- P0 issues: нет. P1 зафиксированы (индексы, поведение ListUsers для super_admin).
Notes:
Аудит завершён. Дальнейшие JOB по плану — на усмотрение MasterAgent.

[2026-02-22] [be-p1-001]
Agent: BackendAgent
Commit: 688df8e
Summary:
- Закрыты P1 из RV-P0-BACKEND-001: миграция 0005_indexes_admin_and_lists (idx_users_company_id_created_at, idx_check_results_company_stream_created, idx_streams_company_project).
- В docs/api_contract.md зафиксировано поведение GET /admin/users: без company_id — cross-company, с company_id — только эта компания (super_admin only).
- docs/schema.md обновлён: порядок миграций и описание индексов 0005.
Notes:
P0-поведение не менялось. Миграции воспроизводимы.

[2026-02-23] [be-alerts-001]
Agent: BackendAgent
Commit: 8482e33
Summary:
- Контракт Telegram Alerts: docs/telegram_alerts_contract.md (переходы OK→WARN, WARN→FAIL, FAIL→OK; формат; антиспам cooldown/streak).
- Ссылки в docs/api_contract.md (§5.8) и docs/decisions.md (ADR-0007). Антиспам и тесты уже в коде.
Notes:
API алерты не шлёт; только Worker.

[2026-02-23] [be-retention-001]
Agent: BackendAgent
Commit: 2aeecca
Summary:
- docs/retention_cleanup.md: TTL 30 дней, батчи, tenant scope, путь storage, конфиг ENV, идемпотентность. ADR-0005 и api_contract §6 обновлены.
Notes:
Cleanup только в Worker; код не менялся.

[2026-02-23] [be-ai-001]
Agent: BackendAgent
Commit: 9213225
Summary:
- Контракт B6: docs/ai_incident_contract.md, ADR-0012 в docs/decisions.md, ссылка в architecture_master.md § B6. Реализация не делалась.
Notes:
Worker-only; API не дергает AI.

[2026-02-23] [be-docs-001]
Agent: BackendAgent
Commit: 8c768bd
Summary:
- README: раздел «Документация» со ссылками на api_contract, schema, telegram_alerts_contract, retention_cleanup, ai_incident_contract, decisions, screenshot_automation; отсылка к DevLog/retention/telegram в .env.example.
Notes:
Только правки README.

[2026-02-23] [be-readme-mig-001]
Agent: BackendAgent
Commit: 01e291f
Summary:
- README: в блок Apply migrations добавлена миграция 0005_indexes_admin_and_lists.up.sql.
Notes:
Список миграций 0001–0005 актуален.

[2026-02-23] [be-readme-test-001]
Agent: BackendAgent
Commit: c3c7acf
Summary:
- README: подраздел «Тесты» (go test ./..., DATABASE_URL для тестов с БД).
Notes:
Только правки README.

[2026-02-23] [fe-readme-build-001]
Agent: FrontendAgent
Commit: a268db1
Summary:
- README: подраздел «Frontend (сборка и запуск)» — npm install, npm run dev, npm run build, npm run start. Проверка: npm run build в web/ прошла (Next.js 14.2.5).
Notes:
Ограничения не выявлены.

[2026-02-23] [be-init-001]
Agent: BackendAgent
Commit: 1c35bb0
Summary:
- scripts/run-init.sh: в цикл миграций добавлена 0005_indexes_admin_and_lists.up.sql; init применяет 0001–0005.
Notes:
Идемпотентность сохранена.

[2026-02-23] [rv-mig-001]
Agent: ReviewAgent
Commit: N/A
Summary:
- Проверка согласованности миграций (RV-MIG-001): run-init.sh, README (Apply migrations), docs/schema.md (apply и rollback) — везде 0001–0005, порядок совпадает, откат обратный. Вердикт PASS.
Notes:
Пропусков и расхождений нет.

[2026-02-23] [be-readme-devlog-001]
Agent: BackendAgent
Commit: 76befd1
Summary:
- README: в раздел «Документация» добавлена ссылка на docs/agent_devlog.md.
Notes:
Только правки README.

[2026-02-23] [be-env-comment-001]
Agent: BackendAgent
Commit: 2df1f35
Summary:
- .env.example: комментарий над группой RETENTION_* (retention cleanup Worker).
Notes:
Только комментарий.

[2026-02-23] [be-readme-aor-001]
Agent: BackendAgent
Commit: 1c26ec1
Summary:
- README: в раздел «Документация» добавлена ссылка на docs/agents_and_responsibilities.md (процесс, роли, JOB→RESULT→ROUTING).
Notes:
Только правки README.

[2026-02-23] [rv-p0-state-001]
Agent: ReviewAgent
Commit: db7e426
Summary:
- P0-ревью состояния кодовой базы (RV-P0-STATE-001): tenant scoping, API≠Worker, UI/скриншоты, общие P0. Правки не вносились.
- Tenant: company_id в запросах api_* репозиториев; ListCompanies и admin ListUsers — исключения по архитектуре (companies / super_admin). API: ffmpeg/ffprobe только в worker. UI: скриншоты по всем модулям в screenshots/{module}/, score 9 в devlog.
Notes:
Вердикт PASS. Закоммитить только эту запись в agent_devlog при отсутствии других изменений.

[2026-02-23] [be-worker-hls-001]
Agent: BackendAgent
Commit: (после коммита подставить hash)
Summary:
- Проверен полный цикл HLS Worker: enqueue (API) → check_jobs (PostgreSQL) → Worker claim → processJob (все 7 проверок) → persist check_result (status + checks) → alert_state → finalize. Tenant scope везде (company_id в repo).
- Все атомарные проверки реализованы и пишутся в checks: playlist, freshness, segments, declared_bitrate, effective_bitrate, freeze, blackframe (+ *_details где нужно). Агрегация FAIL > WARN > OK (checks.AggregateStatuses). Изменений кода не потребовалось.
Notes:
Тесты internal/service/worker проходят. Redis в контракте не используется для очереди — только PostgreSQL check_jobs.

[2026-02-23] [be-telegram-alerts-001]
Agent: BackendAgent
Commit: 0ca9968
Summary:
- Добавлена отправка алертов при переходе OK→WARN: internal/domain (WorkerAlertEventWarn, ComputeWorkerAlertTransition — ветка ok_to_warn с cooldown). WARN→FAIL уже был (streak + cooldown). Recovered без изменений.
- Антиспам сохранён: cooldown по stream_id, streak для FAIL, send_recovered в настройках компании. Токены/chat_id не логируются.
- Обновлён docs/telegram_alerts_contract.md; тесты TestComputeAlertTransition дополнены (ok_to_warn sends, ok_to_warn blocked by cooldown).
Notes:
Только Worker шлёт алерты; tenant scope соблюдён.
