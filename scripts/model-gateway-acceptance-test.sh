#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BASE_URL="${BASE_URL:-http://127.0.0.1:8088}"
USERNAME="model_$(date +%s)"
PASSWORD="ModelGateway123!"

api() {
  local method="$1" endpoint="$2" body="${3:-}"
  local args=(-fsS -X "$method" "$BASE_URL$endpoint" -H 'Content-Type: application/json')
  [[ -n "${TOKEN:-}" ]] && args+=(-H "Authorization: Bearer $TOKEN")
  [[ -n "$body" ]] && args+=(-d "$body")
  curl "${args[@]}"
}

expect_ok() {
  local name="$1" payload="$2"
  [[ "$(jq -r '.code' <<<"$payload")" == "0" ]] || { echo "[FAIL] $name: $payload"; exit 1; }
  echo "[PASS] $name"
}

register="$(api POST /api/auth/register "{\"username\":\"$USERNAME\",\"email\":\"$USERNAME@example.com\",\"password\":\"$PASSWORD\",\"nickname\":\"模型验收\"}")"
expect_ok auth.register "$register"
TOKEN="$(jq -r '.data.access_token // .data.token' <<<"$register")"

providers="$(api GET /api/models/providers)"
expect_ok model.providers "$providers"
[[ "$(jq -r '.data | length' <<<"$providers")" -ge 6 ]] || { echo "[FAIL] providers incomplete: $providers"; exit 1; }

blocked="$(api POST /api/models/configs '{"name":"私网地址","provider_code":"deepseek","base_url":"http://127.0.0.1:1/v1","model":"acceptance-model","api_key":"sk-acceptance-secret","temperature":0.2,"timeout_seconds":5,"max_tokens":512,"enabled":true}' || true)"
[[ "$(jq -r '.code' <<<"$blocked")" != "0" ]] || { echo "[FAIL] private model URL was accepted: $blocked"; exit 1; }
echo "[PASS] model.ssrf_block"

created="$(api POST /api/models/configs '{"name":"验收模型","provider_code":"deepseek","base_url":"https://example.com/v1","model":"acceptance-model","api_key":"sk-acceptance-secret","temperature":0.2,"timeout_seconds":5,"max_tokens":512,"enabled":true}')"
expect_ok model.create "$created"
CONFIG_ID="$(jq -r '.data.id' <<<"$created")"
[[ "$(jq -r '.data.api_key_masked' <<<"$created")" == "********" ]] || { echo "[FAIL] api key not masked: $created"; exit 1; }

configs="$(api GET /api/models/configs)"
expect_ok model.list "$configs"
if grep -q 'sk-acceptance-secret' <<<"$configs"; then echo "[FAIL] plaintext key leaked"; exit 1; fi

tested="$(api POST "/api/models/configs/$CONFIG_ID/test" '{}')"
expect_ok model.test_failure_is_recorded "$tested"
[[ "$(jq -r '.data.success // false' <<<"$tested")" == "false" ]] || { echo "[FAIL] unreachable endpoint unexpectedly succeeded: $tested"; exit 1; }

usage="$(api GET /api/models/usage?days=30)"
expect_ok model.usage "$usage"

deleted="$(api DELETE "/api/models/configs/$CONFIG_ID")"
expect_ok model.delete "$deleted"

echo "Model gateway acceptance passed."
