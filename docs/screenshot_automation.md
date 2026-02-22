# Screenshot automation (UI modules)

Для автоматического снятия скриншотов защищённых страниц (например, секция «Telegram Alerts (Company)» на `/settings`) используется тестовая учётная запись. Только для local/dev; в прод не разворачивать.

## Тестовая БД и env_dev

Чтобы не трогать основную БД и иметь предсказуемое окружение для скриншотов и тестов, в репозитории есть конфиг **env_dev** (в корне) и скрипты в **scripts/**:

- **env_dev** — переменные для тестовой БД `hls_monitoring_test` (логин/пароль как в .env.example). Файл в репозитории; при необходимости скопируйте в `.env.dev` и подставьте свои значения (`.env.dev` в .gitignore).
- **scripts/ensure-test-db.ps1** — создаёт БД `hls_monitoring_test`, если её ещё нет (требует `psql` в PATH).
- **scripts/run-migrations-dev.ps1** — подгружает env_dev и применяет миграции к тестовой БД. Запускать после создания БД.
- **scripts/run-api-dev.ps1** — подгружает env_dev и запускает API (`go run ./cmd/api/`). Использовать для пайплайна скриншотов.

**Однократная подготовка тестовой БД (Windows, PowerShell из корня репозитория):**

```powershell
.\scripts\ensure-test-db.ps1
.\scripts\run-migrations-dev.ps1
```

Далее для автоматического скриншота: в одном терминале запустить `.\scripts\run-api-dev.ps1`, в другом — из `web/` выполнить `$env:ENV_FILE="env_dev"; npm run screenshot:settings`. Пайплайн возьмёт `DATABASE_URL` из env_dev, запустит сидер и сделает скриншот.

## Тестовая учётная запись

- **Login:** `test_screenshot_admin`
- **Password:** `TestScreenshot1`
- **Роль:** `company_admin`
- **Компания:** создаётся сидером «Screenshot Test Company»; у пользователя `company_id` этой компании, scope подставляется автоматически (выбор в topbar для company_admin не показывается).

## Подготовка (один раз или после сброса БД)

1. Применить миграции (как в README).
2. Запустить сидер:
   ```bash
   export DATABASE_URL="postgres://..."
   go run ./cmd/seed/
   ```
   Или из корня: `go run ./cmd/seed/` при установленном `DATABASE_URL`. Сидер идемпотентен: повторный запуск не падает (если пользователь уже есть — пишет "user already exists").

## Снятие скриншота (ReviewAgent / любой агент с MCP browser)

1. **Запустить API** (порт 8080 по умолчанию): `go run ./cmd/api/` или через Docker.
2. **Запустить frontend:** `cd web && npm run dev` (порт 3000).
3. **Браузер (MCP cursor-ide-browser):**
   - Перейти на `http://localhost:3000/login`.
   - Заполнить поле «Login or email» (id: `login-or-email`): `test_screenshot_admin`.
   - Заполнить поле «Password» (id: `login-password`): `TestScreenshot1`.
   - Нажать кнопку входа (Login).
   - Дождаться редиректа на главную.
4. **Перейти на целевую страницу**, например `http://localhost:3000/settings`.
5. **Сделать скриншот** (например, `browser_take_screenshot`) и сохранить в `screenshots/<module>/<timestamp>.png`. Файл из Temp при необходимости скопировать в корень репозитория в ту же относительную путь.

Для секции «Telegram Alerts (Company)» модуль — `telegram-delivery-settings`; путь: `screenshots/telegram-delivery-settings/<timestamp>.png`.

## Полностью автоматический pipeline (без участия пользователя)

Из каталога `web/` выполнить:

```bash
npm run screenshot:settings
```

Скрипт (Playwright):

1. Проверяет, что API запущен на порту 8080 (иначе выходит с подсказкой запустить `go run ./cmd/api/`).
2. При наличии `DATABASE_URL` в окружении (или в `.env` в корне репозитория) запускает сидер `go run ./cmd/seed/`.
3. Если dev-сервер не слушает порт 3000 — запускает `npm run dev` и ждёт готовности.
4. Логинится на `/login` (учётная запись по умолчанию: `test_screenshot_admin` / `TestScreenshot1`; переопределение через `SCREENSHOT_LOGIN`, `SCREENSHOT_PASSWORD`).
5. Открывает `/settings`, делает fullPage-скриншот, сохраняет в `screenshots/telegram-delivery-settings/<timestamp>.png`.
6. Пишет self-score и краткие проверки в `screenshots/telegram-delivery-settings/REPORT.txt` и в stdout.
7. Выполняет `git add -A` и `git commit -m "ui: automate settings page screenshot (playwright)"` (если нечего коммитить — коммит пропускается).

Требования: API должен быть запущен (например, `.\scripts\run-api-dev.ps1` с env_dev); для сидера — Go и `DATABASE_URL` (при использовании env_dev задать `$env:ENV_FILE="env_dev"` перед `npm run screenshot:settings`). Первый запуск может скачать Chromium (`npx playwright install chromium`).

## Безопасность

- Учётная запись предназначена только для локальной автоматизации скриншотов.
- Не использовать в production; не коммитить реальные прод-пароли.
- В `.env.example` и публичной документации указывать только факт наличия сидера и ссылку на этот файл, без пароля в открытом виде (пароль зафиксирован в коде сидера и в этом документе как dev-only).
