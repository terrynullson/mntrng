param(
    [string]$ApiBaseUrl = "http://localhost:8080",
    [string]$Login = "",
    [string]$Password = "",
    [string]$ExpectedRole = "",
    [switch]$SkipAuth
)

$ErrorActionPreference = "Stop"

function Assert-StatusCode {
    param(
        [Parameter(Mandatory = $true)][int]$Actual,
        [Parameter(Mandatory = $true)][int]$Expected,
        [Parameter(Mandatory = $true)][string]$Step
    )
    if ($Actual -ne $Expected) {
        throw "$Step failed: expected status $Expected, got $Actual"
    }
    Write-Host "[OK] $Step ($Expected)"
}

function Invoke-Json {
    param(
        [Parameter(Mandatory = $true)][string]$Method,
        [Parameter(Mandatory = $true)][string]$Url,
        [hashtable]$Headers = @{},
        [object]$Body = $null
    )

    $params = @{ Method = $Method; Uri = $Url; Headers = $Headers; SkipHttpErrorCheck = $true }
    if ($null -ne $Body) {
        $params["Body"] = ($Body | ConvertTo-Json -Depth 10 -Compress)
        $params["ContentType"] = "application/json"
    }
    $raw = Invoke-WebRequest @params
    $json = $null
    if (-not [string]::IsNullOrWhiteSpace($raw.Content)) {
        $json = $raw.Content | ConvertFrom-Json
    }
    return @{
        StatusCode = [int]$raw.StatusCode
        Json = $json
    }
}

Write-Host "== Smoke check: $ApiBaseUrl =="

$health = Invoke-WebRequest -Uri "$ApiBaseUrl/api/v1/health" -Method GET -SkipHttpErrorCheck
Assert-StatusCode -Actual $health.StatusCode -Expected 200 -Step "health"

$ready = Invoke-WebRequest -Uri "$ApiBaseUrl/api/v1/ready" -Method GET -SkipHttpErrorCheck
Assert-StatusCode -Actual $ready.StatusCode -Expected 200 -Step "ready"

$metricsNoAuth = Invoke-WebRequest -Uri "$ApiBaseUrl/api/v1/metrics" -Method GET -SkipHttpErrorCheck
if ($metricsNoAuth.StatusCode -eq 200) {
    Write-Host "[WARN] metrics is public (API_METRICS_PUBLIC=true)"
} else {
    Write-Host "[OK] metrics protected by auth (status $($metricsNoAuth.StatusCode))"
}

if ($SkipAuth) {
    Write-Host "SkipAuth requested, smoke check completed."
    exit 0
}

if ([string]::IsNullOrWhiteSpace($Login) -or [string]::IsNullOrWhiteSpace($Password)) {
    throw "Login and Password are required when SkipAuth is not set."
}

$loginResp = Invoke-Json -Method POST -Url "$ApiBaseUrl/api/v1/auth/login" -Body @{
    login_or_email = $Login
    password = $Password
}
Assert-StatusCode -Actual $loginResp.StatusCode -Expected 200 -Step "login"

$token = $loginResp.Json
if ([string]::IsNullOrWhiteSpace($token.access_token)) {
    throw "login response has empty access_token"
}
if (-not [string]::IsNullOrWhiteSpace($ExpectedRole) -and $token.user.role -ne $ExpectedRole) {
    throw "unexpected role '$($token.user.role)', expected '$ExpectedRole'"
}
Write-Host "[OK] auth token acquired for user '$($token.user.login)' (role: $($token.user.role))"

$authHeader = @{ Authorization = "Bearer $($token.access_token)" }
$meResp = Invoke-WebRequest -Uri "$ApiBaseUrl/api/v1/auth/me" -Method GET -Headers $authHeader -SkipHttpErrorCheck
Assert-StatusCode -Actual $meResp.StatusCode -Expected 200 -Step "auth/me"

if ($token.user.company_id) {
    $otherCompany = [int64]$token.user.company_id + 100000
    $tenantEscape = Invoke-WebRequest -Uri "$ApiBaseUrl/api/v1/companies/$otherCompany/projects" -Method GET -Headers $authHeader -SkipHttpErrorCheck
    Assert-StatusCode -Actual $tenantEscape.StatusCode -Expected 403 -Step "tenant escape guard"
}

Write-Host "Smoke check completed successfully."
