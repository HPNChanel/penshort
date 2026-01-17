# Penshort verification runner (PowerShell)
# Runs doctor, starts dependencies, applies migrations, runs tests, docs checks, and security scans.

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

if (-not $env:PENSHORT_BASE_URL) {
  $env:PENSHORT_BASE_URL = "http://localhost:8080"
}
if (-not $env:DATABASE_URL) {
  $env:DATABASE_URL = "postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable"
}
if (-not $env:REDIS_URL) {
  $env:REDIS_URL = "redis://localhost:6379"
}
$env:PENSHORT_URL = $env:PENSHORT_BASE_URL

function Step([string]$message) {
  Write-Host "" 
  Write-Host "[STEP] $message" -ForegroundColor Cyan
}

function Wait-ForHealth([string]$service, [int]$maxAttempts = 30) {
  $id = & docker compose ps -q $service
  if (-not $id) {
    Write-Host "[FAIL] Service $service is not running" -ForegroundColor Red
    Write-Host "  Fix: run 'docker compose up -d $service'" -ForegroundColor Yellow
    exit 1
  }

  for ($i = 1; $i -le $maxAttempts; $i++) {
    $status = & docker inspect --format '{{.State.Health.Status}}' $id 2>$null
    if ($status -eq "healthy") {
      Write-Host "[OK] $service is healthy" -ForegroundColor Green
      return
    }
    Start-Sleep -Seconds 2
  }

  Write-Host "[FAIL] $service did not become healthy" -ForegroundColor Red
  & docker compose logs $service | Out-Null
  exit 1
}

function Cleanup {
  if ($env:VERIFY_SKIP_DOWN -eq "1") { return }
  & docker compose down | Out-Null
}

try {
  Step "Doctor"
  & "$PSScriptRoot\doctor.ps1"

  Step "Start dependencies (postgres, redis)"
  & docker compose up -d postgres redis
  Wait-ForHealth "postgres"
  Wait-ForHealth "redis"

  Step "Apply migrations"
  & migrate -path migrations -database $env:DATABASE_URL up

  Step "Lint"
  & golangci-lint run ./...

  Step "Start API"
  & docker compose up -d api

  $ready = $false
  for ($i = 1; $i -le 30; $i++) {
    try {
      $resp = Invoke-WebRequest -Uri "$env:PENSHORT_BASE_URL/readyz" -UseBasicParsing -TimeoutSec 2
      if ($resp.StatusCode -eq 200) { $ready = $true; break }
    } catch {}
    Start-Sleep -Seconds 2
  }
  if (-not $ready) {
    Write-Host "[FAIL] API did not become ready" -ForegroundColor Red
    & docker compose logs api | Out-Null
    exit 1
  }

  Step "Unit tests"
  & go test -v -race -short -coverprofile=coverage.out ./...

  Step "Integration tests"
  & go test -v -race -tags=integration -run Integration ./...

  Step "Contract tests (OpenAPI schema)"
  & go test -v ./tests/contract/...

  Step "E2E smoke tests"
  & go test -v -tags=e2e ./tests/e2e/...

  Step "Docs examples validation"
  & "$PSScriptRoot\validate-docs.ps1"

  Step "Security checks"
  & "$PSScriptRoot\security.ps1"

  Write-Host "" 
  Write-Host "[OK] Verify complete" -ForegroundColor Green
} finally {
  Cleanup
}
