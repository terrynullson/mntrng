# Диагностика: почему не приходят сообщения в Telegram DevLog.
# Запуск из корня репозитория: .\scripts\devlog_notify_check.ps1
# Выполняет агент при отсутствии доставки в TG (пользователь проверку вручную не делает). См. docs/agents_and_responsibilities.md, раздел 9.

$ErrorActionPreference = "Stop"
$RootDir = if ($PSScriptRoot) { (Resolve-Path (Join-Path $PSScriptRoot "..")).Path } else { (Get-Location).Path }
Set-Location $RootDir

Write-Host "=== DevLog Telegram check ===" -ForegroundColor Cyan
Write-Host ""

# 1. Git hooks path
$hooksPath = git config --get core.hooksPath 2>$null
if (-not $hooksPath) {
    Write-Host "[!] core.hooksPath не задан. Хук post-commit не вызывается при коммите." -ForegroundColor Yellow
    Write-Host "    Выполни: git config core.hooksPath .githooks" -ForegroundColor Gray
} else {
    Write-Host "[OK] core.hooksPath = $hooksPath" -ForegroundColor Green
}

# 2. .env
$envPath = Join-Path $RootDir ".env"
if (-not (Test-Path $envPath)) {
    Write-Host "[!] Файл .env не найден в корне репозитория." -ForegroundColor Yellow
    exit 1
}
Write-Host "[OK] .env найден" -ForegroundColor Green

Get-Content $envPath | ForEach-Object {
    if ($_ -match '^\s*([^#=]+)=(.*)$') {
        $k = $matches[1].Trim()
        $v = $matches[2].Trim() -replace '^["'']|["'']$'
        [Environment]::SetEnvironmentVariable($k, $v, 'Process')
    }
}

$enabled = [Environment]::GetEnvironmentVariable("DEV_LOG_TELEGRAM_ENABLED", "Process")
$token = [Environment]::GetEnvironmentVariable("DEV_LOG_TELEGRAM_TOKEN", "Process")
$chatId = [Environment]::GetEnvironmentVariable("DEV_LOG_TELEGRAM_CHAT_ID", "Process")

if ($enabled -ne "true") {
    Write-Host "[!] DEV_LOG_TELEGRAM_ENABLED не true (сейчас: '$enabled'). В .env поставь DEV_LOG_TELEGRAM_ENABLED=true" -ForegroundColor Yellow
}
else { Write-Host "[OK] DEV_LOG_TELEGRAM_ENABLED=true" -ForegroundColor Green }

if ([string]::IsNullOrWhiteSpace($token)) {
    Write-Host "[!] DEV_LOG_TELEGRAM_TOKEN пустой или отсутствует в .env" -ForegroundColor Yellow
}
else { Write-Host "[OK] DEV_LOG_TELEGRAM_TOKEN задан" -ForegroundColor Green }

if ([string]::IsNullOrWhiteSpace($chatId)) {
    Write-Host "[!] DEV_LOG_TELEGRAM_CHAT_ID пустой или отсутствует в .env" -ForegroundColor Yellow
}
else { Write-Host "[OK] DEV_LOG_TELEGRAM_CHAT_ID задан" -ForegroundColor Green }

Write-Host ""

# 3. Последний коммит
$hash = git rev-parse -q --short HEAD 2>$null
if (-not $hash) {
    Write-Host "[!] Не удалось получить hash последнего коммита (git rev-parse HEAD)" -ForegroundColor Yellow
}
else {
    Write-Host "[OK] Последний коммит: $hash" -ForegroundColor Green
}

# 4. Запуск devnotify (тест)
Write-Host ""
Write-Host "Запуск: go run ./cmd/devnotify/ -test (тестовое сообщение в Telegram)..." -ForegroundColor Cyan
& go run ./cmd/devnotify/ -test
if ($LASTEXITCODE -ne 0) {
    Write-Host "[!] devnotify -test завершился с ошибкой. Проверь токен, chat_id и сеть." -ForegroundColor Yellow
    exit 1
}
Write-Host "[OK] Тестовое сообщение отправлено. Проверь Telegram." -ForegroundColor Green
Write-Host ""
Write-Host "Если тест прошёл, но после коммитов сообщения не приходят — проверь, что git config core.hooksPath .githooks выполнен в этом репозитории." -ForegroundColor Gray
