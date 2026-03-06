# Incident And Rollback Runbook

## 1. Scope

This runbook is for production incidents in HLS Monitoring stack:
- `api`
- `worker`
- `frontend`
- `postgres`
- `redis`

Primary goals:
- restore service quickly;
- preserve tenant data integrity;
- avoid unsafe ad-hoc fixes.

## 2. First 10 Minutes Checklist

1. Confirm blast radius:
   - which modules are down (`api`, `worker`, `frontend`);
   - single-tenant or multi-tenant impact.
2. Capture health:
   - `curl -fsS http://<api-host>:8080/api/v1/health`
   - `curl -fsS http://<api-host>:8080/api/v1/ready`
3. Capture logs (do not redact later manually):
   - `docker compose logs --since=15m api worker frontend postgres redis > incident-<ts>.log`
4. Freeze non-critical deploys until root cause is identified.

## 3. Fast Diagnostics

## 3.1 API degraded

Commands:

```bash
docker compose ps
docker compose logs --since=10m api
curl -sS http://localhost:8080/api/v1/ready
```

Common causes:
- DB unavailable / pool exhaustion;
- migration drift;
- invalid env (`DATABASE_URL`, auth cookie settings, CORS).

## 3.2 Worker stalled

Commands:

```bash
docker compose logs --since=10m worker
curl -sS http://127.0.0.1:${WORKER_METRICS_PORT:-9091}/health
```

If `WORKER_METRICS_TOKEN` is set:

```bash
curl -sS -H "Authorization: Bearer $WORKER_METRICS_TOKEN" \
  http://127.0.0.1:${WORKER_METRICS_PORT:-9091}/metrics | head
```

Check:
- `worker_cycle_total{result="error"}`;
- `worker_job_finalized_total{status="failed"}`.

## 3.3 Frontend auth loops / unexpected logout

Check:
- `AUTH_COOKIE_SECURE` and TLS termination;
- browser has `hm_refresh_token` cookie;
- `/api/v1/auth/refresh` returns `200` for same-origin call.

## 4. Rollback Procedure

Prerequisites:
- known stable commit/image tag;
- DB backups are available.

Steps:

1. Stop current stack:

```bash
docker compose down
```

2. Checkout stable version or set stable image tags.
3. Start stack:

```bash
docker compose up --build -d
```

4. Validate:
- `GET /api/v1/health` -> `200`
- `GET /api/v1/ready` -> `200`
- key UI flows: login, streams list, watch page.

5. If rollback includes DB schema downgrade:
- use `scripts/rollback_migrations.ps1` only with verified backup snapshot;
- document exact versions before and after.

## 5. Data Safety Rules

- Never run destructive SQL manually during incident without backup.
- Never run `git reset --hard` on production hosts.
- Keep tenant boundaries intact: no cross-tenant manual data copy.

## 6. Post-Incident

Within 24 hours:
- publish incident timeline (UTC);
- attach logs/metrics screenshots;
- add action items with owner and due date;
- update `docs/agent_devlog.md` and relevant contracts if behavior changed.
