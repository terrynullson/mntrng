# Load env_dev and run migrations on hls_monitoring_test.
# Prerequisite: database must exist (e.g. createdb -U app_postgres hls_monitoring_test).
$RootDir = Split-Path -Parent $PSScriptRoot
$EnvFile = Join-Path $RootDir "env_dev"
if (-not (Test-Path $EnvFile)) {
    Write-Error "env_dev not found at $EnvFile"
    exit 1
}
Get-Content $EnvFile | ForEach-Object {
    if ($_ -match '^\s*([^#=]+)=(.*)$') {
        $k = $matches[1].Trim()
        $v = $matches[2].Trim() -replace '^["'']|["'']$'
        [Environment]::SetEnvironmentVariable($k, $v, 'Process')
    }
}
$db = $env:DATABASE_URL
if (-not $db) {
    Write-Error "DATABASE_URL not set in env_dev"
    exit 1
}
$MigrationsDir = Join-Path $RootDir "migrations"
$order = @(
    "0001_baseline_schema.up.sql",
    "0002_telegram_delivery_settings.up.sql",
    "0003_preserve_company_audit_history.up.sql",
    "0004_auth_and_registration.up.sql",
    "0005_indexes_admin_and_lists.up.sql",
    "0006_ai_incident_results.up.sql",
    "0007_stream_favorites_and_incidents.up.sql",
    "0008_embed_whitelist_and_stream_sources.up.sql",
    "0009_incident_diagnostics_lite.up.sql"
)
foreach ($f in $order) {
    $path = Join-Path $MigrationsDir $f
    if (-not (Test-Path $path)) { continue }
    Write-Host "Applying $f ..."
    & psql "$db" -v ON_ERROR_STOP=1 -f "$path" 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Migration $f failed. Ensure DB exists: createdb -U app_postgres hls_monitoring_test"
        exit 1
    }
}
Write-Host "Migrations done."
