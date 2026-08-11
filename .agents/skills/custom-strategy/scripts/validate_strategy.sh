#!/usr/bin/env bash
set -euo pipefail

# Validate a portable strategy through the same API used by the frontend.
repo_path="${1:-$(pwd)}"
strategy_file="${2:-}"
symbol="${3:-BTCUSDT}"

if [[ -z "${strategy_file}" ]]; then
  echo "Usage: validate_strategy.sh <repo-path> <strategy-json> [symbol]" >&2
  exit 2
fi

for command_name in jq curl awk; do
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "Missing required command: ${command_name}" >&2
    exit 2
  fi
done

app_conf="${repo_path}/conf/app.conf"
if [[ ! -f "${app_conf}" || ! -f "${strategy_file}" ]]; then
  echo "Missing app.conf or strategy JSON" >&2
  exit 2
fi

jq -e '
  (.name | type == "string") and
  (.technology | type == "object") and
  (.strategy | type == "array") and
  (all(.strategy[];
    (.name | type == "string") and
    (.type | IN("long", "short", "close_long", "close_short")) and
    (.code | type == "string") and
    (.enable | type == "boolean")
  ))
' "${strategy_file}" >/dev/null

read_conf_value() {
  local section="$1"
  local key="$2"
  awk -v target_section="${section}" -v target_key="${key}" '
    function trim(value) {
      gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
      return value
    }
    $0 == "[" target_section "]" { in_section = 1; next }
    /^\[/ { in_section = 0 }
    in_section && index($0, "=") > 0 {
      raw_key = trim(substr($0, 1, index($0, "=") - 1))
      if (raw_key == target_key) {
        value = trim(substr($0, index($0, "=") + 1))
        gsub(/^"|"$/, "", value)
        print value
        exit
      }
    }
  ' "${app_conf}"
}

web_port="$(read_conf_value web port)"
web_username="$(read_conf_value web username)"
web_password="$(read_conf_value web password)"

if [[ -z "${web_port}" || -z "${web_username}" || -z "${web_password}" ]]; then
  echo "Unable to resolve [web] connection values from app.conf" >&2
  exit 2
fi

base_url="http://127.0.0.1:${web_port}"
login_body="$(jq -cn --arg username "${web_username}" --arg password "${web_password}" '{username:$username,password:$password}')"
login_response="$(curl -sS --max-time 10 -H 'Content-Type: application/json' --data-raw "${login_body}" "${base_url}/login")"
auth_token="$(jq -r '.data.token // empty' <<<"${login_response}")"

if [[ -z "${auth_token}" ]]; then
  jq -c '{code,msg}' <<<"${login_response}" >&2
  exit 1
fi

rule_count="$(jq '.strategy | length' "${strategy_file}")"
failed=0

for ((rule_index = 0; rule_index < rule_count; rule_index++)); do
  rule_name="$(jq -r --argjson index "${rule_index}" '.strategy[$index].name' "${strategy_file}")"
  request_body="$(jq -c --argjson index "${rule_index}" '{technology:(.technology|tojson),strategy:([.strategy[$index]]|tojson)}' "${strategy_file}")"
  response="$(curl -sS --max-time 45 -H 'Content-Type: application/json' -H "Authorization: ${auth_token}" --data-raw "${request_body}" "${base_url}/strategy-templates/test/${symbol}")"

  if ! jq -e . >/dev/null 2>&1 <<<"${response}"; then
    printf '{"name":"%s","error":"non-json response"}\n' "${rule_name}"
    failed=1
    continue
  fi

  jq -c --arg name "${rule_name}" '{name:$name,code,msg,data}' <<<"${response}"
  if [[ "$(jq -r '.code // 0' <<<"${response}")" != "200" ]]; then
    failed=1
  fi
done

exit "${failed}"
