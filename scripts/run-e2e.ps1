# Penshort E2E Test Runner (PowerShell)
# Run: .\scripts\run-e2e.ps1 or make test-e2e (on Windows with PowerShell)
#
# Environment Variables:
#   PENSHORT_BASE_URL - API base URL (default: http://localhost:8080)
#   E2E_SKIP_COMPOSE  - Set to "1" to skip Docker Compose startup
#   E2E_SKIP_DOWN     - Set to "1" to skip Docker Compose teardown

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if (-not $env:PENSHORT_BASE_URL) {
    $env:PENSHORT_BASE_URL = "http://localhost:8080"
}

$skipCompose = $env:E2E_SKIP_COMPOSE -eq "1"
$skipDown = $env:E2E_SKIP_DOWN -eq "1"

function Cleanup {
    if ($skipDown -or $skipCompose) { return }
    Write-Host "[CLEANUP] Stopping Docker Compose..." -ForegroundColor Gray
    & docker compose down -v 2>$null | Out-Null
}

function Wait-ForReady {
    param([int]$maxAttempts = 30)
    
    Write-Host "[STEP] Waiting for API readiness..." -ForegroundColor Cyan
    
    for ($i = 1; $i -le $maxAttempts; $i++) {
        try {
            $resp = Invoke-WebRequest -Uri "$env:PENSHORT_BASE_URL/readyz" -UseBasicParsing -TimeoutSec 2 -ErrorAction SilentlyContinue
            if ($resp.StatusCode -eq 200) {
                Write-Host "[OK] API is ready" -ForegroundColor Green
                return
            }
        } catch {
            # Continue waiting
        }
        Write-Host "  Waiting for API ($i/$maxAttempts)..." -ForegroundColor Gray
        Start-Sleep -Seconds 2
    }
    
    Write-Host "[FAIL] API did not become ready within $($maxAttempts * 2) seconds" -ForegroundColor Red
    Write-Host "  Fix: Check 'docker compose logs api' for errors" -ForegroundColor Yellow
    Write-Host "  Fix: Ensure DATABASE_URL and REDIS_URL are correct" -ForegroundColor Yellow
    & docker compose logs api 2>$null | Select-Object -Last 20
    exit 1
}

try {
    Write-Host "Penshort E2E Test Runner" -ForegroundColor Cyan
    Write-Host "==========================" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Base URL: $env:PENSHORT_BASE_URL" -ForegroundColor White

    if (-not $skipCompose) {
        Write-Host ""
        Write-Host "[STEP] Starting Docker Compose stack..." -ForegroundColor Cyan
        & docker compose up -d --build
        if ($LASTEXITCODE -ne 0) {
            Write-Host "[FAIL] Docker Compose failed to start" -ForegroundColor Red
            Write-Host "  Fix: Run 'docker compose up' manually to see detailed errors" -ForegroundColor Yellow
            exit 1
        }
        
        Wait-ForReady
    } else {
        Write-Host ""
        Write-Host "[SKIP] Docker Compose startup (E2E_SKIP_COMPOSE=1)" -ForegroundColor Yellow
    }

    Write-Host ""
    Write-Host "[STEP] Running E2E tests..." -ForegroundColor Cyan
    & go test -v -tags=e2e ./tests/e2e/...
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host ""
        Write-Host "[FAIL] E2E tests failed" -ForegroundColor Red
        Write-Host "  Fix: Review test output above for specific failures" -ForegroundColor Yellow
        Write-Host "  Fix: Check 'docker compose logs api' for server errors" -ForegroundColor Yellow
        exit 1
    }

    Write-Host ""
    Write-Host "[OK] E2E tests passed" -ForegroundColor Green
    
} finally {
    Cleanup
}
