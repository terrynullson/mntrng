# Post-commit: load .env and send DevLog to Telegram (DEV_LOG_TELEGRAM_* in .env).
# Run from repo root. Uses last commit hash and subject for module/summary.
$ErrorActionPreference = "Stop"
$RootDir = if ($PSScriptRoot) { (Resolve-Path (Join-Path $PSScriptRoot "..")).Path } else { (Get-Location).Path }
Set-Location $RootDir

$envPath = Join-Path $RootDir ".env"
if (-not (Test-Path $envPath)) {
    Write-Host "No .env; skip DevLog notify."
    exit 0
}
Get-Content $envPath | ForEach-Object {
    if ($_ -match '^\s*([^#=]+)=(.*)$') {
        $k = $matches[1].Trim()
        $v = $matches[2].Trim() -replace '^["'']|["'']$'
        [Environment]::SetEnvironmentVariable($k, $v, 'Process')
    }
}

if ([Environment]::GetEnvironmentVariable("DEV_LOG_TELEGRAM_ENABLED", "Process") -ne "true") {
    Write-Host "DEV_LOG_TELEGRAM_ENABLED not true; skip."
    exit 0
}

$hash = (git rev-parse -q --short HEAD 2>$null)
if (-not $hash) {
    Write-Host "Could not get last commit; skip DevLog."
    exit 0
}

# Summary and mood are read inside devnotify from git (UTF-8) to avoid PowerShell/console encoding
& go run ./cmd/devnotify/ -agent=Hook -module=commit -commit=$hash -readSummaryFromGit
if ($LASTEXITCODE -ne 0) {
    Write-Host "DevLog notify failed (check DEV_LOG_TELEGRAM_TOKEN and DEV_LOG_TELEGRAM_CHAT_ID in .env)."
    exit 0
}
Write-Host "DevLog sent to Telegram."
