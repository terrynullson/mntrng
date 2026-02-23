# Create a PostgreSQL backup (pg_dump) using DATABASE_URL from .env.
# Run from repo root. Output: backups/hls_monitoring_YYYYMMDD_HHMMSS.sql
$ErrorActionPreference = "Stop"
$RootDir = if ($PSScriptRoot) { (Resolve-Path (Join-Path $PSScriptRoot "..")).Path } else { (Get-Location).Path }
Set-Location $RootDir

$envPath = Join-Path $RootDir ".env"
if (-not (Test-Path $envPath)) {
    Write-Error "No .env found. Create .env from .env.example and set DATABASE_URL."
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

$BackupsDir = Join-Path $RootDir "backups"
if (-not (Test-Path $BackupsDir)) {
    New-Item -ItemType Directory -Path $BackupsDir | Out-Null
}
$timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$outFile = Join-Path $BackupsDir "hls_monitoring_$timestamp.sql"

Write-Host "Backing up to $outFile ..."
& pg_dump "$db" -v ON_ERROR_STOP=1 -f "$outFile" 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Error "pg_dump failed. Ensure PostgreSQL client (pg_dump) is in PATH."
    exit 1
}
Write-Host "Backup done: $outFile"
