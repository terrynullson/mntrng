# docs: navigation

Короткая карта документации проекта `mntrng`.

## Start here

1. `../README.md` — быстрый вход: запуск, env, smoke, ограничения.
2. `phase1_baseline_checklist.md` — минимальный воспроизводимый baseline.
3. `api_contract.md` + `schema.md` — API и модель данных.

## Product / architecture / ops (active)

- `api_contract.md` — активный REST-контракт.
- `schema.md` — активная схема БД и миграции.
- `decisions.md` — актуальные ADR/технические решения.
- `ai_incident_contract.md` — текущее состояние AI incident flow (stub-aware).
- `telegram_alerts_contract.md` — контракт алертов.
- `retention_cleanup.md` — retention/cleanup правила.
- `arch_overview.md` — актуальный обзор текущей runtime-архитектуры.
- `deploy_timeweb.md` — deployment guide.
- `incident_runbook.md` — эксплуатационный runbook.
- `phase1_baseline_checklist.md` — smoke/baseline checklist.

## Contributor / process layer (active)

Эти документы нужны для процесса разработки, но не являются source-of-truth для runtime/API:

- `quality.md` — система качества, green build, что проверяет CI.
- `agents_and_responsibilities.md`
- `agent_devlog.md`
- `frontend_ui_rules.md`
- `screenshot_automation.md`

## Historical docs

Исторические/аудитные материалы перенесены в `_archive/docs/` и не должны использоваться как текущий источник истины.
