# Baseline API Contract

## 1. Scope and principles

- Base path: `/api/v1`
- Transport: JSON over HTTP.
- Baseline scope: contract only. No runtime implementation details.
- Architectural boundary: API does not run `ffmpeg/ffprobe`; heavy checks are executed by Worker.

### Tenant scoping (mandatory)

- Every tenant endpoint is scoped by `company_id` in route: `/companies/{company_id}/...`.
- `company_id` from route must be validated against auth context.
- Every DB read/write for tenant data must include tenant filter by `company_id`.
- Unscoped tenant endpoints are not allowed in baseline contract.

## 2. Error envelope

All non-2xx responses use one JSON envelope:

```json
{
  "code": "string_machine_code",
  "message": "human readable message",
  "details": {},
  "request_id": "req_123"
}
```

### Standard error codes

- `validation_error` -> `400`
- `unauthorized` -> `401`
- `forbidden` -> `403`
- `tenant_scope_required` -> `403`
- `not_found` -> `404`
- `conflict` -> `409`
- `rate_limited` -> `429`
- `internal_error` -> `500`

Unless stated otherwise, every endpoint in this contract can return the standard error envelope with the codes above.

## 3. Data models (API)

### Company

```json
{
  "id": 10,
  "name": "Acme Media",
  "created_at": "2026-02-13T10:00:00Z"
}
```

### Project

```json
{
  "id": 301,
  "company_id": 10,
  "name": "News Channel",
  "created_at": "2026-02-13T10:00:00Z",
  "updated_at": "2026-02-13T10:00:00Z"
}
```

### Stream

```json
{
  "id": 1201,
  "company_id": 10,
  "project_id": 301,
  "name": "Primary HLS",
  "url": "https://cdn.example.com/live/index.m3u8",
  "is_active": true,
  "created_at": "2026-02-13T10:00:00Z",
  "updated_at": "2026-02-13T10:00:00Z"
}
```

### Check job

```json
{
  "id": 9001,
  "company_id": 10,
  "stream_id": 1201,
  "planned_at": "2026-02-13T10:05:00Z",
  "status": "queued",
  "created_at": "2026-02-13T10:04:50Z",
  "started_at": null,
  "finished_at": null,
  "error_message": null
}
```

Allowed `check_jobs.status` values: `queued`, `running`, `done`, `failed`.

### Check result (read-only)

Atomic check statuses inside `checks` use uppercase values: `OK`, `WARN`, `FAIL`.

```json
{
  "id": 7001,
  "company_id": 10,
  "job_id": 9001,
  "stream_id": 1201,
  "status": "FAIL",
  "checks": {
    "playlist": "OK",
    "segments": "OK",
    "freshness": "FAIL",
    "declared_bitrate": "OK",
    "effective_bitrate": "WARN",
    "freeze": "OK",
    "blackframe": "OK",
    "blackframe_details": {
      "dark_frame_ratio": 0.12,
      "analyzed_frames": 320,
      "reason": "within_threshold",
      "source": "ffmpeg_blackframe"
    }
  },
  "screenshot_path": "storage/10/1201/9001.jpg",
  "created_at": "2026-02-13T10:05:12Z"
}
```

`checks.blackframe_details` schema:
- `dark_frame_ratio` (`number`)
- `analyzed_frames` (`integer`)
- `reason` (`string`)
- `source` (`string`, expected value: `ffmpeg_blackframe`)

Deterministic blackframe fallback contract:
- If blackframe analysis cannot be completed, `checks.blackframe` must be `WARN`.
- `checks.blackframe_details.reason` must be one of:
  - `playlist_unavailable`
  - `segments_not_available`
  - `no_downloaded_segments`
  - `blackframe_analysis_failed`
  - `blackframe_analyzer_not_available`

## 4. Status model and aggregation rule

### Stream quality status

- `OK`
- `WARN`
- `FAIL`

API contract uses uppercase status values. Persistence layer may store lowercase equivalents (`ok`, `warn`, `fail`) as an internal representation.

### Aggregation rule (from thresholds_and_rules)

- Any atomic check with `FAIL` makes overall status `FAIL`.
- If there is no `FAIL` and at least one `WARN`, overall status is `WARN`.
- Otherwise overall status is `OK`.

## 5. Endpoint contract

## 5.1 Health

### `GET /health`

- Purpose: liveness/readiness probe of API process.
- `200` response example:

```json
{
  "status": "ok",
  "service": "api",
  "time": "2026-02-13T10:00:00Z"
}
```

## 5.2 Companies (CRUD)

### `POST /companies`

- Purpose: create company (tenant root).
- Body:

```json
{
  "name": "Acme Media"
}
```

- `201` -> `Company`.
- Errors: `400`, `401`, `403`, `409`, `500`.

### `GET /companies`

- Purpose: list companies.
- `200`:

```json
{
  "items": [
    {
      "id": 10,
      "name": "Acme Media",
      "created_at": "2026-02-13T10:00:00Z"
    }
  ],
  "next_cursor": null
}
```

### `GET /companies/{company_id}`

- Purpose: get company by id.
- `200` -> `Company`.
- Errors: `401`, `403`, `404`, `500`.

### `PATCH /companies/{company_id}`

- Purpose: update company fields.
- Body:

```json
{
  "name": "Acme Media Group"
}
```

- `200` -> `Company`.
- Errors: `400`, `401`, `403`, `404`, `409`, `500`.

### `DELETE /companies/{company_id}`

- Purpose: delete company and cascade tenant data.
- `204` no body.
- Errors: `401`, `403`, `404`, `500`.

## 5.3 Projects (tenant-scoped CRUD)

All endpoints in this section are tenant-scoped by route `company_id`.

### `POST /companies/{company_id}/projects`

- Body:

```json
{
  "name": "News Channel"
}
```

- `201` -> `Project`.
- Errors: `400`, `401`, `403`, `404`, `409`, `500`.

### `GET /companies/{company_id}/projects`

- Query params: `limit`, `cursor`.
- `200`:

```json
{
  "items": [
    {
      "id": 301,
      "company_id": 10,
      "name": "News Channel",
      "created_at": "2026-02-13T10:00:00Z",
      "updated_at": "2026-02-13T10:00:00Z"
    }
  ],
  "next_cursor": null
}
```

### `GET /companies/{company_id}/projects/{project_id}`

- `200` -> `Project`.
- Errors: `401`, `403`, `404`, `500`.

### `PATCH /companies/{company_id}/projects/{project_id}`

- Body:

```json
{
  "name": "Updated Project Name"
}
```

- `200` -> `Project`.

### `DELETE /companies/{company_id}/projects/{project_id}`

- `204` no body.

## 5.4 Streams (tenant-scoped CRUD)

All endpoints in this section are tenant-scoped by route `company_id`.

### `POST /companies/{company_id}/projects/{project_id}/streams`

- Body:

```json
{
  "name": "Primary HLS",
  "url": "https://cdn.example.com/live/index.m3u8",
  "is_active": true
}
```

- `201` -> `Stream`.
- Errors: `400`, `401`, `403`, `404`, `409`, `500`.

### `GET /companies/{company_id}/streams`

- Query params: `project_id`, `is_active`, `limit`, `cursor`.
- `200` -> paginated list of `Stream`.

### `GET /companies/{company_id}/streams/{stream_id}`

- `200` -> `Stream`.
- Errors: `401`, `403`, `404`, `500`.

### `PATCH /companies/{company_id}/streams/{stream_id}`

- Body (partial):

```json
{
  "name": "Backup HLS",
  "url": "https://cdn.example.com/backup/index.m3u8",
  "is_active": false
}
```

- `200` -> `Stream`.

### `DELETE /companies/{company_id}/streams/{stream_id}`

- `204` no body.

## 5.5 Check jobs (enqueue, status, history)

All endpoints in this section are tenant-scoped by route `company_id`.

### `POST /companies/{company_id}/streams/{stream_id}/check-jobs`

- Purpose: enqueue check job for stream.
- Body:

```json
{
  "planned_at": "2026-02-13T10:05:00Z"
}
```

- Idempotency key at domain level: `(stream_id, planned_at)`.
- `202`:

```json
{
  "job": {
    "id": 9001,
    "company_id": 10,
    "stream_id": 1201,
    "planned_at": "2026-02-13T10:05:00Z",
    "status": "queued",
    "created_at": "2026-02-13T10:04:50Z",
    "started_at": null,
    "finished_at": null,
    "error_message": null
  }
}
```

- Errors: `400`, `401`, `403`, `404`, `409`, `500`.

### `GET /companies/{company_id}/check-jobs/{job_id}`

- Purpose: get job status.
- `200` -> `Check job`.
- Errors: `401`, `403`, `404`, `500`.

### `GET /companies/{company_id}/streams/{stream_id}/check-jobs`

- Purpose: job history for stream.
- Query params: `status`, `from`, `to`, `limit`, `cursor`.
- `200`:

```json
{
  "items": [
    {
      "id": 9001,
      "company_id": 10,
      "stream_id": 1201,
      "planned_at": "2026-02-13T10:05:00Z",
      "status": "done",
      "created_at": "2026-02-13T10:04:50Z",
      "started_at": "2026-02-13T10:05:01Z",
      "finished_at": "2026-02-13T10:05:12Z",
      "error_message": null
    }
  ],
  "next_cursor": null
}
```

## 5.6 Check results (read-only)

All endpoints in this section are tenant-scoped by route `company_id`.

### `GET /companies/{company_id}/check-results/{result_id}`

- Purpose: fetch single check result.
- `200` -> `Check result`.
- Errors: `401`, `403`, `404`, `500`.

### `GET /companies/{company_id}/streams/{stream_id}/check-results`

- Purpose: history of results for stream.
- Query params: `status`, `from`, `to`, `limit`, `cursor`.
- `200` -> paginated list of `Check result`.

### `GET /companies/{company_id}/check-jobs/{job_id}/result`

- Purpose: get result by job id.
- `200` -> `Check result`.
- Errors: `401`, `403`, `404`, `500`.

## 6. Consistency with schema and ADR

- Tenant contract aligns with ADR-0002 and ADR-0006: no tenant endpoint without `company_id` scope.
- Job lifecycle (`queued/running/done/failed`) aligns with `check_jobs.status` constraint from schema.
- Results are read-only at API contract level, aligned with immutable `check_results` rows in schema.
- Retention policy is Worker responsibility (ADR-0005); API contract does not expose retention execution endpoints.
