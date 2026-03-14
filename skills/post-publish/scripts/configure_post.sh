#!/bin/sh
set -eu

resolve_config_path() {
  if [ -n "${POST_CONFIG:-}" ]; then
    printf '%s\n' "$POST_CONFIG"
    return 0
  fi

  printf '%s/.config/post/config.json\n' "$HOME"
}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

read_config_values() {
  _config_path="$1"
  [ -f "$_config_path" ] || return 0

  python3 - "$_config_path" <<'PY'
import json
import sys

config_path = sys.argv[1]
with open(config_path, "r", encoding="utf-8") as handle:
    payload = json.load(handle)

host = payload.get("host", "")
token = payload.get("token", "")
print(host)
print(token)
PY
}

prompt_value() {
  _label="$1"
  _secret="$2"

  if [ "$_secret" = "true" ] && command -v stty >/dev/null 2>&1; then
    printf '%s: ' "$_label" >&2
    stty -echo
    IFS= read -r _value || true
    stty echo
    printf '\n' >&2
  else
    printf '%s: ' "$_label" >&2
    IFS= read -r _value || true
  fi

  printf '%s\n' "$_value"
}

write_config() {
  _config_path="$1"
  _host="$2"
  _token="$3"

  mkdir -p "$(dirname "$_config_path")"
  python3 - "$_config_path" "$_host" "$_token" <<'PY'
import json
import sys

config_path, host, token = sys.argv[1], sys.argv[2], sys.argv[3]
payload = {"host": host, "token": token}
with open(config_path, "w", encoding="utf-8") as handle:
    json.dump(payload, handle, indent=2)
    handle.write("\n")
PY
}

main() {
  command -v python3 >/dev/null 2>&1 || die "python3 is required"

  _config_path=$(resolve_config_path)
  _env_host=$(printf '%s' "${POST_HOST:-}" | awk '{$1=$1};1')
  _env_token=$(printf '%s' "${POST_TOKEN:-}" | awk '{$1=$1};1')
  _config_host=""
  _config_token=""

  if [ -f "$_config_path" ]; then
    _config_values=$(read_config_values "$_config_path") || die "failed to parse config file $_config_path"
    _config_host=$(printf '%s\n' "$_config_values" | sed -n '1p')
    _config_token=$(printf '%s\n' "$_config_values" | sed -n '2p')
  fi

  _final_host="$_env_host"
  [ -n "$_final_host" ] || _final_host="$_config_host"
  _final_token="$_env_token"
  [ -n "$_final_token" ] || _final_token="$_config_token"

  if [ -z "$_final_host" ]; then
    _final_host=$(prompt_value "POST_HOST" "false")
  fi
  if [ -z "$_final_token" ]; then
    _final_token=$(prompt_value "POST_TOKEN" "true")
  fi

  [ -n "$_final_host" ] || die "POST_HOST is required"
  [ -n "$_final_token" ] || die "POST_TOKEN is required"

  if [ -n "$_env_host" ] && [ -n "$_env_token" ]; then
    printf 'info: using POST_HOST and POST_TOKEN from environment\n' >&2
    return 0
  fi

  if [ "$_final_host" != "$_config_host" ] || [ "$_final_token" != "$_config_token" ] || [ ! -f "$_config_path" ]; then
    write_config "$_config_path" "$_final_host" "$_final_token"
    printf 'info: wrote config to %s\n' "$_config_path" >&2
  fi
}

main "$@"
