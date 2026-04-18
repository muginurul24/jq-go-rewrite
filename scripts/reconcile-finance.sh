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

: "${DB_HOST:?DB_HOST is required}"
: "${DB_PORT:=5432}"
: "${DB_DATABASE:?DB_DATABASE is required}"
: "${DB_USERNAME:?DB_USERNAME is required}"
: "${DB_PASSWORD:?DB_PASSWORD is required}"

if ! command -v psql >/dev/null 2>&1; then
  echo "psql is required to run reconcile-finance.sh" >&2
  exit 1
fi

export PGPASSWORD="${DB_PASSWORD}"

psql_cmd=(
  psql
  -X
  -v
  ON_ERROR_STOP=1
  -P
  pager=off
  -h
  "${DB_HOST}"
  -p
  "${DB_PORT}"
  -U
  "${DB_USERNAME}"
  -d
  "${DB_DATABASE}"
)

run_query() {
  local title="$1"
  local query="$2"

  printf '\n== %s ==\n' "${title}"
  "${psql_cmd[@]}" -c "${query}"
}

echo "financial reconciliation snapshot"
echo "database=${DB_DATABASE} host=${DB_HOST}:${DB_PORT}"
echo "generated_at=$(date '+%Y-%m-%d %H:%M:%S %Z')"

run_query "Core Counts" "
SELECT 'users_total' AS metric, COUNT(*)::BIGINT AS value FROM users
UNION ALL
SELECT 'tokos_total', COUNT(*)::BIGINT FROM tokos WHERE deleted_at IS NULL
UNION ALL
SELECT 'banks_total', COUNT(*)::BIGINT FROM banks WHERE deleted_at IS NULL
UNION ALL
SELECT 'players_total', COUNT(*)::BIGINT FROM players WHERE deleted_at IS NULL
UNION ALL
SELECT 'transactions_total', COUNT(*)::BIGINT FROM transactions WHERE deleted_at IS NULL
UNION ALL
SELECT 'notifications_total', COUNT(*)::BIGINT FROM notifications
ORDER BY metric;
"

run_query "User Roles" "
SELECT role, is_active, COUNT(*)::BIGINT AS total
FROM users
GROUP BY role, is_active
ORDER BY role, is_active DESC;
"

run_query "Toko Coverage" "
SELECT
  t.id,
  t.name,
  u.username AS owner_username,
  t.is_active,
  COALESCE(b.pending, 0) AS pending_balance,
  COALESCE(b.settle, 0) AS settle_balance,
  COALESCE(b.nexusggr, 0) AS nexusggr_balance
FROM tokos t
JOIN users u ON u.id = t.user_id
LEFT JOIN balances b ON b.toko_id = t.id
WHERE t.deleted_at IS NULL
ORDER BY t.id;
"

run_query "Balance Totals" "
SELECT
  COALESCE(SUM(pending), 0) AS total_pending,
  COALESCE(SUM(settle), 0) AS total_settle,
  COALESCE(SUM(nexusggr), 0) AS total_nexusggr
FROM balances;
"

run_query "Income Snapshot" "
SELECT id, ggr, fee_transaction, fee_withdrawal, amount, created_at, updated_at
FROM incomes
ORDER BY id;
"

run_query "Transaction Totals by Category, Type, Status" "
SELECT
  category,
  type,
  status,
  COUNT(*)::BIGINT AS total_records,
  COALESCE(SUM(amount), 0) AS total_amount
FROM transactions
WHERE deleted_at IS NULL
GROUP BY category, type, status
ORDER BY category, type, status;
"

run_query "Pending Transactions Older Than 30 Minutes" "
SELECT
  id,
  toko_id,
  code,
  amount,
  created_at
FROM transactions
WHERE deleted_at IS NULL
  AND status = 'pending'
  AND created_at <= (CURRENT_TIMESTAMP - INTERVAL '30 minutes')
ORDER BY created_at ASC;
"

run_query "Consistency Checks" "
SELECT 'tokos_without_balance' AS check_name, COUNT(*)::BIGINT AS total
FROM tokos t
LEFT JOIN balances b ON b.toko_id = t.id
WHERE t.deleted_at IS NULL
  AND b.id IS NULL
UNION ALL
SELECT 'players_without_toko', COUNT(*)::BIGINT
FROM players p
LEFT JOIN tokos t ON t.id = p.toko_id
WHERE p.deleted_at IS NULL
  AND t.id IS NULL
UNION ALL
SELECT 'transactions_without_toko', COUNT(*)::BIGINT
FROM transactions trx
LEFT JOIN tokos t ON t.id = trx.toko_id
WHERE trx.deleted_at IS NULL
  AND t.id IS NULL
UNION ALL
SELECT 'duplicate_balance_rows_per_toko', COUNT(*)::BIGINT
FROM (
  SELECT toko_id
  FROM balances
  GROUP BY toko_id
  HAVING COUNT(*) > 1
) dup;
"
