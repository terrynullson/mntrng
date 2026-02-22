# Agent DevLog Protocol

## После своей работы (обязательно для агентов)

Это касается **всех агентов**: MasterAgent, ReviewAgent, BackendAgent, FrontendAgent.

1. **Запись в этот файл** — добавить одну запись по формату ниже (DATE, MODULE, Agent, Commit, Summary, Notes; до 12 строк).
2. **Коммит** — закоммитить изменения (включая эту запись и код). После коммита сообщение в Telegram DevLog уходит **автоматически** (post-commit hook → `scripts/devlog_notify.ps1` → `cmd/devnotify` с `-readSummaryFromGit`). Ручной вызов devnotify не нужен; достаточно включённых хуков (`git config core.hooksPath .githooks`) и переменных в `.env` (DEV_LOG_TELEGRAM_*). Если изменений кода не было — достаточно закоммитить только запись в этом файле.

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
Commit: bf978c6 (api), d99ed92 + b83781b (ui + playwright)
Summary:
- API: GET/PATCH company telegram delivery settings (tenant-scoped, company_admin+).
- UI: секция «Telegram Alerts (Company)» на /settings (skeleton, empty/error, micro animation).
- Автоматизация скриншота: npm run screenshot:settings в web/ (playwright, тестовый логин).
Notes:
ReviewAgent RV-TDS-001 / RV-TDS-002 / RV-TDS-003: BLOCK по P0 (в репо только старый скриншот с логином). Для PASS: локально API + DATABASE_URL, затем cd web && npm run screenshot:settings, закоммитить новый .png из screenshots/telegram-delivery-settings/.
