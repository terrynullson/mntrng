# Architecture Overview

## Scope
This document reflects the current layered structure after the API + Worker split and defines allowed dependency flow.

## Runtime components
- Frontend (`web/*`) calls HTTP API.
- API (`cmd/api/main.go`) handles HTTP contracts and persistence through service/repo layers.
- Worker (`cmd/worker/main.go`) executes asynchronous checks, alert decisions, Telegram delivery, and retention cleanup.
- PostgreSQL is the source of truth for entities, jobs, results, alert state, and audit.
- Redis is used for queue-related integration in the overall architecture baseline.
- Local filesystem stores screenshots referenced by `check_results.screenshot_path`.

## Layered dependency rules
- API chain: `http -> service -> repo -> domain`.
- Worker chain: `app -> service -> repo/telegram -> domain`.
- `cmd/*` is wiring-only: env/config, DB init, DI construction, server/app start.
- Repository layer must not depend on service layer.

Disallowed dependencies:
- `repo -> service`
- `domain -> repo|service|http`
- `cmd/*` containing business logic or SQL

Current status:
- API and Worker repositories comply with the `repo -> service` rule.
- Final API transitional debt (`repo -> service` for error contracts) was closed in commit `18a93ef`.

## Current package map

### Entrypoints (wiring only)
- `cmd/api/main.go`
- `cmd/worker/main.go`

### Shared/core
- `internal/config/env.go`
- `internal/domain/api_models.go`
- `internal/domain/worker_models.go`
- `internal/domain/worker_alert_transition.go`

### API side
- HTTP adapters/router:
  - `internal/http/api/router.go`
  - `internal/http/api/bootstrap.go`
  - `internal/http/api/server.go`
  - `internal/http/api/handlers_*.go`
- Use-cases:
  - `internal/service/api/company_service.go`
  - `internal/service/api/project_service.go`
  - `internal/service/api/stream_service.go`
  - `internal/service/api/check_job_service.go`
  - `internal/service/api/check_result_service.go`
- Persistence:
  - `internal/repo/postgres/api_company_repo.go`
  - `internal/repo/postgres/api_project_repo.go`
  - `internal/repo/postgres/api_stream_repo.go`
  - `internal/repo/postgres/api_check_job_repo.go`
  - `internal/repo/postgres/api_check_result_repo.go`

### Worker side
- App loop/orchestration:
  - `internal/service/worker/app.go`
  - `internal/service/worker/job_flow.go`
  - `internal/service/worker/worker_cycle.go`
- Worker use-case helpers:
  - `internal/service/worker/worker_alert_state.go`
  - `internal/service/worker/worker_telegram.go`
  - `internal/service/worker/worker_retention_finalize.go`
  - `internal/service/worker/worker_repositories.go`
  - `internal/service/worker/worker_types.go`
- Atomic checks:
  - `internal/service/worker/checks/playlist.go`
  - `internal/service/worker/checks/freshness.go`
  - `internal/service/worker/checks/segments.go`
  - `internal/service/worker/checks/declared_bitrate.go`
  - `internal/service/worker/checks/effective_bitrate.go`
  - `internal/service/worker/checks/freeze.go`
  - `internal/service/worker/checks/blackframe.go`
  - `internal/service/worker/checks/aggregate.go`
- Worker persistence repos:
  - `internal/repo/postgres/worker_repo.go`
  - `internal/repo/postgres/worker_job_repo.go`
  - `internal/repo/postgres/worker_stream_repo.go`
  - `internal/repo/postgres/worker_check_result_repo.go`
  - `internal/repo/postgres/worker_alert_state_repo.go`
  - `internal/repo/postgres/worker_telegram_repo.go`
  - `internal/repo/postgres/worker_retention_repo.go`
- Telegram transport:
  - `internal/telegram/client.go`

## Behavioral boundaries
- API does not run heavy media checks (`ffmpeg/ffprobe`), only CRUD/query/enqueue contracts.
- Worker performs check lifecycle (`claim -> process -> persist -> alert/telegram -> finalize`) and retention cleanup.
- Tenant scoping (`company_id`) is mandatory in API and Worker data access paths.
