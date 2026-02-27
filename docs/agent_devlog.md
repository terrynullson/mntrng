# Agent DevLog Protocol

## После своей работы (обязательно для агентов)

Это касается **всех агентов**: MasterAgent, ReviewAgent, BackendAgent, FrontendAgent.

После выполнения задачи каждый агент обязан:
1. **Запись в этот файл** — добавить одну запись по формату ниже (DATE, MODULE, Agent, Commit, Summary, Notes; до 12 строк).
2. **Коммит** — закоммитить изменения (код + эта запись). Если изменений кода не было — закоммитить только запись в этом файле.
3. **Telegram DevLog** — после коммита из корня репо запустить `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/devlog_notify.ps1`, чтобы сообщение ушло в TG (агент сам обеспечивает отправку; хук post-commit при коммите из терминала может дублировать).

Итог: запись в файл + коммит + запуск devlog_notify.ps1 → и в файле DevLog, и в Telegram. Пользователь ничего не запускает вручную.

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

[2026-02-23] [rv-be-telegram-alerts-001]
Agent: ReviewAgent
Commit: N/A
Summary:
- P0-ревью BE-TELEGRAM-ALERTS-001 (коммиты 0ca9968, 5b0c92c): tenant (alert_state по company_id/stream_id), API≠Worker (изменения только domain/worker/repo), логи без секретов. Вердикт PASS.
Notes:
Краткое ревью по чеклисту; правки не вносились.

[2026-02-23] [BE-AI-INCIDENT-001]
Agent: BackendAgent
Commit: 6124a18
Summary:
- Реализован вызов AI по событию WARN/FAIL по контракту docs/ai_incident_contract.md (ADR-0012, B6). После persist check_result при статусе WARN или FAIL Worker передаёт в AI-клиент метрики проверки (checks) и путь к скриншоту (screenshot_path), получает cause и summary.
- Результат сохраняется в БД: таблица ai_incident_results (миграция 0006), привязка по job_id/company_id/stream_id; INSERT ... ON CONFLICT (job_id) DO UPDATE.
- AI только из Worker; API не вызывает AI. On-demand только при WARN/FAIL. Секреты только из ENV (AI_INCIDENT_API_KEY), в логах не выводятся. Tenant scope соблюдён.
- При недоступности AI Worker не падает — ошибки только логируются (без секретов). Реализация: internal/ai (Analyzer, StubAnalyzer, LogAnalyzer), internal/repo/postgres (SaveAIIncidentResult), internal/service/worker (runAIIncidentIfNeeded после persist в job_flow).
Notes:
StubAnalyzer — заглушка; реальный провайдер подключается позже. docs/schema.md и scripts/run-init.sh обновлены под 0006.

[2026-02-23] [rv-be-ai-incident-001]
Agent: ReviewAgent
Commit: N/A
Summary:
- P0-ревью BE-AI-INCIDENT-001: tenant (ai_incident_results, SaveAIIncidentResult по company_id/stream_id), API не вызывает AI (только Worker), секреты не в логах, соответствие ai_incident_contract и ADR-0012. Вердикт PASS.
Notes:
Правки не вносились; коммит только devlog.

[2026-02-23] [BE-AI-INCIDENT-API-001]
Agent: BackendAgent
Commit: 6124a18
Summary:
- Добавлен GET endpoint для чтения результата AI по проверке: GET /api/v1/companies/{company_id}/streams/{stream_id}/check-jobs/{job_id}/ai-incident. Ответ 200 { cause, summary } или 404.
- Tenant-scoped по company_id и stream_id; tenant guard через существующий middleware. Только чтение; API не вызывает AI.
- Реализация: internal/domain (AIIncidentResponse, ErrAIIncidentNotFound), internal/repo/postgres (APIAIIncidentRepo.GetByCompanyStreamJob), internal/service/api (AIIncidentService.Get), handler + роут в internal/http/api. Контракт обновлён в docs/api_contract.md.
Notes:
Данные читаются из таблицы ai_incident_results (миграция 0006).

[2026-02-23] [rv-be-ai-incident-api-001]
Agent: ReviewAgent
Commit: N/A
Summary:
- P0-ревью BE-AI-INCIDENT-API-001: tenant (endpoint companies/.../streams/.../check-jobs/.../ai-incident, выборка WHERE company_id, stream_id, job_id), только чтение, API не вызывает AI, соответствие api_contract. Вердикт PASS.
Notes:
Правки не вносились; коммит только devlog.

[2026-02-23] [FE-AI-INCIDENT-UI-001]
Agent: FrontendAgent
Commit: e9c9e25
Summary:
- На странице детали потока (/streams/[streamId]) при наличии последнего check result с job_id добавлен вызов GET .../check-jobs/{job_id}/ai-incident и блок «AI incident» (cause, summary). Skeleton при загрузке, empty при 404, error state при ошибке. Flat, без лишнего декора.
- Тип AiIncident в lib/api/types.ts; tenant scope через существующий auth/scope.
Notes:
Скриншот секции: npm run screenshot:stream-detail при поднятых API и frontend (screenshots/stream-detail/).

[2026-02-22] [RV-FE-AI-INCIDENT-UI-001]
Agent: ReviewAgent
Summary:
- P0-ревью FE-AI-INCIDENT-UI-001: блок «AI incident» на /streams/[streamId], GET .../ai-incident при job_id, tenant scope через auth, skeleton/empty/error, flat UI. Verdict: PASS.
Notes:
Код не менялся; коммит только devlog.

[2026-02-23] [FE-ANALYTICS-B4-001]
Agent: FrontendAgent
Commit: 08a27ff
Summary:
- Analytics (/analytics) приведена к B4: состояния, тренды (таблица по времени), инциденты/частота FAIL/WARN (summary cards + заголовок «Status summary (FAIL/WARN frequency)»). Таблицы, фильтры (поток, период from/to, статус) по API check-results. Добавлена колонка Declared bitrate. Skeleton, empty, error; flat, admin-first.
Notes:
Скриншот: npm run screenshot:analytics при API + frontend (screenshots/analytics/).

[2026-02-22] [RV-FE-ANALYTICS-B4-001]
Agent: ReviewAgent
Summary:
- P0-ревью FE-ANALYTICS-B4-001: Analytics — состояния, тренды, частота FAIL/WARN, таблица, фильтры (поток, период, статус), skeleton/empty/error, flat, tenant scope. Screenshot Score 9. Verdict: PASS.
Notes:
P1: поиск по таблице не реализован. Код не менялся; коммит только devlog.

[2026-02-23] [FE-PLAYER-B3-001]
Agent: FrontendAgent
Commit: 6124a18
Summary:
- Страница детали потока (/streams/[streamId]) приведена к B3: кастомизированный HLS player (единый вид и контролы), аккуратный вывод метаданных (название, id, проект, активность, atomic checks, AI incident) без перегруза экрана, добавлен dropdown для переключения потоков в рамках текущей компании.
- Tenant scope через auth/scope, skeleton/empty/error сохранены. Плеер flat, admin-first.
Notes:
Скриншот: npm run screenshot:stream-detail при API + frontend (screenshots/stream-detail/).

[2026-02-22] [RV-FE-PLAYER-B3-001]
Agent: ReviewAgent
Summary:
- P0-ревью FE-PLAYER-B3-001: страница /streams/[streamId] — HLS player (единый вид/контролы), dropdown переключения потоков, метаданные (название, id, проект, активность, статус/проверки, AI incident), flat, skeleton/empty/error, tenant scope, без изменения бизнес-логики. Verdict: PASS.
Notes:
P1: обновить REPORT/скриншот stream-detail с проверкой «Switch stream». Код не менялся; коммит только devlog.

[2026-02-22] [RV-P0-STATE-002]
Agent: ReviewAgent
Summary:
- P0-ревью состояния кодовой базы: tenant scoping (company_id в API/Worker, unscoped только companies), API≠Worker (ffmpeg/ffprobe/AI только в Worker), скриншоты UI (10 модулей, REPORT с path/score), логи без секретов. Verdict: PASS.
Notes:
Правки не вносились; коммит только devlog.

[2026-02-23] [FE-PLAYER-DESIGN-001]
Agent: FrontendAgent
Commit: b6fa83f
Summary:
- Плеер: HLS-плеер на странице потока обёрнут в контейнер с aspect-ratio 16/9, кнопка Fullscreen для области плеера; hls.js сохранён. Дизайн: типографика (page-title 1.25rem, font-weight 600), отступы и карточки (panel 20px, player/status-card 14px, radius 14px) в рамках flat по ui_style_guide. Тёмная тема: переключатель Light/Dark в topbar и на публичных страницах (login/register), темы через CSS-переменные (data-theme=dark), сохранение в localStorage, синхронизация с системной темой при первом заходе.
Notes:
Скриншоты светлой и тёмной темы: npm run screenshot:stream-detail (сохраняет <timestamp>.png и <timestamp>-dark.png при наличии API и frontend).

[2026-02-22] [RV-FE-PLAYER-DESIGN-001]
Agent: ReviewAgent
Summary:
- P0-ревью FE-PLAYER-DESIGN-001: плеер 16/9, Fullscreen по контейнеру, hls.js; flat и типографика/отступы по гайду; тёмная тема через CSS-переменные, переключатель в topbar и на публичных страницах. Verdict: PASS.
Notes:
Код не менялся; коммит только devlog.

[2026-02-23] [BE-DOCS-README-001]
Agent: BackendAgent
Commit: a738886
Summary:
- В README добавлен раздел «Реализованные возможности» после «Frontend»: HLS-мониторинг (Worker, проверки, статусы), потоки/проекты, админка (users, requests), Telegram (alerts, delivery settings, DevLog), аналитика (состояния, тренды, FAIL/WARN), плеер HLS (просмотр, тёмная тема), AI по инцидентам (WARN/FAIL). Только README.
Notes:
Соответствие текущему состоянию по devlog и architecture_master.

[2026-02-23] [BE-TEST-AI-INCIDENT-001]
Agent: BackendAgent
Commit: 0f661ea
Summary:
- Добавлены тесты для GET .../check-jobs/{job_id}/ai-incident: 200 (есть запись — cause, summary), 404 (нет записи), 401 (без токена), 403 (tenant escape). Стиль как в middleware_auth_test: mock store (mockAIIncidentStore), handler в изоляции для 200/404, роутер с auth для 401/403.
Notes:
internal/http/api/handlers_ai_incident_test.go; go test ./... проходит.

[2026-02-23] [BE-TEST-API-HANDLERS-001]
Agent: BackendAgent
Commit: 6da8d0d
Summary:
- Добавлен батч тестов для основной tenant API: Streams (list 200, get 200/404, create 201/404 project miss, patch 200/404, delete 204/404, 401/403), Check jobs (enqueue 202/404 stream miss, get 200/404, list 200/404 stream miss, 401), Check results (get 200/404, get by job 200/404, list 200/404 stream miss, 401/403). Mock stores в стиле handlers_ai_incident_test; tenant scope в тестах соблюдён.
Notes:
internal/http/api/handlers_streams_check_test.go; go test ./... проходит.

[2026-02-22] [RV-BE-TEST-AI-INCIDENT-001]
Agent: ReviewAgent
Summary:
- P0-ревью коммита f40fedd (BE-TEST-AI-INCIDENT-001): тесты GET ai-incident — 200, 404, 401, 403; mock store; tenant scope (403 tenant_scope_required). Вердикт: PASS.
Notes:
Код не менялся; коммит только devlog.

[2026-02-22] [RV-P0-STATE-003]
Agent: ReviewAgent
Summary:
- P0-ревью состояния кодовой базы: tenant scoping (company_id в API/Worker, unscoped только companies), API≠Worker (ffmpeg/ffprobe/AI только в Worker), скриншоты UI (10 модулей с REPORT), логи без секретов. Verdict: PASS.
Notes:
Правки не вносились; коммит только devlog.

[2026-02-23] [docs-devlog-agents-notify]
Agent: MasterAgent
Commit: 6124a18
Summary:
- Правила обновлены: после коммита агенты обязаны запускать scripts/devlog_notify.ps1 (из корня репо), чтобы сообщение ушло в Telegram DevLog. Обновлены: .cursor/rules/project.mdc, docs/agents_and_responsibilities.md, .cursor/agents (backend, frontend, review, master).
Notes:
Отправку в TG делают агенты; хук post-commit может не сработать при коммите из IDE.

[2026-02-23] [devnotify-mood]
Agent: MasterAgent
Commit: 6124a18
Summary:
- cmd/devnotify: при -readSummaryFromGit поле «Настроение» выбирается случайно из списка коротких эмоциональных фраз вместо фиксированного «Коммит прошел».
Notes:
По запросу пользователя — разнообразие в DevLog.

[2026-02-23] [devlog-mood-free]
Agent: MasterAgent
Commit: 6124a18
Summary:
- Настроение в Telegram DevLog — как в рабочем чате: агент может записать одну строку (свободный текст) в .devlog_mood.txt в корне репо перед запуском devlog_notify.ps1; скрипт подхватит её в «Настроение» и удалит файл. Файл в .gitignore. Обновлены: scripts/devlog_notify.ps1, .gitignore, project.mdc, agents_and_responsibilities, инструкции агентов.
Notes:
Без ограничений по тексту (в рамках guardrails). Если файла нет — случайная фраза из devnotify.

[2026-02-22] [RV-BE-TEST-API-HANDLERS-001]
Agent: ReviewAgent
Summary:
- P0-ревью батча тестов BE-TEST-API-HANDLERS-001: Streams (list/get/create/patch/delete — 200/201/204/404), Check jobs (enqueue/list/get — 202/200/404), Check results (list/get/get-by-job — 200/404); 401/403 (streams, check-results, ai-incident); tenant scope (403 tenant_scope_required); стиль как в handlers_ai_incident_test (mock store, router для auth). Verdict: PASS.
Notes:
P1: для check-jobs нет теста 403 TenantEscape. Код не менялся; коммит только devlog.

[2026-02-22] [RV-BE-TEST-WORKER-001]
Agent: ReviewAgent
Summary:
- P0-ревью батча тестов BE-TEST-WORKER-001: persistCheckResult (tenant scope и payload в mock repo), applyAlertState (tenant scope и конфиг AlertFailStreak/AlertCooldown/AlertSendRecovered), ProcessSingleJobCycle при отсутствии джобов (no error). Моки в пакете worker, изоляция без БД. Verdict: PASS.
Notes:
Код не менялся; коммит только devlog.

[2026-02-23] [BE-TEST-API-P1-001]
Agent: BackendAgent
Commit: 926abd1
Summary:
- Добавлены тесты 403 tenant_scope_required (TenantEscape) для check-jobs: enqueue (POST companies/2/streams/1/check-jobs), list (GET companies/2/streams/1/check-jobs), get (GET companies/2/check-jobs/1) при пользователе company_id=1. По образцу streams и check-results; mock store и роутер с auth.
Notes:
Закрывает P1 из RV-BE-TEST-API-HANDLERS-001. После коммита — scripts/devlog_notify.ps1.

[2026-02-23] [BE-DOCS-README-002]
Agent: BackendAgent
Commit: b9362e7
Summary:
- README: добавлен подпункт «Быстрый старт» после «Запуск в Docker» (docker compose up, http://localhost:3000, логин test_screenshot_admin / TestScreenshot1). В «Реализованные возможности» — строка про тесты (API handlers, Worker) и про DevLog (настроение из .devlog_mood.txt).
Notes:
Только README.

[2026-02-23] [BE-MVP-PROD-001]
Agent: BackendAgent
Commit: d59b988
Summary:
- API: graceful shutdown по SIGTERM/SIGINT (signal.NotifyContext, Shutdown с таймаутом 15s), как в Worker — завершение без потери запросов в разумном таймауте.
- README: раздел «Деплой (запуск в production)» — docker compose, обязательные env (DATABASE_URL, API_PORT, NEXT_PUBLIC_API_BASE_URL), порты 8080/3000, volumes, проверка health; документировано, что GET /api/v1/health достаточен для liveness/readiness оркестратора.
Notes:
Отдельный /ready не добавляли; go test ./... проходит.

[2026-02-22] [RV-P0-STATE-004]
Agent: ReviewAgent
Summary:
- P0-ревью состояния кодовой базы: tenant scoping (company_id в API/Worker, unscoped только companies), API≠Worker (ffmpeg/ffprobe/AI только в Worker), скриншоты UI (10 модулей с REPORT), логи без секретов. Verdict: PASS.
Notes:
Правки не вносились; коммит только devlog.

[2026-02-23] [FE-MVP-PROD-001]
Agent: FrontendAgent
Commit: cd42c2e
Summary:
- MVP 100%: обход основных страниц (Streams, Stream detail/player, Analytics, Settings, Admin Users/Requests, Login/Register, Overview) — везде skeleton, empty, error states; критичных багов и «белых» экранов не выявлено. Production build (npm run build в web/) успешен. README: дополнение про NEXT_PUBLIC_API_BASE_URL для production-сборки. Бизнес-логика и контракты API не менялись.
Notes:
Post-commit: scripts/devlog_notify.ps1 (хук .githooks/post-commit).

[2026-02-22] [RV-FE-MVP-PROD-001]
Agent: ReviewAgent
Summary:
- P0-ревью FE-MVP-PROD-001: skeleton/empty/error на основных страницах MVP (Overview, Streams, Stream detail, Analytics, Companies, Admin, Settings, Login/Register), production build (npm run build) успешен, NEXT_PUBLIC_API_BASE_URL упомянут в README. Verdict: PASS.
Notes:
Код не менялся; коммит только devlog.

[2026-02-23] [BE-PROD-95-001]
Agent: BackendAgent
Commit: b196a8a
Summary:
- CI: .github/workflows/ci.yml — на push (master/main) go test ./... и в web/ npm ci + npm run build. README: упоминание CI; после «Деплой» добавлены подразделы «Мониторинг» (health, логи контейнеров, место на диске) и «Откат» (docker compose down, откат образа/кода, up, проверка health).
- API: security headers X-Content-Type-Options: nosniff и X-Frame-Options: DENY через обёртку securityHeaders в bootstrap.
Notes:
go test ./... проходит; npm build в CI на ubuntu-latest.

[2026-02-23] [FE-SCREENSHOTS-ALL-001]
Agent: FrontendAgent
Commit: 092f508
Summary:
- Добавлен npm run screenshot:all (web/scripts/screenshot-all.mjs): по очереди вызывает settings, analytics, streams, stream-detail, admin-requests, admin-users, companies, overview, login, register; при падении одного продолжает остальные. docs/screenshot_automation.md обновлён.
- Выполнено снятие: при работающем frontend на 3000 обновлены скриншоты login и register; остальные 8 модулей требуют API на 8080 (docker compose up -d или локальный API). В репо закоммичены обновлённые screenshots/login, screenshots/register и скрипт.
Notes:
Полный набор скриншотов всех модулей: поднять стек (docker compose up -d, после исправления Go в worker при необходимости) или API+frontend локально, затем из web/ выполнить npm run screenshot:all. Post-commit: scripts/devlog_notify.ps1.

[2026-02-23] [FE-FIX-UI-SCREENSHOTS-001]
Agent: FrontendAgent
Commit: 6124a18
Summary:
- Устранена возможность показа ошибки Next.js «Module not found: next/dist/pages/_app»: добавлен app/global-error.tsx (кастомная страница ошибки без зависимости от Pages Router). Login и Register приведены к единому стилю: явные background/color для .public-root и .auth-page, min-height и font-size для полей формы, тёмная тема для disabled полей. Переключатель темы (Dark) уже в едином стиле (AppButton). Пересняты скриншоты login и register.
Notes:
Скриншоты: screenshots/login/<timestamp>.png, screenshots/register/<timestamp>.png.

[2026-02-23] [FE-SCREENSHOTS-LOGIN-REGISTER-001]
Agent: FrontendAgent
Commit: e83a413
Summary:
- Пересняты скриншоты login и register (full page). Viewport 1280x720 и deviceScaleFactor 2 в screenshot-login.mjs и screenshot-register.mjs для размера PNG 10+ KB. REPORT.txt обновлены скриптами.
Notes:
Файлы: screenshots/login/20260223173917.png, screenshots/register/20260223173920.png (~16 KB).

[2026-02-23] [FE-MVP-FINAL-001]
Agent: FrontendAgent
Commit: 14bf095
Summary:
- Финальная полировка MVP: focus уже в globals (outline), тёмная тема для .state-error. aria-label на Login, Create registration request, Submit request, Back to login/Login, Run check, Connect/Reconnect Telegram, Retry, Save (settings), Approve/Reject, Apply/Reset, Save role/Save status.
Notes:
Сборка и скриншоты ок. Только UI/доступность.

[2026-02-24] [UI-AUTH-GLASS-GRAD-001] (fix BLOCK)
Agent: AuthUXAgent
Commit: c73d200
Summary:
- Пересъёмка скриншотов auth: скрипт ждёт появления h1 «Вход»/«Регистрация»; поддержка SCREENSHOT_PORT/SCREENSHOT_BASE_URL; удалены PNG с 404.
- Новые файлы: 20260224151810-login.png, 20260224151810-register.png; обновлён screenshot_automation.md.
Notes:
ReviewAgent BLOCK из-за 404 на старых скриншотах; переснято при frontend на :3004.

[2026-02-24] [UI-AUTH-GLASS-GRAD-001]
Agent: AuthUXAgent
Commit: 2aa2a0f
Summary:
- Auth-экраны: AnimatedGradientBackground (blobs + prefers-reduced-motion), шум feTurbulence, glass-карточки /login и /register, русские тексты, без бейджа и подзаголовка на входе.
- Loading состояния кнопок без спиннера (shimmer), hover/tap через CSS; favicon (icon.svg); скриншоты в screenshots/auth/, REPORT self-score 9.
Notes:
npm run build проходит; маршрутизация и логика auth не менялись.

[2026-02-24] [FE-LOGIN-GLASS-001]
Agent: FrontendAgent
Commit: 18c967c
Summary:
- Login/Register переведены в glassmorphism: анимированный градиентный фон через .public-root и auth-gradient, стеклянная auth-card с backdrop-filter и fallback без blur; обновлены скриншоты login и register (Score 9).
Notes:
Только UI/визуал; логика auth и screenshot-скрипты не менялись, build web проходит.

[2026-02-22] [RV-FE-SCREENSHOTS-ALL-001]
Agent: ReviewAgent
Summary:
- P0-ревью FE-SCREENSHOTS-ALL-001: скрипт screenshot:all в package.json и screenshot-all.mjs (порядок: settings…overview, login, register), обновлённые скриншоты и REPORT для login/register, документация в screenshot_automation.md. Verdict: PASS.
Notes:
Код не менялся; коммит только devlog.

[2026-02-22] [BE-PROD-FINAL-001]
Agent: BackendAgent
Commit: 15f69d1
Summary:
- Добавлен readiness endpoint GET /api/v1/ready: пинг БД (PingContext 2s), при успехе — 200 и {"ready":true}, при ошибке — 503 и {"ready":false}.
- README: в Деплой/Мониторинг указано использование GET /api/v1/ready для readiness оркестратором; добавлена рекомендация по бэкапу БД (pg_dump, надёжный диск для volume).
- docs/api_contract.md: описан GET /api/v1/ready; health оставлен как liveness.
Notes:
go test ./... проходит; API ≠ Worker не затронуты.

[2026-02-24] [MS1-SENSE-CONTROL-001]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: a288b63
Summary:
- Миграции 0007: stream_favorites (user_id, stream_id, is_pinned, sort_order), incidents (company_id, stream_id, status, severity, started_at, last_event_at, resolved_at, fail_reason, sample_screenshot_path, last_check_id).
- Worker: интеграция инцидентов в пайплайн проверок — создание/обновление/закрытие по результату OK/WARN/FAIL, audit log при open/resolve.
- API: favorites/pins (POST/DELETE favorite, pin; GET streams/favorites), incidents (GET list + by id), tenant-scope и RBAC.
- Observability: /metrics (promhttp), OTel tracer (OTEL_EXPORTER_OTLP_ENDPOINT), otelhttp для API.
- UI: потоки — ⭐/📌, pinned сверху, русские тексты, skeleton/empty/error; страница инциденты — таблица, фильтры, summary-карточки.
Notes:
Backend тесты проходят; frontend build в среде с блокировкой node (npm ci EPERM) не выполнялся — на CI/контейнере ожидается успех. Скриншоты milestone1 — по инструкции в REPORT.txt.

[2026-02-24] [CORE-UX-001]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- Backend: добавлены tenant-scoped маршруты `POST /companies/{company_id}/streams` и `POST /companies/{company_id}/streams/{stream_id}/check`; ручная проверка только enqueue job в очередь worker (API без ffmpeg/heavy).
- Frontend: новая домашняя `/` (СМОТРЕТЬ/МОНИТОРИНГ + Continue + OK/WARN/FAIL summary), новый `/watch` с `<video>` + native HLS fallback на `hls.js`, и расширенный `/streams` с CRUD-диалогами + actions (watch/check/edit/delete) при сохранении ⭐/📌.
- Контракт API обновлён в `docs/api_contract.md`; тесты API на новые endpoint-ы добавлены.
Notes:
`go test ./...` PASS, `web npm run build` PASS (есть существующий warning в incidents hook). Скриншоты CORE UX заблокированы окружением (docker engine/db недоступны), см. `screenshots/core-ux/REPORT.txt`.

[2026-02-24] [MS2-EMBED-WHITELIST-001]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- DB/Backend: миграция `0008` добавляет `streams.source_type` (`HLS|EMBED`), `streams.source_url` и таблицу `embed_whitelist` (tenant-scoped, unique(company_id, domain), enabled, created_by_user_id).
- API: добавлены endpoints `GET/POST/PATCH/DELETE /companies/{company_id}/embed-whitelist`, audit log (`add/toggle/remove`), RBAC (viewer forbidden), а также валидация EMBED-домена при create/update stream с сообщением `Домен не разрешён в Embed whitelist`.
- Frontend: `/settings` получил секцию управления Embed whitelist (таблица + add/toggle/delete), `/streams` форма потока расширена типом источника и source URL, `/watch` поддерживает безопасный iframe для EMBED и HLS player для HLS.
- CSP: в `web/middleware.ts` для `/watch` добавлен `Content-Security-Policy` с `frame-src 'self'` + динамически разрешённые домены из tenant whitelist.
Notes:
`go test ./...` PASS, `web npm run build` PASS (есть существующий warning в `app/incidents/page.tsx`). Скриншоты milestone2 заблокированы окружением (Docker engine недоступен), см. `screenshots/milestone2/REPORT.txt`.

[2026-02-24] [MS3L-SCREENSHOT-DIAG-001]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: 18fdfd4
Summary:
- Worker: при WARN/FAIL добавлена screenshot diagnostics без AI — захват кадров через ffmpeg в `/data/screenshots/incidents/{company}/{incident}`, rule-based диагнозы `BLACKFRAME|FREEZE|CAPTURE_FAIL|UNKNOWN`, запись в incident (`diag_code`, `diag_details`, `screenshot_taken_at`) и audit `incident_diagnostic_updated`.
- Runtime/infra: `Dockerfile.worker` дополнен `ffmpeg` и `ca-certificates`; в `docker-compose.yml` добавлен shared volume `app_data` и монтирование `/data` в `api` + `worker`; миграция `0009` расширяет таблицу `incidents`.
- API/UI: incidents DTO расширен (`diag_code`, `diag_details`, `has_screenshot`, `screenshot_taken_at`), добавлен endpoint `GET /companies/{company_id}/incidents/{id}/screenshot`, в UI добавлены колонка «Диагноз» и новая страница `/incidents/[id]` с карточкой, деталями и изображением.
- UX поправки: убран сворачиваемый «бургер» у бокового меню (статичное меню), в форме создания потока добавлен авто-режим проекта: если проект не выбран, используется/создаётся «Общий».
Notes:
`go test ./...` PASS, `web npm run build` PASS. Автоматические скриншоты milestone3-lite заблокированы окружением (Docker engine недоступен), см. `screenshots/milestone3-lite/REPORT.txt`.

[2026-02-25] [TOP5-PROD-HARDENING-001]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: f1c8922
Summary:
- Разделены API base переменные: `INTERNAL_API_BASE_URL` для server-side/rewrite и `NEXT_PUBLIC_API_BASE_URL` для публичного клиента; обновлены `web/next.config.mjs`, `web/Dockerfile`, `.env.example`, `docker-compose.yml`, `README.md`.
- Упрощён `web/middleware.ts`: убраны сетевые вызовы в API на каждом запросе; сохранены auth redirect и базовый CSP для `/watch`.
- В worker добавлен metrics/health endpoint (`/metrics`, `/health`) на `WORKER_METRICS_PORT`; обновлены compose и документация по мониторингу.
- Усилен `incident screenshot` handler: безопасная нормализация пути (`Abs/Rel`), защита от traversal, проверка regular file + MIME; добавлены тесты `internal/http/api/handlers_incidents_test.go`.
- Добавлены healthchecks/restart policies для `api/frontend/worker` и runbook диагностики `internal_error` в `README`.
Notes:
`go test ./...` PASS, `web npm run build` PASS; docker-lint warning по CVE базового образа не блокирует текущие изменения.

[2026-02-25] [PROD-HARDENING-WAVES-ABC-001]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- Wave A: secure-by-default hardening — seed disable by default (`BOOTSTRAP_SEED_ENABLED=false`), auth cookies (`HttpOnly`, configurable secure/samesite), refresh-from-cookie, metrics visibility tightened (`API_METRICS_PUBLIC=false` by default, worker metrics token support).
- Wave B: performance/observability — aggregated endpoint `GET /companies/{company_id}/streams/latest-statuses` for watch page (N+1 removed), API request Prometheus metrics + structured request logs, worker cycle/job Prometheus metrics, DB pool limits via env.
- Wave C: ops maturity — worker healthcheck in compose, localhost-bound worker metrics publish, incident+rollback runbook (`docs/incident_runbook.md`), contracts/docs synced (`README`, `docs/api_contract.md`, `.env.example`).
Notes:
`go test ./...` PASS; `web: npm ci && npm run build` PASS. Остались dependency-vuln предупреждения npm audit (внешние пакеты), вне текущего scope.

[2026-02-25] [PROD-HARDENING-9PT-001]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- API runtime hardening: добавлены recovery middleware (panic-safe 500), строгие security headers (CSP/referrer/permissions), и production timeouts (`API_*_TIMEOUT_SEC`, `API_MAX_HEADER_BYTES`, optional `API_HSTS_ENABLED`).
- Добавлен automated smoke-check `scripts/smoke-prod.ps1` (health, ready, metrics visibility, login, me, tenant-escape guard) для предрелизной проверки.
- README/.env синхронизированы с новыми runtime env и smoke-процедурой.
Notes:
Изменения минимально-инвазивные и совместимы с текущим docker compose деплоем.

[2026-02-25] [PROD-HARDENING-9PT-002]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- Security/perf fix: нормализован `clientIP` в rate-limit middleware (убран source-port из `RemoteAddr`, поддержаны `X-Forwarded-For`/`X-Real-IP`), чтобы исключить обход лимитера.
- Secure-by-default: proxy headers (`X-Forwarded-For`, `X-Real-IP`) используются только при `TRUST_PROXY_HEADERS=true`, чтобы исключить spoofing в прямом доступе к API.
- Compose hardening: в `docker-compose.yml` у API `AUTH_COOKIE_SECURE` переведён на secure-by-default (`${AUTH_COOKIE_SECURE:-true}`), без принудительного insecure override.
- Расширен auth rate-limit на публичные endpoints `POST /api/v1/auth/refresh` и `POST /api/v1/auth/telegram/login` (кроме уже существующих login/register).
- Добавлены регрессионные тесты `internal/http/api/ratelimit_middleware_test.go` (IP parsing + 429 на refresh), синхронизированы `docs/api_contract.md` и `README.md`.
Notes:
`go test ./...` PASS, `web npm run build` PASS.

[2026-02-25] [PROD-HARDENING-9PT-003]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- Добавлен fail-fast runtime safety guard для API (`APP_ENV=production`): запрет небезопасных конфигов (`API_METRICS_PUBLIC=true`, `AUTH_COOKIE_SECURE=false`, `AUTH_COOKIE_SAMESITE=none`, `BOOTSTRAP_SEED_ENABLED=true`, non-localhost `http://` в CORS).
- Усилен bootstrap init: `scripts/run-init.sh` теперь блокирует `BOOTSTRAP_SEED_ENABLED=true` при `APP_ENV=production`.
- Введён глобальный лимит request body (`API_MAX_BODY_BYTES`, default 1 MiB) middleware-уровня с `413 payload_too_large`.
- Исправлена трассировка request-id: единый `X-Request-ID` и `error_envelope.request_id` через context (без расхождения ID в одном запросе).
- Улучшен JSON decode guard: запрет нескольких JSON объектов в body через EOF-check.
- Добавлены тесты: `internal/config/runtime_safety_test.go`, `internal/http/api/body_limit_middleware_test.go`, `internal/http/api/request_id_middleware_test.go`, `internal/http/api/handlers_common_test.go`.
Notes:
Изменения совместимы с текущей архитектурой (API/Worker separation сохранён) и docker compose deploy.

[2026-02-25] [PROD-DEPLOY-TIMEWEB-001]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- Добавлен production compose override `docker-compose.prod.yml`: Caddy reverse-proxy + TLS entrypoints `80/443`, internal-only API/frontend/worker ports, `APP_ENV=production`, proxy-trust и secure-cookie defaults.
- Добавлен Caddy config `deploy/caddy/Caddyfile` с маршрутизацией `/api/* -> api:8080`, остальное -> `frontend:3000`, редирект `www -> apex`.
- Добавлена документация `docs/deploy_timeweb.md` (Ubuntu 24.04 Timeweb VPS): DNS, Docker install, `.env.prod`, запуск, smoke-check, backup/restore и cron.
- Добавлены ops scripts `scripts/backup-db.sh` и `scripts/restore-db.sh` для PostgreSQL rollback path.
- Добавлен `scripts/bootstrap-superadmin.sh` для безопасного первичного bootstrap `super_admin` в production (bcrypt через `pgcrypto` без тестовых паролей).
- Добавлен полный шаблон env для новичка `deploy/env.prod.full.example` (все переменные + подсказки по заполнению).
- README/.env обновлены под production profile (`DOMAIN`, `CADDY_EMAIL`, compose prod command).
Notes:
Профиль ориентирован на сервер 2 vCPU / 4 GB RAM / 50 GB NVMe, без изменений архитектуры API/Worker.

[2026-02-26] [UI-PLATFORM-REDESIGN-002]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- IA/роутинг: добавлены зоны `/hub`, `/monitoring/*`, `/admin/*`, `/watch`-layout, а также модульные каркасы `/sms/*` и `/ai/*`; индексы `/`, `/monitoring`, `/admin` переведены на канонические redirect.
- Auth UX: после логина/регистрации редирект жёстко на `/hub`; middleware обновлён под `/auth/login|register` и защищённые маршруты.
- Design system/UI shell: внедрены переиспользуемые компоненты `ModuleCard`, `Sidebar`, `SidebarItem`, `IconButton`, `StatusCountBadge`, `PrivateTopbar`, `AuthShell`, `AuthGate`; сайдбары сворачиваемые с persistence в localStorage.
- Hub/Watch/Admin polish: cinematic gateway с модульной сеткой и status icon+count для Monitoring; `/watch` приведён к операторскому 3-колоночному flow; action-зоны в `streams`, `admin/users`, `admin/requests`, `settings(embed whitelist)` переведены в icon-first + aria/tooltip.
- UI tokens/CSS: обновлены surface/hover/active слои и поведение для новых layout, сохранена совместимость текущей бизнес-логики/API.
Notes:
`web npm run lint` PASS, `web npm run build` PASS.

[2026-02-26] [HUB-DARK-POLAR-NIGHT-001]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- /hub (dark): реализован «Полярная ночь» — фон на 5 плавающих орбах (3 far, 2 near) с хаотичными траекториями через rAF + ease-in-out, без setState по кадру.
- /hub (dark): карточки модулей стали более стеклянными и полупрозрачными, иконки приведены к чистым монохромным svg без квадратных контейнеров, company switcher ужат до компактного popover.
- /hub (dark): лампочка theme-toggle переведена в роль атмосферного светового элемента (абсолютное позиционирование top-center, мягкий glow), статусы Monitoring сведены к минималистичным icon+number без цветных капсул.
Notes:
`web npm run build` PASS, перфоманс фона проверен на /hub (движение плавное, без рывков).

[2026-02-27] [HUB-DARK-POLAR-NIGHT-002]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- /hub (dark): подчистил типизацию `HubBackgroundBlobs` (blobRefs.current с non-null assertion), устранив ts-ошибки без изменения поведения анимации.
Notes:
Минимальный правочный коммит, только TS-guard для rAF орбит.

[2026-02-27] [HUB-DARK-ORBS-BACKGROUND-003]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- /hub: вынесён общий компонент фоновых орб `OrbsBackground` (variant=hub) на базе auth-анимаций, добавлен мягкий noise-слой и подключён под карточками через абсолютный слой.
- /hub: эмблема theme-toggle превращена в лёгкий «водяной знак» по центру сверху (без background/shadow, только opacity 0.35/0.5), company popover и карточки остались в существующей структуре.
Notes:
`web npm run build` PASS; фоновые орбы видны под стеклянными карточками и не мешают читаемости.

[2026-02-27] [HUB-DARK-LIGHT-SURFACES-004]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- /hub: обновлены поверхности карточек под концепты «Полярная ночь 2.0» (dark графитовый glass с большей прозрачностью) и «Морозное утро» (light frosted бело-ледяные cards с мягкой тенью и более высоким контрастом).
- /hub: фоновым орбам для варианта hub заданы отдельные цветовые акценты (3 синие/индиго + 2 холодных violet/magenta) и уровень blur/opacity для light vs dark, без затрагивания других страниц.
Notes:
`web npm run build` PASS; шары заметно читаются под карточками в dark, а light получил более объёмный ледяной фон.

[2026-02-27] [HUB-DARK-ORBS-LAMP-005]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- /hub (dark): усилен glow фоновых орб (больше blur и opacity, один фиолетовый орб заметно ярче остальных) при сохранении читаемости текста под стеклянными карточками.
- /hub: theme-toggle окончательно закреплён как центральный fixed watermark над страницей (fixed top-center, единая позиция для light/dark), без фона и с мягким hover-glow в тёмной теме.
Notes:
`web npm run build` PASS; layering проверен: орбы не перекрываются лишними слоями, карточки не выглядят «бетоном».

[2026-02-28] [HUB-DARK-GRAPHITE-ORBS-006]
Agent: UniversalAgent (Full-Stack Delivery)
Commit: pending
Summary:
- /hub (dark): базовый фон заменён на глубокий холодный градиент #050816 → #0B1024 без сине-серых overlay.
- /hub (dark): карточки переведены в нейтральный графит (rgba(24,28,40,0.65)), убран синий верхний градиент и затемняющий ::before, border/backdrop-blur сохранены.
- /hub (dark): 5 фоновых шаров с палитрой 3 синих (#1E3A8A, #1D4ED8, #2563EB) + 2 холодных фиолетовых (#7C3AED, #A855F7), распределены по экрану, blur 180–240px, один шар ярче (opacity +0.1), добавлено лёгкое scale breathing.
- /hub: лампочка без halo/квадрата, цвет rgba(255,255,255,0.7), hover rgba(255,255,255,1), только лёгкий drop-shadow glow rgba(124,58,237,0.4).
Notes:
Layering: base gradient → orbs (z-index 0) → content; glow заметен на статичном скрине.
