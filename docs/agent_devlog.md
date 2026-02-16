# Agent DevLog Protocol

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
Commit: pending_single_commit
Summary:
- ������������� approve/reject �������� � `/admin/requests` ������ ��� super_admin.
- ���������� role-management controls � `/admin/users` ��� super_admin; ��������� ���� ��������� � read-only.
- � `/streams` ��������� Run check � RBAC-�������, � `/settings` ������������ Telegram link/reconnect flow.
Notes:
Protected shell � auth guard �� ���������; ������ web ������, ��� ������ ��� ��������� backend/runtime API.
