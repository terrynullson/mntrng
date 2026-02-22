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

### Authentication and authorization

- Every non-public endpoint requires `Authorization: Bearer <access_token>`.
- Public endpoints:
  - `GET /api/v1/health`
  - `POST /api/v1/auth/register`
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/refresh`
  - `POST /api/v1/auth/telegram/login`
- RBAC roles:
  - `super_admin` — cross-company operations by policy.
  - `company_admin` — read/write only in own company scope.
  - `viewer` — read-only in own company scope.
- Tenant guard: for tenant routes, `company_id` from route must match authenticated tenant context (except allowed `super_admin` cross-company flows).

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
- `method_not_allowed` -> `405`

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

### `GET /api/v1/health`

- Purpose: liveness/readiness probe of API process.
- `200` response example:

```json
{
  "status": "ok",
  "service": "api",
  "time": "2026-02-13T10:00:00Z"
}
```

## 5.2 Auth and controlled registration

### `POST /api/v1/auth/register`

- Public controlled-registration endpoint.
- Body:

```json
{
  "company_id": 10,
  "email": "operator@example.com",
  "login": "operator",
  "password": "StrongPass123",
  "requested_role": "viewer"
}
```

- `202` -> registration request payload (`status=pending`).
- Errors: `400`, `404`, `409`, `500`.

### `POST /api/v1/auth/login`

- Public password login for active approved users.
- Body:

```json
{
  "login_or_email": "operator",
  "password": "StrongPass123"
}
```

- `200`:

```json
{
  "access_token": "string",
  "refresh_token": "string",
  "token_type": "Bearer",
  "expires_in": 900,
  "user": {
    "id": 1,
    "company_id": 10,
    "email": "operator@example.com",
    "login": "operator",
    "role": "viewer",
    "status": "active",
    "created_at": "2026-02-13T10:00:00Z",
    "updated_at": "2026-02-13T10:00:00Z"
  }
}
```

- Errors: `400`, `401`, `403`, `500`.
- Pending/rejected/disabled identities cannot login.

### `POST /api/v1/auth/refresh`

- Public token refresh by refresh token.
- Body:

```json
{
  "refresh_token": "string"
}
```

- `200` -> token response (same schema as login).
- Errors: `400`, `401`, `500`.

### `POST /api/v1/auth/logout`

- Protected endpoint.
- Body (optional):

```json
{
  "refresh_token": "string"
}
```

- `204` no body.
- Errors: `400`, `401`, `500`.

### `GET /api/v1/auth/me`

- Protected endpoint.
- `200` -> authenticated user profile.
- Errors: `401`, `500`.

### `POST /api/v1/auth/telegram/link`

- Protected endpoint.
- Request body: Telegram login payload fields + `hash`.
- Backend verifies Telegram signature and links `telegram_user_id` to current user.
- `204` no body.
- Errors: `400`, `401`, `403`, `409`, `500`.

### `POST /api/v1/auth/telegram/login`

- Public endpoint with Telegram login payload.
- Allowed only for approved+active linked users.
- `200` -> token response (same schema as password login).
- Errors: `400`, `401`, `403`, `500`.

### `GET /api/v1/admin/registration-requests`

- Protected, `super_admin` only.
- Returns pending registration requests list.
- `200` -> `{items, next_cursor}`.
- Errors: `401`, `403`, `500`.

### `POST /api/v1/admin/registration-requests/{request_id}/approve`

- Protected, `super_admin` only.
- Body:

```json
{
  "company_id": 10,
  "role": "viewer"
}
```

- Approves request, creates active user, writes audit log.
- `200` -> created user profile.
- Errors: `400`, `401`, `403`, `404`, `409`, `500`.

### `POST /api/v1/admin/registration-requests/{request_id}/reject`

- Protected, `super_admin` only.
- Body:

```json
{
  "reason": "string"
}
```

- Rejects request and writes audit log.
- `204` no body.
- Errors: `400`, `401`, `403`, `404`, `409`, `500`.

### `PATCH /api/v1/admin/users/{user_id}/role`

- Protected, `super_admin` only.
- Body:

```json
{
  "role": "company_admin",
  "company_id": 10
}
```

- Updates role/company scope (`company_admin` or `viewer`) and writes audit log.
- `200` -> updated user profile.
- Errors: `400`, `401`, `403`, `404`, `500`.

### `GET /api/v1/admin/users`

- Protected, `super_admin` only.
- Purpose: list users for Admin UI with optional filters.
- Query params (all optional):
  - `company_id` (`integer`, filter by tenant company)
  - `role` (`super_admin|company_admin|viewer`)
  - `status` (`active|disabled`)
  - `limit` (`integer`, positive, capped server-side for safe response size)
- `200`:

```json
{
  "items": [
    {
      "id": 14,
      "company_id": 10,
      "email": "viewer@example.com",
      "login": "viewer01",
      "role": "viewer",
      "status": "active",
      "created_at": "2026-02-16T10:00:00Z",
      "updated_at": "2026-02-16T10:00:00Z"
    }
  ],
  "next_cursor": null
}
```

- Errors: `400`, `401`, `403`, `500`.

### `PATCH /api/v1/admin/users/{user_id}/status`

- Protected, `super_admin` only.
- Body:

```json
{
  "status": "disabled"
}
```

- Allowed status values: `active`, `disabled`.
- Writes audit log entry (`entity_type=user`, `action=status_change`) with payload:
  - `user_id`
  - `old_status`
  - `new_status`
  - `actor_user_id`
- `200` -> updated user profile.
- Errors: `400`, `401`, `403`, `404`, `500`.

## 5.3 Companies (CRUD)

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

## 5.4 Projects (tenant-scoped CRUD)

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

## 5.5 Streams (tenant-scoped CRUD)

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

## 5.6 Check jobs (enqueue, status, history)

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

## 5.7 Check results (read-only)

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

## 5.8 Telegram delivery settings (tenant-scoped)

All endpoints in this section are tenant-scoped by route `company_id`. Access: `company_admin` or `super_admin` only (viewer forbidden).

### `GET /api/v1/companies/{company_id}/telegram-delivery-settings`

- Purpose: get company Telegram delivery settings.
- `200`:

```json
{
  "is_enabled": true,
  "chat_id": "-1001234567890",
  "send_recovered": true,
  "created_at": "2026-02-13T10:00:00Z",
  "updated_at": "2026-02-13T10:00:00Z"
}
```

- `404` when no row exists for the company.
- Errors: `401`, `403`, `404`, `500`.

### `PATCH /api/v1/companies/{company_id}/telegram-delivery-settings`

- Purpose: create or update (upsert) company Telegram delivery settings. Does not expose or modify `bot_token_ref`.
- Body (all fields optional; omitted fields are left unchanged on update, or use defaults on create):

```json
{
  "is_enabled": true,
  "chat_id": "-1001234567890",
  "send_recovered": true
}
```

- Validation: when `chat_id` is provided, it must be non-empty after trim.
- `200` -> same response model as GET (`is_enabled`, `chat_id`, `send_recovered`, `created_at`, `updated_at`).
- Errors: `400` (e.g. empty `chat_id`), `401`, `403`, `404` (company not found), `500`.

## 6. Consistency with schema and ADR

- Tenant contract aligns with ADR-0002 and ADR-0006: no tenant endpoint without `company_id` scope.
- Job lifecycle (`queued/running/done/failed`) aligns with `check_jobs.status` constraint from schema.
- Results are read-only at API contract level, aligned with immutable `check_results` rows in schema.
- Retention policy is Worker responsibility (ADR-0005); API contract does not expose retention execution endpoints.
