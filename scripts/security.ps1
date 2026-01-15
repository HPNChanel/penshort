# Penshort security checks (PowerShell)
# Runs secrets scan (gitleaks), dependency scan (govulncheck), and SAST (gosec)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

function Require-Command([string]$name, [string]$fix) {
  if (-not (Get-Command $name -ErrorAction SilentlyContinue)) {
    Write-Host "[FAIL] Missing required tool: $name" -ForegroundColor Red
    Write-Host "  Fix: $fix" -ForegroundColor Yellow
    exit 1
  }
}

Require-Command "gitleaks" "go install github.com/zricethezav/gitleaks/v8@latest"
Require-Command "govulncheck" "go install golang.org/x/vuln/cmd/govulncheck@latest"
Require-Command "gosec" "go install github.com/securego/gosec/v2/cmd/gosec@latest"

Write-Host "[STEP] Secret scan (gitleaks)"
& gitleaks detect --source . --no-banner

Write-Host "[STEP] Dependency scan (govulncheck)"
& govulncheck ./...

Write-Host "[STEP] Static analysis (gosec)"
& gosec -exclude-dir=testdata -exclude-dir=tests -exclude-dir=docs/examples ./...

Write-Host "[OK] Security checks passed" -ForegroundColor Green
