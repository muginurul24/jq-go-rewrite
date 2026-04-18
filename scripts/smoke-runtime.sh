#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

load_dotenv() {
  local dotenv_file="$1"
  local line key value

  while IFS= read -r line || [[ -n "${line}" ]]; do
    [[ -z "${line}" || "${line}" =~ ^[[:space:]]*# ]] && continue
    key="${line%%=*}"
    value="${line#*=}"
    key="${key#"${key%%[![:space:]]*}"}"
    key="${key%"${key##*[![:space:]]}"}"
    value="${value%$'\r'}"
    if [[ -z "${!key+x}" ]]; then
      export "${key}=${value}"
    fi
  done < "${dotenv_file}"
}

if [[ -f "${ROOT_DIR}/.env" ]]; then
  load_dotenv "${ROOT_DIR}/.env"
fi

BASE_URL="${SMOKE_BASE_URL:-${APP_URL:-http://localhost:8080}}"
BASE_URL="${BASE_URL%/}"
SMOKE_TIMEOUT="${SMOKE_TIMEOUT:-20}"
SMOKE_USERNAME="${SMOKE_USERNAME:-justqiu}"
SMOKE_PASSWORD="${SMOKE_PASSWORD:-justqiu}"
SMOKE_TOKO_TOKEN="${SMOKE_TOKO_TOKEN:-}"
NEXUSGGR_READY=false

if [[ -n "${NEXUSGGR_BASE_URL:-}" && -n "${NEXUSGGR_AGENT_CODE:-}" && -n "${NEXUSGGR_AGENT_TOKEN:-}" ]]; then
  NEXUSGGR_READY=true
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required to run smoke-runtime.sh" >&2
  exit 1
fi

if ! command -v node >/dev/null 2>&1; then
  echo "node is required to run smoke-runtime.sh" >&2
  exit 1
fi

TMP_DIR="$(mktemp -d)"
COOKIE_JAR="${TMP_DIR}/cookies.txt"
trap 'rm -rf "${TMP_DIR}"' EXIT

json_read() {
  local file="$1"
  local path="$2"

  node -e '
    const fs = require("fs")
    const file = process.argv[1]
    const path = process.argv[2]
    const data = JSON.parse(fs.readFileSync(file, "utf8"))
    let current = data
    for (const key of path.split(".")) {
      if (!key) continue
      if (current == null || !(key in current)) process.exit(2)
      current = current[key]
    }
    if (typeof current === "object") {
      process.stdout.write(JSON.stringify(current))
      process.exit(0)
    }
    process.stdout.write(String(current))
  ' "${file}" "${path}"
}

request_json() {
  local method="$1"
  local url="$2"
  local body="$3"
  local output_file="$4"
  shift 4

  local status_code
  local curl_args=(
    --silent
    --show-error
    --location
    --connect-timeout
    "${SMOKE_TIMEOUT}"
    --max-time
    "${SMOKE_TIMEOUT}"
    --cookie
    "${COOKIE_JAR}"
    --cookie-jar
    "${COOKIE_JAR}"
    --output
    "${output_file}"
    --write-out
    "%{http_code}"
    --request
    "${method}"
  )

  if [[ "${method}" != "GET" && "${method}" != "HEAD" && "${method}" != "OPTIONS" ]]; then
    curl_args+=(
      -H
      "Origin: ${BASE_URL}"
      -H
      "Referer: ${BASE_URL}/"
    )
  fi

  while (($#)); do
    curl_args+=("$1")
    shift
  done

  if [[ -n "${body}" ]]; then
    curl_args+=(--data "${body}")
  fi

  status_code="$(curl "${curl_args[@]}" "${url}")"

  if [[ "${status_code}" != 2* ]]; then
    echo "request failed: ${method} ${url} -> ${status_code}" >&2
    if [[ -f "${output_file}" ]]; then
      cat "${output_file}" >&2
      echo >&2
    fi
    return 1
  fi
}

log_step() {
  printf '\n[%s] %s\n' "$(date '+%H:%M:%S')" "$1"
}

BOOTSTRAP_JSON="${TMP_DIR}/bootstrap.json"
LOGIN_JSON="${TMP_DIR}/login.json"
ME_JSON="${TMP_DIR}/me.json"
OVERVIEW_JSON="${TMP_DIR}/overview.json"
PULSE_JSON="${TMP_DIR}/pulse.json"
TRANSACTIONS_JSON="${TMP_DIR}/transactions.json"
TOPUP_JSON="${TMP_DIR}/topup.json"
WITHDRAWAL_JSON="${TMP_DIR}/withdrawal.json"
CALL_JSON="${TMP_DIR}/call.json"
BALANCE_JSON="${TMP_DIR}/balance.json"
MERCHANT_ACTIVE_JSON="${TMP_DIR}/merchant-active.json"
PROVIDERS_JSON="${TMP_DIR}/providers.json"
HEALTH_LIVE_JSON="${TMP_DIR}/health-live.json"
HEALTH_READY_JSON="${TMP_DIR}/health-ready.json"
LOGOUT_JSON="${TMP_DIR}/logout.json"

log_step "Checking health endpoints"
request_json GET "${BASE_URL}/health/live" "" "${HEALTH_LIVE_JSON}"
request_json GET "${BASE_URL}/health/ready" "" "${HEALTH_READY_JSON}"

log_step "Loading CSRF bootstrap"
request_json GET "${BASE_URL}/backoffice/api/auth/bootstrap" "" "${BOOTSTRAP_JSON}"
CSRF_TOKEN="$(json_read "${BOOTSTRAP_JSON}" "csrfToken")"

log_step "Logging in as smoke operator"
request_json POST "${BASE_URL}/backoffice/api/auth/login" \
  "{\"login\":\"${SMOKE_USERNAME}\",\"password\":\"${SMOKE_PASSWORD}\",\"remember\":false}" \
  "${LOGIN_JSON}" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: ${CSRF_TOKEN}"

if node -e 'const fs=require("fs"); const payload=JSON.parse(fs.readFileSync(process.argv[1],"utf8")); process.exit(payload.requiresMfa ? 0 : 1)' "${LOGIN_JSON}"; then
  echo "smoke user requires MFA; use a non-MFA account for smoke-runtime.sh" >&2
  exit 1
fi

CSRF_TOKEN="$(json_read "${LOGIN_JSON}" "csrfToken")"

log_step "Checking authenticated backoffice endpoints"
request_json GET "${BASE_URL}/backoffice/api/auth/me" "" "${ME_JSON}"
request_json GET "${BASE_URL}/backoffice/api/dashboard/overview" "" "${OVERVIEW_JSON}"
request_json GET "${BASE_URL}/backoffice/api/dashboard/operational-pulse" "" "${PULSE_JSON}"
request_json GET "${BASE_URL}/backoffice/api/transactions?page=1&per_page=10" "" "${TRANSACTIONS_JSON}"
request_json GET "${BASE_URL}/backoffice/api/nexusggr-topup/bootstrap" "" "${TOPUP_JSON}"
request_json GET "${BASE_URL}/backoffice/api/withdrawal/bootstrap" "" "${WITHDRAWAL_JSON}"
request_json GET "${BASE_URL}/backoffice/api/call-management/bootstrap" "" "${CALL_JSON}"

if [[ -n "${SMOKE_TOKO_TOKEN}" ]]; then
  log_step "Checking bearer-auth public compatibility endpoints"
  request_json GET "${BASE_URL}/api/v1/balance" "" "${BALANCE_JSON}" \
    -H "Authorization: Bearer ${SMOKE_TOKO_TOKEN}"

  request_json POST "${BASE_URL}/api/v1/merchant-active" "{\"label\":\"runtime-smoke\"}" "${MERCHANT_ACTIVE_JSON}" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${SMOKE_TOKO_TOKEN}"

  if [[ "${NEXUSGGR_READY}" == "true" ]]; then
    request_json GET "${BASE_URL}/api/v1/providers" "" "${PROVIDERS_JSON}" \
      -H "Authorization: Bearer ${SMOKE_TOKO_TOKEN}"
  else
    log_step "Skipping provider-backed public smoke because NexusGGR env is incomplete"
  fi
else
  log_step "Skipping bearer-auth public API smoke because SMOKE_TOKO_TOKEN is empty"
fi

log_step "Logging out"
request_json POST "${BASE_URL}/backoffice/api/auth/logout" "{}" "${LOGOUT_JSON}" \
  -H "Content-Type: application/json" \
  -H "X-CSRF-Token: ${CSRF_TOKEN}"

echo
echo "runtime smoke passed"
echo "base_url=${BASE_URL}"
echo "smoke_user=$(json_read "${ME_JSON}" "user.username")"
echo "overview_role=$(json_read "${OVERVIEW_JSON}" "data.role")"
echo "recent_transactions=$(node -e 'const fs=require("fs"); const data=JSON.parse(fs.readFileSync(process.argv[1],"utf8")); process.stdout.write(String((data.data?.recentTransactions ?? []).length))' "${OVERVIEW_JSON}")"
echo "transaction_rows=$(node -e 'const fs=require("fs"); const data=JSON.parse(fs.readFileSync(process.argv[1],"utf8")); process.stdout.write(String((data.data ?? []).length))' "${TRANSACTIONS_JSON}")"
if [[ -n "${SMOKE_TOKO_TOKEN}" ]]; then
  echo "public_pending_balance=$(json_read "${BALANCE_JSON}" "pending_balance")"
  echo "merchant_active_store=$(json_read "${MERCHANT_ACTIVE_JSON}" "store.name")"
fi
