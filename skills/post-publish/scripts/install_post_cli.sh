#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SKILL_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
BIN_DIR="$SKILL_DIR/bin"
POST_BIN="$BIN_DIR/post"
VERSION_FILE="$BIN_DIR/post.version"
RELEASE_VERSION="0.1.3"

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

build_asset_url() {
  _os=$(resolve_os)
  _arch=$(resolve_arch)
  _tag_name="v$RELEASE_VERSION"

  if [ -x "$POST_BIN" ] && [ -f "$VERSION_FILE" ] && [ "$(cat "$VERSION_FILE")" = "$_tag_name" ]; then
    verify_binary
    return 0
  fi

  case "$_os" in
    windows) _asset_name="post_${RELEASE_VERSION}_${_os}_${_arch}.zip" ;;
    *) _asset_name="post_${RELEASE_VERSION}_${_os}_${_arch}.tar.gz" ;;
  esac

  printf 'https://github.com/mirtlecn/post-cli/releases/download/%s/%s\n' "$_tag_name" "$_asset_name"
}

install_binary() {
  _asset_url=$(build_asset_url)
  _os=$(resolve_os)
  mkdir -p "$BIN_DIR"
  _temp_dir=$(mktemp -d)
  _archive_name=$(basename "$_asset_url")
  _archive_path="$_temp_dir/$_archive_name"
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
  if [ -x "$POST_BIN" ]; then
    if [ ! -f "$VERSION_FILE" ]; then
      verify_binary
    fi
    exit 0
  fi

  require_command curl

  install_binary
}

main "$@"
