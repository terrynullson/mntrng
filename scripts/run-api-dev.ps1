# Load env_dev and start API (for test DB / screenshot pipeline).
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
if (-not $env:DATABASE_URL) {
    Write-Error "DATABASE_URL not set in env_dev"
    exit 1
}
Set-Location $RootDir
& go run ./cmd/api/
