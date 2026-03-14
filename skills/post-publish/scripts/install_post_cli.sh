#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SKILL_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
BIN_DIR="$SKILL_DIR/bin"
POST_BIN="$BIN_DIR/post"
VERSION_FILE="$BIN_DIR/post.version"
LATEST_RELEASE_URL="https://api.github.com/repos/mirtlecn/post-cli/releases/latest"

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "$1 is required"
}

resolve_os() {
  case "$(uname -s)" in
    Darwin) printf 'darwin' ;;
    Linux) printf 'linux' ;;
    MINGW*|MSYS*|CYGWIN*) printf 'windows' ;;
    *) die "unsupported operating system: $(uname -s)" ;;
  esac
}

resolve_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64' ;;
    arm64|aarch64) printf 'arm64' ;;
    *) die "unsupported architecture: $(uname -m)" ;;
  esac
}

release_json_field() {
  _json_path="$1"
  python3 -c '
import json
import sys

payload = json.load(sys.stdin)
path = sys.argv[1]
value = payload
for part in path.split("."):
    value = value[part]
if not isinstance(value, str):
    raise SystemExit(f"expected string at {path}")
print(value)
' "$_json_path"
}

select_asset_url() {
  _asset_name="$1"
  python3 -c '
import json
import sys

asset_name = sys.argv[1]
payload = json.load(sys.stdin)
for asset in payload.get("assets", []):
    if asset.get("name") == asset_name:
        print(asset["browser_download_url"])
        raise SystemExit(0)
raise SystemExit(f"asset not found: {asset_name}")
' "$_asset_name"
}

remove_quarantine_if_needed() {
  if [ "$(resolve_os)" != "darwin" ]; then
    return 0
  fi

  if command -v xattr >/dev/null 2>&1; then
    xattr -d com.apple.quarantine "$POST_BIN" >/dev/null 2>&1 || true
  fi
}

verify_binary() {
  _output=$("$POST_BIN" version 2>&1) || {
    case "$(resolve_os)" in
      darwin)
        die "post-cli is installed but macOS blocked it. Open System Settings > Privacy & Security and allow the binary, then retry. Original error: $_output"
        ;;
      *)
        die "post-cli verification failed: $_output"
        ;;
    esac
  }

  _reported_version=$(printf '%s\n' "$_output" | awk 'NR==1 { print $2 }')
  case "$_reported_version" in
    v*) ;;
    *) _reported_version="v$_reported_version" ;;
  esac
  printf '%s\n' "$_reported_version" >"$VERSION_FILE"
}

download_release_json() {
  curl -fsSL "$LATEST_RELEASE_URL"
}

download_archive() {
  _url="$1"
  _destination="$2"
  curl -fsSL "$_url" -o "$_destination"
}

extract_archive() {
  _archive_path="$1"
  _os="$2"
  _extract_dir="$3"

  mkdir -p "$_extract_dir"

  case "$_os" in
    windows)
      require_command unzip
      unzip -q "$_archive_path" -d "$_extract_dir"
      ;;
    *)
      require_command tar
      tar -xzf "$_archive_path" -C "$_extract_dir"
      ;;
  esac
}

install_binary() {
  _release_json="$1"
  _os=$(resolve_os)
  _arch=$(resolve_arch)
  _tag_name=$(printf '%s' "$_release_json" | release_json_field tag_name)
  _version=${_tag_name#v}

  if [ -x "$POST_BIN" ] && [ -f "$VERSION_FILE" ] && [ "$(cat "$VERSION_FILE")" = "$_tag_name" ]; then
    verify_binary
    return 0
  fi

  case "$_os" in
    windows) _asset_name="post_${_version}_${_os}_${_arch}.zip" ;;
    *) _asset_name="post_${_version}_${_os}_${_arch}.tar.gz" ;;
  esac

  _asset_url=$(printf '%s' "$_release_json" | select_asset_url "$_asset_name") || die "failed to find release asset $_asset_name"
  mkdir -p "$BIN_DIR"
  _temp_dir=$(mktemp -d)
  _archive_path="$_temp_dir/$_asset_name"
  _extract_dir="$_temp_dir/extract"
  trap 'rm -rf "$_temp_dir"' EXIT INT TERM

  download_archive "$_asset_url" "$_archive_path" || die "failed to download $_asset_url"
  extract_archive "$_archive_path" "$_os" "$_extract_dir"

  _binary_name="post"
  [ "$_os" = "windows" ] && _binary_name="post.exe"
  [ -f "$_extract_dir/$_binary_name" ] || die "release archive did not contain $_binary_name"

  cp "$_extract_dir/$_binary_name" "$POST_BIN"
  chmod +x "$POST_BIN"
  remove_quarantine_if_needed
  verify_binary
}

main() {
  require_command curl
  require_command python3

  _release_json=$(download_release_json) || die "failed to fetch latest release metadata from $LATEST_RELEASE_URL"
  install_binary "$_release_json"
}

main "$@"
