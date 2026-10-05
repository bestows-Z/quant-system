#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
USERNAME="portfolio_$(date +%s)"
PASSWORD="Portfolio123!"

api() {
  local method="$1" path="$2" body="${3:-}"
  if [[ -n "${TOKEN:-}" ]]; then
    curl -fsS -X "$method" "$BASE_URL$path" -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" ${body:+-d "$body"}
  else
    curl -fsS -X "$method" "$BASE_URL$path" -H "Content-Type: application/json" ${body:+-d "$body"}
  fi
}

expect_ok() {
  local label="$1" payload="$2"
  if [[ "$(jq -r '.code' <<<"$payload")" != "0" ]]; then
    echo "[FAIL] $label: $payload" >&2
    exit 1
  fi
  echo "[PASS] $label"
}

register="$(api POST /api/auth/register "{\"username\":\"$USERNAME\",\"email\":\"$USERNAME@example.com\",\"password\":\"$PASSWORD\",\"nickname\":\"持仓验收\"}")"
expect_ok auth.register "$register"
login="$(api POST /api/auth/login "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")"
expect_ok auth.login "$login"
TOKEN="$(jq -r '.data.access_token // .data.token' <<<"$login")"

created="$(api POST /api/portfolios '{"name":"真实持仓验收","description":"自动化验收组合"}')"
expect_ok portfolio.create "$created"
PORTFOLIO_ID="$(jq -r '.data.id' <<<"$created")"

position="$(api POST "/api/portfolios/$PORTFOLIO_ID/positions" '{"symbol":"600176","name":"中国巨石","quantity":1000,"available_quantity":1000,"avg_cost":"46.80","opened_on":"2026-07-01","note":"核心观察仓"}')"
expect_ok portfolio.position_add "$position"

preview="$(curl -fsS -X POST "$BASE_URL/api/portfolios/$PORTFOLIO_ID/import/preview" -H "Authorization: Bearer $TOKEN" -H "Idempotency-Key: portfolio-fixture-v1" -F "file=@$ROOT_DIR/scripts/fixtures/real_positions.csv;type=text/csv")"
expect_ok portfolio.import_preview "$preview"
TASK_ID="$(jq -r '.data.task_id' <<<"$preview")"
[[ "$(jq -r '.data.invalid_rows // 0' <<<"$preview")" == "0" ]] || { echo "[FAIL] preview contains errors: $preview"; exit 1; }

confirmed="$(api POST "/api/portfolios/$PORTFOLIO_ID/import/confirm" "{\"task_id\":$TASK_ID}")"
expect_ok portfolio.import_confirm "$confirmed"
replayed="$(api POST "/api/portfolios/$PORTFOLIO_ID/import/confirm" "{\"task_id\":$TASK_ID}")"
expect_ok portfolio.import_idempotent "$replayed"
[[ "$(jq -r '.data.idempotent_replay' <<<"$replayed")" == "true" ]] || { echo "[FAIL] confirm was not idempotent: $replayed"; exit 1; }

positions="$(api GET "/api/portfolios/$PORTFOLIO_ID/positions")"
expect_ok portfolio.positions "$positions"
[[ "$(jq '.data | length' <<<"$positions")" == "2" ]] || { echo "[FAIL] expected 2 positions: $positions"; exit 1; }

analysis="$(api POST "/api/portfolios/$PORTFOLIO_ID/analyze" '{}')"
expect_ok portfolio.analysis "$analysis"
[[ "$(jq -r '.data.analysis_version' <<<"$analysis")" == "portfolio-v2" ]] || { echo "[FAIL] analysis version missing: $analysis"; exit 1; }
advanced="$(jq -r '.data.advanced_json | fromjson' <<<"$analysis")"
[[ "$(jq -r '.scenarios | length' <<<"$advanced")" -eq 4 ]] || { echo "[FAIL] scenario analysis missing: $analysis"; exit 1; }
[[ "$(jq -r 'has("sharpe_ratio") and has("var_95") and has("correlations") and has("contributions")' <<<"$advanced")" == "true" ]] || { echo "[FAIL] advanced metrics missing: $analysis"; exit 1; }
awk "BEGIN { exit !($(jq -r '.data.coverage_rate // 0' <<<"$analysis") > 0) }" || { echo "[FAIL] analysis did not use realtime quotes: $analysis"; exit 1; }
[[ -n "$(jq -r '.data.data_source // empty' <<<"$analysis")" ]] || { echo "[FAIL] analysis data source missing: $analysis"; exit 1; }
[[ "$(jq -r '.data.disclaimer' <<<"$analysis")" == *"不构成投资建议"* ]] || { echo "[FAIL] disclaimer missing: $analysis"; exit 1; }

deleted="$(api DELETE "/api/portfolios/$PORTFOLIO_ID")"
expect_ok portfolio.delete "$deleted"

echo "PORTFOLIO_ACCEPTANCE_PASSED"
