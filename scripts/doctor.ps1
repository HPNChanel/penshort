# Penshort Environment Doctor (PowerShell)
# Run: .\scripts\doctor.ps1

$ErrorActionPreference = "Stop"

$errors = 0
$warnings = 0

function Ok([string]$message) {
  Write-Host "[OK] $message" -ForegroundColor Green
}

function Fail([string]$message, [string]$fix) {
  Write-Host "[FAIL] $message" -ForegroundColor Red
  Write-Host "  Fix: $fix" -ForegroundColor Yellow
  $script:errors++
}

function Warn([string]$message, [string]$fix) {
  Write-Host "[WARN] $message" -ForegroundColor Yellow
  Write-Host "  Install: $fix" -ForegroundColor Yellow
  $script:warnings++
}

function HasCommand([string]$name) {
  return $null -ne (Get-Command $name -ErrorAction SilentlyContinue)
}

Write-Host "Penshort Environment Doctor" -ForegroundColor Cyan
Write-Host "============================" -ForegroundColor Cyan
Write-Host ""

Write-Host "Required tools:" -ForegroundColor White

if (HasCommand "go") {
  $goVersion = & go version 2>$null
  if ($goVersion -match "go(\d+)\.(\d+)") {
    $major = [int]$Matches[1]
    $minor = [int]$Matches[2]
    if ($major -gt 1 -or ($major -eq 1 -and $minor -ge 22)) {
      Ok "Go 1.22+"
    } else {
      Fail "Go 1.22+" "Install Go from https://golang.org/dl/"
    }
  } else {
    Fail "Go 1.22+" "Install Go from https://golang.org/dl/"
  }
} else {
  Fail "Go 1.22+" "Install Go from https://golang.org/dl/"
}

if (HasCommand "docker") {
  & docker info 2>$null | Out-Null
  if ($LASTEXITCODE -eq 0) {
    Ok "Docker is running"
  } else {
    Fail "Docker is running" "Start Docker Desktop"
  }
} else {
  Fail "Docker is running" "Install Docker from https://docs.docker.com/get-docker/"
}

if (HasCommand "docker") {
  & docker compose version 2>$null | Out-Null
  if ($LASTEXITCODE -eq 0) {
    Ok "Docker Compose v2+"
  } else {
    Fail "Docker Compose v2+" "Install Docker Compose: https://docs.docker.com/compose/install/"
  }
} else {
  Fail "Docker Compose v2+" "Install Docker Compose: https://docs.docker.com/compose/install/"
}

if (HasCommand "git") { Ok "Git" } else { Fail "Git" "Install Git: https://git-scm.com/downloads" }

if (HasCommand "curl.exe") { Ok "curl" } else { Fail "curl" "Install curl or ensure curl.exe is on PATH" }

if (HasCommand "migrate") { Ok "migrate CLI" } else { Fail "migrate CLI" "go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest" }

if (HasCommand "golangci-lint") { Ok "golangci-lint" } else { Fail "golangci-lint" "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" }

if (HasCommand "govulncheck") { Ok "govulncheck" } else { Fail "govulncheck" "go install golang.org/x/vuln/cmd/govulncheck@latest" }

if (HasCommand "gosec") { Ok "gosec" } else { Fail "gosec" "go install github.com/securego/gosec/v2/cmd/gosec@latest" }

if (HasCommand "gitleaks") { Ok "gitleaks" } else { Fail "gitleaks" "go install github.com/zricethezav/gitleaks/v8@latest" }

Write-Host ""
Write-Host "Port availability:" -ForegroundColor White

function CheckPort([int]$port, [string]$service) {
  $inUse = $false
  if (HasCommand "Get-NetTCPConnection") {
    $conn = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
    if ($conn) { $inUse = $true }
  } else {
    $netstat = netstat -ano | Select-String -Pattern ":$port " -SimpleMatch
    if ($netstat) { $inUse = $true }
  }

  if ($inUse) {
    Fail "Port $port ($service)" "Stop the process using port $port"
  } else {
    Ok "Port $port ($service)"
  }
}

CheckPort 8080 "Penshort API"
CheckPort 5432 "PostgreSQL"
CheckPort 6379 "Redis"

Write-Host ""
Write-Host "System resources:" -ForegroundColor White

$drive = (Get-Location).Drive
$freeGb = [math]::Round($drive.Free / 1GB, 1)
if ($freeGb -ge 2) {
  Ok "Disk space ($freeGb GB available)"
} else {
  Fail "Disk space ($freeGb GB available)" "Free up disk space"
}

Write-Host ""
if ($errors -eq 0) {
  Write-Host "All required checks passed." -ForegroundColor Green
  if ($warnings -gt 0) {
    Write-Host "$warnings warning(s) detected." -ForegroundColor Yellow
  }
  Write-Host ""
  Write-Host "Next: make verify"
  exit 0
}

Write-Host ""
Write-Host "$errors required check(s) failed." -ForegroundColor Red
Write-Host "Fix the issues above and run: make doctor"
exit 1
