# Validate documentation examples against a running API (PowerShell)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$baseUrl = if ($env:PENSHORT_BASE_URL) { $env:PENSHORT_BASE_URL } else { "http://localhost:8080" }
$databaseUrl = if ($env:DATABASE_URL) { $env:DATABASE_URL } else { "postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable" }
$apiKey = $env:PENSHORT_DOCS_API_KEY

if (-not $apiKey) {
  $apiKey = & go run ./scripts/bootstrap-api-key.go -database-url $databaseUrl -name "docs-check" -scopes "admin" -format plain
}

if (-not $apiKey) {
  Write-Host "[FAIL] Could not obtain API key for docs validation" -ForegroundColor Red
  Write-Host "  Fix: set PENSHORT_DOCS_API_KEY or ensure DATABASE_URL is correct" -ForegroundColor Yellow
  exit 1
}

function Step([string]$message) {
  Write-Host "[STEP] $message" -ForegroundColor Cyan
}

Step "Health checks"
Invoke-WebRequest -Uri "$baseUrl/healthz" -UseBasicParsing | Out-Null
Invoke-WebRequest -Uri "$baseUrl/readyz" -UseBasicParsing | Out-Null

Step "Create link"
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$destination = "https://example.com/docs-$timestamp"
$body = @{ destination = $destination } | ConvertTo-Json
$headers = @{ "Authorization" = "Bearer $apiKey"; "Content-Type" = "application/json" }
$response = Invoke-RestMethod -Uri "$baseUrl/api/v1/links" -Method Post -Headers $headers -Body $body

if (-not $response.id -or -not $response.short_code) {
  Write-Host "[FAIL] Link creation response missing id or short_code" -ForegroundColor Red
  exit 1
}

$linkId = $response.id
$shortCode = $response.short_code

Step "Get link"
Invoke-WebRequest -Uri "$baseUrl/api/v1/links/$linkId" -Headers $headers -UseBasicParsing | Out-Null

Step "Redirect"
try {
  Invoke-WebRequest -Uri "$baseUrl/$shortCode" -MaximumRedirection 0 -UseBasicParsing | Out-Null
  Write-Host "[FAIL] Expected redirect response" -ForegroundColor Red
  exit 1
} catch {
  $status = $_.Exception.Response.StatusCode.value__
  if ($status -ne 301 -and $status -ne 302) {
    Write-Host "[FAIL] Expected redirect status 301/302, got $status" -ForegroundColor Red
    exit 1
  }
}

Write-Host "[OK] Docs examples validated" -ForegroundColor Green
