# Layered Refactor Plan (API + Worker)

## Scope and constraints

- Goal: split current monolith files into layered structure `http/service/repo/domain` with `cmd/*` as wiring only.
- Strict rule: **no feature changes**. Only structural refactor (move/extract/rename, DI wiring, tests adaptation).
- Out of scope: behavior changes, new endpoints, schema changes, new external integrations.

## Completion note (finalization)

- Phase C Step 17 completed: `cb314ef` (`worker: refactor finalize cmd worker bootstrap wiring only`).
- Phase D Step 18 completed: `d439e2c` (`worker: refactor stabilize interfaces and remove dead glue`).
- Phase D Step 19 completed and reconciled:
  - `9b151c3` (`docs: update architecture references after layered split`)
  - `e771353` (`docs: align architecture overview with current transitional repo dependencies`)
  - `18a93ef` (`api: refactor remove repo dependency on service layer error contracts`) closed final API transitional debt `repo -> service`.

## 1) Monolith sources (current)

- `cmd/api/main.go`
  - HTTP routing + handlers
  - request validation + error envelope
  - SQL queries + transactions
  - domain shaping / status normalization
  - audit logging and helper utilities
- `cmd/worker/main.go`
  - loop/scheduler, retries, claim/finalize
  - checks orchestration (playlist/freshness/segments/bitrates/freeze/blackframe)
  - SQL/repository access
  - alert-state transitions and telegram delivery
  - retention cleanup
- `internal/config/env.go`
  - env parsing utilities (`GetString`, `GetInt`) used by both binaries

## 2) Target packages/files

### API target

- `cmd/api/main.go`
  - only bootstrap: config load, DB init, DI, router start.
- `internal/http/api/router.go`
  - route registration, path parsing, middleware wiring.
- `internal/http/api/handlers_*.go`
  - handlers by bounded area:
    - `handlers_company.go`
    - `handlers_project.go`
    - `handlers_stream.go`
    - `handlers_check_job.go`
    - `handlers_check_result.go`
    - `handlers_common.go` (error envelope, method not allowed, decode helpers)
- `internal/service/api/*.go`
  - use-cases:
    - company/project/stream CRUD
    - check jobs/results read APIs
    - audit write orchestration
- `internal/repo/postgres/api_*.go`
  - SQL and tx wrappers per entity.
- `internal/domain/*.go`
  - entities/DTO/status enums shared by handler-service-repo contracts.

### Worker target

- `cmd/worker/main.go`
  - only bootstrap: config load, DB init, worker app run.
- `internal/service/worker/app.go`
  - main loop orchestration, retry policy, cycle scheduling.
- `internal/service/worker/job_flow.go`
  - claim -> process -> persist -> alert -> finalize pipeline.
- `internal/service/worker/checks/*.go`
  - checker modules:
    - `playlist.go`
    - `freshness.go`
    - `segments.go`
    - `declared_bitrate.go`
    - `effective_bitrate.go`
    - `freeze.go`
    - `blackframe.go`
    - `aggregate.go`
- `internal/service/worker/alerts/*.go`
  - anti-spam transition logic and notification decision.
- `internal/repo/postgres/worker_*.go`
  - job claim/finalize, check_results persistence, alert_state updates, retention SQL.
- `internal/telegram/client.go`
  - sendMessage client + retries (only transport client and API call logic).
- `internal/domain/*.go`
  - worker models/status/reasons shared across layers.

## 3) Step-by-step commits (1 step = 1 commit)

### Phase A: baseline safety

1. **docs: add refactor map and invariants**
   - Add short mapping doc (this plan + package boundaries summary).
   - No code change.
   - Check: N/A.

2. **refactor(api): introduce internal/domain models for API contracts**
   - Extract structs/enums from `cmd/api/main.go` into `internal/domain/api_models.go`.
   - Keep imports/wrappers to avoid behavior drift.
   - Check: `go test ./...`.

3. **refactor(worker): introduce internal/domain models for worker**
   - Extract worker-specific models/status consts to `internal/domain/worker_models.go`.
   - Check: `go test ./...`.

### Phase B: API split

4. **refactor(api): extract common HTTP helpers**
   - Move `writeJSON`, `writeJSONError`, method-not-allowed, request-id/decode helpers to `internal/http/api/handlers_common.go`.
   - Check: `go test ./...`.

5. **refactor(api): extract router + path parsing**
   - Move route registration and path parsing into `internal/http/api/router.go`.
   - `cmd/api/main.go` keeps only wiring.
   - Check: `go test ./...`.

6. **refactor(api): extract company handlers into http layer**
   - Move company handlers to `internal/http/api/handlers_company.go`.
   - No SQL changes yet (still via temporary facade).
   - Check: `go test ./...`.

7. **refactor(api): extract project and stream handlers into http layer**
   - Move project/stream handlers to `handlers_project.go` and `handlers_stream.go`.
   - Check: `go test ./...`.

8. **refactor(api): introduce service layer for company/project/stream use-cases**
   - Move business validation/decision flow from handlers to `internal/service/api`.
   - Handlers become thin adapters.
   - Check: `go test ./...`.

9. **refactor(api): introduce postgres repo layer for CRUD/audit**
   - Move SQL and tx logic to `internal/repo/postgres/api_*.go`.
   - Service consumes repo interfaces.
   - Check: `go test ./...`.

10. **refactor(api): extract check-job/check-result handlers + service/repo**
    - Split remaining API logic into layered files.
    - Check: `go test ./...`.

11. **refactor(api): minimize cmd/api to bootstrap-only**
    - Ensure `cmd/api/main.go` only initializes config/db/router/server.
    - Check: `go test ./...`.

### Phase C: Worker split

12. **refactor(worker): extract app loop and retry policy**
    - Move loop/heartbeat/retry glue to `internal/service/worker/app.go`.
    - Check: `go test ./...`.

13. **refactor(worker): extract job lifecycle orchestration**
    - Move claim/process/persist/finalize flow to `job_flow.go`.
    - Check: `go test ./...`.

14. **refactor(worker): split checks into separate modules**
    - Move each checker and aggregator to `internal/service/worker/checks/*.go`.
    - Keep exact thresholds/decision rules.
    - Check: `go test ./...`.

15. **refactor(worker): split alert + telegram logic**
    - Move anti-spam transition to `alerts/transition.go`.
    - Move telegram HTTP client to `internal/telegram/client.go`.
    - Check: `go test ./...`.

16. **refactor(worker): extract repo layer for worker SQL and retention**
    - Move all SQL in worker to `internal/repo/postgres/worker_*.go`.
    - Check: `go test ./...`.

17. **[DONE] refactor(worker): minimize cmd/worker to bootstrap-only**
    - Keep only config wiring + app start in `cmd/worker/main.go`.
    - Check: `go test ./...`.

### Phase D: finish and hardening

18. **[DONE] refactor: stabilize interfaces and remove dead glue**
    - Remove transitional wrappers, keep clear dependency graph:
      - `http -> service -> repo`
      - `worker app -> service -> repo/telegram`
    - Check: `go test ./...`.

19. **[DONE] docs: update architecture references after split**
    - Update docs pointers to new package map.
    - Check: N/A.

## 4) Risks and verification order

## Main risks

- Hidden behavior drift while moving validation branches and error mapping.
- Transaction boundary drift when moving SQL from handlers to repos.
- Status normalization drift (`ok/warn/fail` vs `OK/WARN/FAIL`) in moved code.
- Retry/backoff/timeouts drift in worker during extraction.
- Import cycles between `domain/service/repo/http`.

## Verification order (mandatory per commit)

1. Run `go test ./...`.
2. For API-touching commits:
   - compile and smoke with existing flow:
   - `go test ./cmd/api -count=1`
3. For Worker-touching commits:
   - `go test ./cmd/worker -count=1`
4. For commits moving tx/repo:
   - run focused smoke with DB if available (no behavior assertions changed, only parity).
5. Before merge:
   - compare key API envelopes and worker decision outputs against pre-refactor snapshots.

## Refactor policy gates

- Any step that changes behavior is rejected and must be split into a separate feature task.
- If a step exceeds scope, stop and emit follow-up plan instead of bundling.
- Keep each commit reviewable and reversible.
