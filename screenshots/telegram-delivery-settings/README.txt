Screenshot for FE-TDS-001: Company Telegram Alerts settings.

- 20260222185800.png: НЕКОРРЕКТЕН (страница логина). Для PASS ReviewAgent нужен скриншот именно секции «Telegram Alerts (Company)» на /settings.
- Как получить корректный скриншот:
  1) В .env задать DATABASE_URL (подключение к PostgreSQL с применёнными миграциями).
  2) Запустить API: go run ./cmd/api/ (из корня).
  3) В другом терминале: cd web && npm run screenshot:settings.
  Скрипт выполнит сидер (test_screenshot_admin), логин, /settings, fullPage screenshot в screenshots/telegram-delivery-settings/<timestamp>.png и REPORT.txt.
