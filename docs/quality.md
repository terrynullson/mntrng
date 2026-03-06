# Quality system (Phase 5)

Краткое описание системы качества и критериев зелёной сборки.

## Цель

Уменьшить зависимость от ручных проверок и случайных регрессий за счёт автоматизированных тестов и предсказуемого CI.

## Green build (что считается успешной проверкой)

Сборка считается **зелёной**, если все перечисленные проверки проходят:

| Проверка | Где выполняется | Описание |
|----------|------------------|----------|
| Backend build | CI job `backend` | `go build ./cmd/api ./cmd/worker` |
| Backend tests | CI job `backend` | `go test ./...` (без интеграционных, требующих БД, если `TEST_DATABASE_URL` не задан) |
| Frontend lint | CI job `frontend` | `npm run lint` в `web/` |
| Frontend build | CI job `frontend` | `npm run build` в `web/` |
| Frontend tests | CI job `frontend` | `npm run test` (e2e smoke, Playwright) в `web/` |
| Docker build (sanity) | CI job `docker-build` | Сборка образов `Dockerfile.api` и `Dockerfile.worker` без запуска |

Все job'ы должны завершиться успешно до merge в `main`/`master`.

## Что проверяется автоматически

- **Auth / tenant:** middleware и handlers — отсутствие токена (401), выход за tenant (403), RBAC (viewer read-only, company_admin scope, super_admin cross-company). Тесты: `middleware_auth_test.go`, `handlers_auth_test.go`, `handlers_check_job_test.go` (tenant escape), и др.
- **Critical API handlers:** streams, check-jobs, check-results, incidents, auth login/refresh. Юнит-тесты в `internal/http/api/handlers_*_test.go`.
- **Worker:** job flow, persist check result, alert state, retry/finalize, URL safety, агрегация статусов, пороги. Тесты в `internal/service/worker/*_test.go`.
- **Config / runtime safety:** env, devnotify, runtime safety — тесты в `internal/config/*_test.go`.
- **Frontend:** сборка Next.js и smoke e2e (например, загрузка страницы входа) через Playwright.

## Интеграционные тесты и smoke

- Тесты, требующие БД, используют `TEST_DATABASE_URL`. В CI без этой переменной они пропускаются (`t.Skip`).
- Локальный production-like smoke: `scripts/smoke-prod.ps1` (требует запущенный API и учётные данные). В CI не запускается в рамках Phase 5.

## Запуск проверок локально

То же, что и в CI:

- Backend: `go build ./cmd/api ./cmd/worker && go test ./...`
- Frontend: в `web/`: `npm ci`, `npm run lint`, `npm run build`, `npx playwright install chromium`, `npm run test:ci`
- Docker: `docker build -f Dockerfile.api -t mntrng-api:sanity .` и то же для `Dockerfile.worker`

## Что остаётся на Phase 6 (production hardening)

- Полноценный интеграционный CI с поднятой БД/Redis.
- Расширенный e2e против живого стека.
- Security scanning, dependency audit в pipeline.
- Performance/load тесты.
