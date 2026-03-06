# Telegram Alerts Contract (Worker)

Контракт отправки алертов мониторинга HLS в Telegram. Отправкой занимается **только Worker**; API алерты не отправляет.

Источники: `PROMPTS/architecture_master.md` (J1), `docs/agents_and_responsibilities.md` (раздел 7).

---

## 1. Когда шлём алерт (переходы статуса)

Алерт отправляется при следующих переходах агрегированного статуса проверки (по потоку):

| Переход       | Тип события  | Описание                          |
|---------------|-------------|------------------------------------|
| OK → WARN     | `warn`      | Первое предупреждение по потоку   |
| WARN → FAIL   | `fail`      | Переход в FAIL (с учётом streak)  |
| FAIL → OK     | `recovered` | Восстановление потока             |

**Текущая реализация Worker:**

- **OK→WARN:** алерт шлётся при переходе из OK в WARN, если не действует cooldown по stream_id (тот же cooldown, что для fail/recovered).
- **FAIL:** алерт шлётся при переходе в статус FAIL, если выполнены условия:
  - количество подряд идущих проверок со статусом FAIL достигло порога `fail_streak_threshold` (например 2);
  - не действует cooldown по данному потоку (stream_id).
- **Recovered (FAIL→OK):** алерт шлётся, если включена настройка `send_recovered` для компании и не действует cooldown.
- **WARN→FAIL:** алерт по переходу в FAIL с учётом streak и cooldown (см. FAIL выше).

---

## 2. Формат сообщения

Сообщение в Telegram содержит:

- **Event** — тип события: `WARN`, `FAIL`, `RECOVERED`.
- **Company ID** — идентификатор компании (tenant).
- **Stream ID** — идентификатор потока.
- **Job ID** — идентификатор проверки.
- **Status** — текущий агрегированный статус (`ok` / `warn` / `fail`).
- **Decision reason** — причина решения (например `fail_streak_threshold_met`, `recovered_transition`, `cooldown_active`).

При расширении контракта можно добавить: название компании/проекта/потока (если доступны в контексте job), 1–2 причины из чеков, время проверки в явном виде.

---

## 3. Антиспам

- **Cooldown по stream:** после отправки алерта по потоку (stream_id) в течение заданного интервала (`alert_cooldown`, например 10 минут) повторный алерт по тому же потоку не отправляется. Состояние хранится в таблице `alert_state` (поля `cooldown_until`, `last_alert_at`).
- **Streak для FAIL:** алерт «переход в FAIL» отправляется только после N подряд идущих проверок со статусом FAIL (N = `fail_streak_threshold`, например 2), чтобы не слать алерт на разовый сбой.
- **Recovered:** отправка алерта «восстановление» опциональна и управляется настройкой компании `send_recovered` (Telegram delivery settings).

Реализация: `internal/domain/worker_alert_transition.go` (ComputeWorkerAlertTransition), `internal/repo/postgres/worker_alert_state_repo.go` (ApplyAlertState). Tenant scope: все операции по `company_id` и `stream_id`.

---

## 4. Безопасность и конфигурация

- Токены бота и chat_id не логируются. Токены берутся только из ENV (или через ref в настройках компании).
- Отправка выполняется в Worker с tenant scope: company_id из job, настройки загружаются по company_id.

---

## 5. Ссылки

- API настроек доставки (только CRUD, без отправки): `docs/api_contract.md` § 5.8.
- Архитектура: `PROMPTS/architecture_master.md` § J1.
- Агенты и процесс: `docs/agents_and_responsibilities.md` § 7.
