# Roll back migrations in reverse order (0006 down to 0001).
# Run from repo root. Uses DATABASE_URL from .env. Use with care (data loss possible).
$ErrorActionPreference = "Stop"
$RootDir = if ($PSScriptRoot) { (Resolve-Path (Join-Path $PSScriptRoot "..")).Path } else { (Get-Location).Path }
Set-Location $RootDir

$envPath = Join-Path $RootDir ".env"
if (-not (Test-Path $envPath)) {
    Write-Error "No .env found. Set DATABASE_URL."
    exit 1
}
Get-Content $envPath | ForEach-Object {
    if ($_ -match '^\s*([^#=]+)=(.*)$') {
        $k = $matches[1].Trim()
        $v = $matches[2].Trim() -replace '^["'']|["'']$'
        [Environment]::SetEnvironmentVariable($k, $v, 'Process')
    }
}

$db = [Environment]::GetEnvironmentVariable("DATABASE_URL", "Process")
if (-not $db) {
    Write-Error "DATABASE_URL is not set in .env"
    exit 1
}

$MigrationsDir = Join-Path $RootDir "migrations"
$downOrder = @(
    "0006_ai_incident_results.down.sql",
    "0005_indexes_admin_and_lists.down.sql",
    "0004_auth_and_registration.down.sql",
    "0003_preserve_company_audit_history.down.sql",
    "0002_telegram_delivery_settings.down.sql",
    "0001_baseline_schema.down.sql"
)

foreach ($f in $downOrder) {
    $path = Join-Path $MigrationsDir $f
    if (-not (Test-Path $path)) { continue }
    Write-Host "Rolling back $f ..."
    & psql "$db" -v ON_ERROR_STOP=1 -f "$path" 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Rollback failed at $f"
        exit 1
    }
}
Write-Host "Rollback done."
