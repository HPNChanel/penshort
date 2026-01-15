#!/usr/bin/env bash
# Penshort Environment Doctor
# Diagnoses missing dependencies and environment issues
# Run: make doctor

set -euo pipefail

ERRORS=0
WARNINGS=0

ok() {
  echo "[OK] $1"
}

fail() {
  echo "[FAIL] $1"
  echo "  Fix: $2"
  ERRORS=$((ERRORS + 1))
}

warn() {
  echo "[WARN] $1"
  echo "  Install: $2"
  WARNINGS=$((WARNINGS + 1))
}

has_cmd() {
  command -v "$1" >/dev/null 2>&1
}

go_version_ok() {
  local version
  version=$(go version 2>/dev/null | awk '{print $3}' | sed 's/^go//')
  if [ -z "$version" ]; then
    return 1
  fi
  local major minor
  major=$(echo "$version" | cut -d. -f1)
  minor=$(echo "$version" | cut -d. -f2)
  if [ "${major:-0}" -gt 1 ]; then
    return 0
  fi
  if [ "${major:-0}" -eq 1 ] && [ "${minor:-0}" -ge 22 ]; then
    return 0
  fi
  return 1
}

echo "Penshort Environment Doctor"
echo "============================"
echo ""

echo "Required tools:"
if go_version_ok; then
  ok "Go 1.22+"
else
  fail "Go 1.22+" "Install Go from https://golang.org/dl/"
fi

if has_cmd docker && docker info >/dev/null 2>&1; then
  ok "Docker is running"
else
  fail "Docker is running" "Start Docker Desktop or run: sudo systemctl start docker"
fi

if has_cmd docker && docker compose version >/dev/null 2>&1; then
  ok "Docker Compose v2+"
else
  fail "Docker Compose v2+" "Install Docker Compose: https://docs.docker.com/compose/install/"
fi

if has_cmd git; then
  ok "Git"
else
  fail "Git" "Install Git: https://git-scm.com/downloads"
fi

if has_cmd curl; then
  ok "curl"
else
  fail "curl" "Install curl (package manager or https://curl.se/download.html)"
fi

if has_cmd migrate; then
  ok "migrate CLI"
else
  fail "migrate CLI" "go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
fi

if has_cmd golangci-lint; then
  ok "golangci-lint"
else
  fail "golangci-lint" "go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
fi

if has_cmd govulncheck; then
  ok "govulncheck"
else
  fail "govulncheck" "go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

if has_cmd gosec; then
  ok "gosec"
else
  fail "gosec" "go install github.com/securego/gosec/v2/cmd/gosec@latest"
fi

if has_cmd gitleaks; then
  ok "gitleaks"
else
  fail "gitleaks" "go install github.com/zricethezav/gitleaks/v8@latest"
fi

echo ""
echo "Port availability:"
check_port() {
  local port="$1"
  local service="$2"
  if has_cmd lsof; then
    if lsof -i :"$port" -sTCP:LISTEN >/dev/null 2>&1; then
      fail "Port $port ($service)" "Stop the process using port $port: lsof -i :$port"
    else
      ok "Port $port ($service)"
    fi
  elif has_cmd netstat; then
    if netstat -tuln 2>/dev/null | grep -q ":$port "; then
      fail "Port $port ($service)" "Stop the process using port $port"
    else
      ok "Port $port ($service)"
    fi
  elif has_cmd ss; then
    if ss -tuln 2>/dev/null | grep -q ":$port "; then
      fail "Port $port ($service)" "Stop the process using port $port"
    else
      ok "Port $port ($service)"
    fi
  else
    warn "Port $port ($service)" "Install lsof, netstat, or ss to check ports"
  fi
}

check_port 8080 "Penshort API"
check_port 5432 "PostgreSQL"
check_port 6379 "Redis"

echo ""
echo "System resources:"
if has_cmd df; then
  DISK_AVAIL=$(df -BG . 2>/dev/null | tail -1 | awk '{print $4}' | tr -d 'G' || echo "0")
  if [ "${DISK_AVAIL:-0}" -ge 2 ]; then
    ok "Disk space (${DISK_AVAIL}GB available)"
  else
    fail "Disk space (${DISK_AVAIL}GB available)" "Free up disk space in the current directory"
  fi
fi

echo ""
if [ "$ERRORS" -eq 0 ]; then
  echo "All required checks passed."
  if [ "$WARNINGS" -gt 0 ]; then
    echo "$WARNINGS warning(s) detected."
  fi
  echo ""
  echo "Next: make verify"
  exit 0
else
  echo ""
  echo "$ERRORS required check(s) failed."
  echo "Fix the issues above and run: make doctor"
  exit 1
fi
