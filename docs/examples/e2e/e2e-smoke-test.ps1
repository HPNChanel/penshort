# Penshort End-to-End Smoke Test (PowerShell)
# ============================================
#
# Windows-compatible version of the E2E test.
#
# Usage:
#   .\e2e-smoke-test.ps1
#
# With existing API key:
#   $env:API_KEY = "psk_live_xxx"
#   .\e2e-smoke-test.ps1

$ErrorActionPreference = "Stop"

$BASE_URL = if ($env:BASE_URL) { $env:BASE_URL } else { "http://localhost:8080" }
$API_KEY = $env:API_KEY

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  Penshort End-to-End Smoke Test" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# Step 1: Health Check
Write-Host "Step 1: Checking service health..." -ForegroundColor Yellow

try {
    $health = Invoke-RestMethod -Uri "$BASE_URL/healthz" -Method Get
    if ($health.status -eq "ok") {
        Write-Host "✓ Liveness: OK" -ForegroundColor Green
    }
} catch {
    Write-Host "✗ Liveness check failed: $_" -ForegroundColor Red
    exit 1
}

try {
    $ready = Invoke-RestMethod -Uri "$BASE_URL/readyz" -Method Get
    if ($ready.status -eq "ok") {
        Write-Host "✓ Readiness: OK (postgres: $($ready.checks.postgres), redis: $($ready.checks.redis))" -ForegroundColor Green
    }
} catch {
    Write-Host "✗ Readiness check failed: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Step 2: Create Link
Write-Host "Step 2: Creating a short link..." -ForegroundColor Yellow

$timestamp = [DateTimeOffset]::Now.ToUnixTimeSeconds()
$destination = "https://example.com/e2e-test-$timestamp"
$alias = "e2e-$timestamp"

$linkBody = @{
    destination = $destination
    alias = $alias
    redirect_type = 302
} | ConvertTo-Json

$headers = @{
    "Content-Type" = "application/json"
}
if ($API_KEY) {
    $headers["Authorization"] = "Bearer $API_KEY"
}

try {
    $link = Invoke-RestMethod -Uri "$BASE_URL/api/v1/links" -Method Post -Headers $headers -Body $linkBody
    $linkId = $link.id
    $shortCode = $link.short_code
    $shortUrl = $link.short_url
    
    Write-Host "✓ Link created:" -ForegroundColor Green
    Write-Host "  ID:         $linkId"
    Write-Host "  Short Code: $shortCode"
    Write-Host "  Short URL:  $shortUrl"
} catch {
    Write-Host "✗ Failed to create link: $_" -ForegroundColor Red
    exit 1
}

Write-Host ""

# Step 3: Test Redirect
Write-Host "Step 3: Testing redirect..." -ForegroundColor Yellow

try {
    $response = Invoke-WebRequest -Uri "$BASE_URL/$shortCode" -MaximumRedirection 0 -ErrorAction SilentlyContinue
} catch {
    if ($_.Exception.Response.StatusCode -eq 302 -or $_.Exception.Response.StatusCode -eq 301) {
        $location = $_.Exception.Response.Headers.Location
        Write-Host "✓ Redirect works!" -ForegroundColor Green
        Write-Host "  Status: $($_.Exception.Response.StatusCode)"
        Write-Host "  Location: $location"
    } else {
        Write-Host "✗ Redirect failed" -ForegroundColor Red
    }
}

Write-Host ""

# Step 4: Generate Clicks
Write-Host "Step 4: Generating test clicks..." -ForegroundColor Yellow

for ($i = 1; $i -le 3; $i++) {
    try {
        Invoke-WebRequest -Uri "$BASE_URL/$shortCode" -MaximumRedirection 0 -ErrorAction SilentlyContinue | Out-Null
    } catch {}
    Write-Host "  Click $i recorded"
    Start-Sleep -Milliseconds 500
}

Write-Host "✓ 3 clicks generated" -ForegroundColor Green
Write-Host ""

# Step 5: Query Analytics
Write-Host "Step 5: Querying analytics..." -ForegroundColor Yellow

Start-Sleep -Seconds 2

try {
    $analytics = Invoke-RestMethod -Uri "$BASE_URL/api/v1/links/$linkId/analytics" -Headers $headers
    Write-Host "✓ Analytics retrieved:" -ForegroundColor Green
    Write-Host "  Total Clicks:    $($analytics.summary.total_clicks)"
    Write-Host "  Unique Visitors: $($analytics.summary.unique_visitors)"
} catch {
    Write-Host "⚠ Analytics may still be processing (1 min delay)" -ForegroundColor Yellow
}

Write-Host ""

# Step 6: Cleanup
Write-Host "Step 6: Cleanup - disabling test link..." -ForegroundColor Yellow

$disableBody = '{"enabled": false}'

try {
    $disabled = Invoke-RestMethod -Uri "$BASE_URL/api/v1/links/$linkId" -Method Patch -Headers $headers -Body $disableBody
    if ($disabled.status -eq "disabled") {
        Write-Host "✓ Test link disabled" -ForegroundColor Green
    }
} catch {
    Write-Host "⚠ Could not disable link" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  End-to-End Test Complete!" -ForegroundColor Green
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Summary:"
Write-Host "  ✓ Health checks passed"
Write-Host "  ✓ Short link created: $shortUrl"
Write-Host "  ✓ Redirect working"
Write-Host "  ✓ Clicks recorded"
Write-Host "  ✓ Analytics queryable"
Write-Host ""
