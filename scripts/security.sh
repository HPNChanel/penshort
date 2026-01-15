#!/usr/bin/env bash
# Penshort security checks
# Runs secrets scan (gitleaks), dependency scan (govulncheck), and SAST (gosec)

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

require_cmd() {
  local cmd="$1"
  local fix="$2"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "[FAIL] Missing required tool: $cmd"
    echo "  Fix: $fix"
    exit 1
  fi
}

require_cmd "gitleaks" "go install github.com/zricethezav/gitleaks/v8@latest"
require_cmd "govulncheck" "go install golang.org/x/vuln/cmd/govulncheck@latest"
require_cmd "gosec" "go install github.com/securego/gosec/v2/cmd/gosec@latest"

echo "[STEP] Secret scan (gitleaks)"
gitleaks detect --source . --no-banner

echo "[STEP] Dependency scan (govulncheck)"
govulncheck ./...

echo "[STEP] Static analysis (gosec)"
gosec -exclude-dir=testdata -exclude-dir=tests -exclude-dir=docs/examples ./...

echo "[OK] Security checks passed"
