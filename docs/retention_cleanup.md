# Retention cleanup (Worker)

Очистка устаревших результатов проверок и файлов скриншотов. Выполняется **только в Worker**; API удаление не выполняет.

Источники: `PROMPTS/architecture_master.md` (раздел F), ADR-0005 в `docs/decisions.md`.

---

## 1. Что чистится

- **Таблица `check_results`:** строки с `created_at` старше TTL.
- **Файлы в storage:** для каждой удаляемой строки, у которой заполнен `screenshot_path`, вызывается удаление файла по этому пути. Путь хранится в БД; канонический формат по архитектуре: `storage/{company_id}/{stream_id}/{check_id}.jpg`.

Удаление файла выполняется до удаления строки в БД. Если файл уже отсутствует (например, удалён вручную), ошибка логируется, но cleanup по остальным кандидатам продолжается.

---

## 2. Tenant scope и батчи

- **Tenant scope:** cleanup выполняется по компаниям. Список компаний берётся из `companies`; для каждой `company_id` отдельно вызывается очистка только по строкам этой компании.
- **Запросы в БД:** `ListRetentionCandidates` и `DeleteStaleCheckResult` всегда фильтруют по `company_id`. Переход данных между тенантами невозможен.
- **Батчи:** за один проход по компании обрабатывается не более `RETENTION_CLEANUP_BATCH_SIZE` кандидатов. Цикл повторяется, пока есть строки старше cutoff; так нагрузка на БД и FS ограничена.

---

## 3. Идемпотентность

- Повторный запуск cleanup для тех же данных: строки с `created_at < cutoff` удаляются один раз; при повторном вызове их уже нет. Удаление в БД выполняется по условию `company_id = $1 AND id = $2 AND created_at < $3`, поэтому повторный DELETE по той же строке не изменяет данные.
- Файл: повторный вызов удаления по уже удалённому файлу возвращает «не существовал» и не считается ошибкой.

---

## 4. Конфигурация (ENV)

| Переменная | Описание | По умолчанию |
|------------|----------|--------------|
| `RETENTION_TTL_DAYS` | TTL в днях: записи и файлы старше этого срока удаляются | `30` |
| `RETENTION_CLEANUP_INTERVAL_MIN` | Интервал между запусками cleanup (минуты) | `60` |
| `RETENTION_CLEANUP_BATCH_SIZE` | Максимум кандидатов за один батч по компании | `100` |

Минимальные значения: TTL и интервал не менее 1; batch не менее 1. Задаются в `cmd/worker/main.go`, пример — `.env.example`.

---

## 5. Реализация

- **Worker:** `internal/service/worker/worker_retention_finalize.go` — `RunRetentionCleanup`, `cleanupCompanyRetention`, батч-цикл, удаление файла по `screenshot_path`.
- **Репозиторий:** `internal/repo/postgres/worker_retention_repo.go` — `ListCompanyIDsForRetention`, `ListRetentionCandidates` (по `company_id`, `created_at < cutoff`, LIMIT batch), `DeleteStaleCheckResult` (по `company_id`, id, cutoff).

---

## 6. Ссылки

- Хранилище скриншотов и путь: `docs/decisions.md` ADR-0003.
- 30-day retention: `docs/decisions.md` ADR-0005.
- Архитектура хранилища: `PROMPTS/architecture_master.md` § F.
