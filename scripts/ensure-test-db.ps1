# Load env_dev and ensure database hls_monitoring_test exists.
# Uses postgres DB to run CREATE DATABASE (idempotent).
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
$user = $env:POSTGRES_USER
$pass = $env:POSTGRES_PASSWORD
$port = $env:POSTGRES_PORT
if (-not $user) { $user = "app_postgres" }
if (-not $port) { $port = "5432" }
$connPostgres = "postgres://${user}:${pass}@127.0.0.1:${port}/postgres?sslmode=disable"
$dbName = $env:POSTGRES_DB
if (-not $dbName) { $dbName = "hls_monitoring_test" }
$result = & psql "$connPostgres" -t -A -c "SELECT 1 FROM pg_database WHERE datname='$dbName'" 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "psql failed (is PostgreSQL installed and running?). Create DB manually: createdb -U $user $dbName"
    exit 1
}
if (-not $result -or $result.Trim() -ne "1") {
    Write-Host "Creating database $dbName ..."
    & psql "$connPostgres" -c "CREATE DATABASE $dbName" 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to create database $dbName"
        exit 1
    }
    Write-Host "Created $dbName"
} else {
    Write-Host "Database $dbName already exists."
}
