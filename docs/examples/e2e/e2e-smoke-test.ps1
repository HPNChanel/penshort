# Penshort End-to-End Smoke Test (PowerShell)
#
# Usage:
#   .\e2e-smoke-test.ps1
#
# With existing API key:
#   $env:API_KEY = "pk_live_xxx"
#   .\e2e-smoke-test.ps1

$ErrorActionPreference = "Stop"

$baseUrl = if ($env:BASE_URL) { $env:BASE_URL } else { "http://localhost:8080" }
$databaseUrl = if ($env:DATABASE_URL) { $env:DATABASE_URL } else { "postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable" }
$apiKey = $env:API_KEY

if (-not $apiKey) {
  $apiKey = & go run ./scripts/bootstrap-api-key.go -database-url $databaseUrl -name "e2e-smoke" -scopes "admin" -format plain
}

if (-not $apiKey) {
  Write-Host "Failed to obtain API key. Set API_KEY or DATABASE_URL." -ForegroundColor Red
  exit 1
}

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  Penshort End-to-End Smoke Test" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Step 1: Health checks" -ForegroundColor Yellow
Invoke-WebRequest -Uri "$baseUrl/healthz" -UseBasicParsing | Out-Null
Invoke-WebRequest -Uri "$baseUrl/readyz" -UseBasicParsing | Out-Null

Write-Host "Step 2: Create link" -ForegroundColor Yellow
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$destination = "https://example.com/e2e-test-$timestamp"
$alias = "e2e-test-$timestamp"

$headers = @{ "Authorization" = "Bearer $apiKey"; "Content-Type" = "application/json" }
$body = @{ destination = $destination; alias = $alias; redirect_type = 302 } | ConvertTo-Json

$link = Invoke-RestMethod -Uri "$baseUrl/api/v1/links" -Method Post -Headers $headers -Body $body

if (-not $link.id -or -not $link.short_code) {
  Write-Host "Failed to create link" -ForegroundColor Red
  exit 1
}

$linkId = $link.id
$shortCode = $link.short_code

Write-Host "Step 3: Redirect" -ForegroundColor Yellow
try {
  Invoke-WebRequest -Uri "$baseUrl/$shortCode" -MaximumRedirection 0 -UseBasicParsing | Out-Null
  Write-Host "Expected redirect response" -ForegroundColor Red
  exit 1
} catch {
  $status = $_.Exception.Response.StatusCode.value__
  if ($status -ne 301 -and $status -ne 302) {
    Write-Host "Redirect failed (status $status)" -ForegroundColor Red
    exit 1
  }
}

Write-Host "Step 4: Generate clicks" -ForegroundColor Yellow
for ($i = 1; $i -le 3; $i++) {
  try {
    Invoke-WebRequest -Uri "$baseUrl/$shortCode" -MaximumRedirection 0 -UseBasicParsing | Out-Null
  } catch {}
  Start-Sleep -Milliseconds 200
}

Start-Sleep -Seconds 1

Write-Host "Step 5: Analytics" -ForegroundColor Yellow
$analytics = Invoke-RestMethod -Uri "$baseUrl/api/v1/links/$linkId/analytics" -Headers $headers
if (-not $analytics.summary) {
  Write-Host "Analytics response missing summary" -ForegroundColor Red
  exit 1
}

Write-Host "Step 6: Disable link" -ForegroundColor Yellow
$disableBody = '{"enabled": false}'
Invoke-RestMethod -Uri "$baseUrl/api/v1/links/$linkId" -Method Patch -Headers $headers -Body $disableBody | Out-Null

Write-Host "" 
Write-Host "Smoke test complete." -ForegroundColor Green
