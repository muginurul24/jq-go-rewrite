#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

log_step() {
  printf '\n[%s] %s\n' "$(date '+%H:%M:%S')" "$1"
}

log_step "running runtime smoke"
"${ROOT_DIR}/scripts/smoke-runtime.sh"

log_step "capturing finance reconciliation snapshot"
"${ROOT_DIR}/scripts/reconcile-finance.sh"

if command -v curl >/dev/null 2>&1; then
  BASE_URL="${SMOKE_BASE_URL:-${APP_URL:-http://localhost:8080}}"
  BASE_URL="${BASE_URL%/}"

  log_step "checking metrics endpoint"
  curl --silent --show-error --fail "${BASE_URL}/metrics" >/dev/null
fi

echo
echo "staging preflight passed"
