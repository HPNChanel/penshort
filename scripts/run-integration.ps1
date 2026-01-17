# run-integration.ps1 - Integration test runner with automatic dependency management (Windows)
# This script ensures Postgres and Redis are running before running integration tests

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir

function Write-Info { param($Message) Write-Host "[INFO] $Message" -ForegroundColor Green }
function Write-Warn { param($Message) Write-Host "[WARN] $Message" -ForegroundColor Yellow }
function Write-Err { param($Message) Write-Host "[ERROR] $Message" -ForegroundColor Red }

function Test-Docker {
    try {
        $null = docker info 2>&1
        return $true
    } catch {
        return $false
    }
}

function Start-Services {
    Write-Info "Checking Docker Compose services..."
    
    Push-Location $ProjectRoot
    try {
        # Start services
        docker compose up -d postgres redis
        
        if ($LASTEXITCODE -ne 0) {
            Write-Err "Failed to start Docker Compose services"
            exit 1
        }
    } finally {
        Pop-Location
    }
}

function Wait-ForServices {
    Write-Info "Waiting for services to be healthy..."
    
    $maxAttempts = 30
    
    # Wait for Postgres
    $attempt = 0
    while ($attempt -lt $maxAttempts) {
        $result = docker compose exec -T postgres pg_isready -U penshort 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Info "PostgreSQL is ready"
            break
        }
        $attempt++
        Start-Sleep -Seconds 1
    }
    
    if ($attempt -eq $maxAttempts) {
        Write-Err "PostgreSQL did not become ready in time"
        exit 1
    }
    
    # Wait for Redis
    $attempt = 0
    while ($attempt -lt $maxAttempts) {
        $result = docker compose exec -T redis redis-cli ping 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Info "Redis is ready"
            break
        }
        $attempt++
        Start-Sleep -Seconds 1
    }
    
    if ($attempt -eq $maxAttempts) {
        Write-Err "Redis did not become ready in time"
        exit 1
    }
}

function Invoke-Migrations {
    Write-Info "Applying database migrations..."
    
    $env:DATABASE_URL = "postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable"
    
    $migratePath = Get-Command migrate -ErrorAction SilentlyContinue
    if (-not $migratePath) {
        Write-Warn "migrate tool not found, skipping migrations"
        Write-Warn "Install with: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
        return
    }
    
    Push-Location $ProjectRoot
    try {
        & migrate -path "$ProjectRoot\migrations" -database $env:DATABASE_URL up 2>&1 | Out-Null
    } finally {
        Pop-Location
    }
}

function Invoke-Tests {
    param([string[]]$TestArgs)
    
    Write-Info "Running integration tests..."
    
    $env:DATABASE_URL = "postgres://penshort:penshort@localhost:5432/penshort?sslmode=disable"
    $env:REDIS_URL = "redis://localhost:6379"
    $env:APP_ENV = "test"
    
    Push-Location $ProjectRoot
    try {
        if ($TestArgs.Count -gt 0) {
            & go test -v -race -tags=integration -run Integration @TestArgs
        } else {
            & go test -v -race -tags=integration -run Integration ./...
        }
        return $LASTEXITCODE
    } finally {
        Pop-Location
    }
}

# Main
Write-Info "=== Penshort Integration Test Runner (Windows) ==="

$skipCompose = $env:INTEGRATION_SKIP_COMPOSE -eq "true"

if (-not $skipCompose) {
    if (-not (Test-Docker)) {
        Write-Err "Docker is not available"
        exit 1
    }
    
    Start-Services
    Wait-ForServices
    Invoke-Migrations
} else {
    Write-Info "Skipping Docker Compose (INTEGRATION_SKIP_COMPOSE=true)"
}

$exitCode = Invoke-Tests -TestArgs $args

if ($exitCode -eq 0) {
    Write-Info "=== Integration tests passed ==="
} else {
    Write-Err "=== Integration tests failed ==="
}

exit $exitCode
