#!/bin/sh
set -eu

ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
POST_HOST=${POST_HOST:-}
POST_TOKEN=${POST_TOKEN:-}

if [ -z "$POST_HOST" ] || [ -z "$POST_TOKEN" ]; then
  printf 'error: POST_HOST and POST_TOKEN must be set for smoke tests.\n' >&2
  exit 1
fi

cd "$ROOT_DIR"
go build -o post ./cmd/post

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

SAMPLE_FILE="$TMP_DIR/sample.txt"
CONFIG_FILE="$TMP_DIR/config.json"
PREFIX="smoke-$(date +%s)"

printf 'file payload\n' > "$SAMPLE_FILE"
cat > "$CONFIG_FILE" <<EOF
{"host":"$POST_HOST","token":"$POST_TOKEN"}
EOF

run_success() {
  name="$1"
  shift
  if output=$("$@" 2>&1); then
    printf 'PASS\t%s\t%s\n' "$name" "$output"
  else
    printf 'FAIL\t%s\t%s\n' "$name" "$output"
    exit 1
  fi
}

run_failure() {
  name="$1"
  shift
  if output=$("$@" 2>&1); then
    printf 'FAIL\t%s\tunexpected success: %s\n' "$name" "$output"
    exit 1
  else
    printf 'PASS\t%s\t%s\n' "$name" "$output"
  fi
}

run_success "help" ./post help
run_success "ls-all" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls
run_success "new-text" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-text" "hello text"
run_success "new-file-content" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-file-content" -f "$SAMPLE_FILE"
run_success "new-file-upload" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-file-upload" -c file -f "$SAMPLE_FILE"
run_success "new-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -x -s "$PREFIX-export" "export text"
run_success "new-update-initial" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-update" "before update"
run_success "new-update-overwrite" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -u -s "$PREFIX-update" "after update"
run_success "new-ttl-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -x -s "$PREFIX-ttl" -t 60 "ttl text"
run_success "new-type-text" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-type-text" -c text "typed text"
run_success "new-type-html" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-type-html" -c html "<h1>Hello</h1>"
run_success "new-type-url" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-type-url" -c url "https://example.com"
run_success "new-md2html" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-md2html" -c md2html "# Hello"
run_success "new-qrcode" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-qrcode" -c qrcode "https://example.com/qr"
run_success "ls-one" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls "$PREFIX-text"
run_success "ls-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls -x "$PREFIX-text"
run_success "export-all" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post export
run_success "export-one" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post export "$PREFIX-update"
run_success "config-file" env POST_HOST= POST_TOKEN= POST_CONFIG="$CONFIG_FILE" ./post ls "$PREFIX-text"
run_success "rm-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post rm -x "$PREFIX-file-content"
run_success "rm" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post rm "$PREFIX-export"

run_failure "missing-config" env POST_HOST= POST_TOKEN= ./post ls
run_failure "invalid-convert" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -c bad value
run_failure "missing-file-flag" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -c file
run_failure "missing-file" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -f "$TMP_DIR/not-found.txt"
run_failure "missing-rm-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post rm
run_failure "unknown-command" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post oops
run_failure "unknown-option" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -z text
run_failure "invalid-ttl" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -t nope text
run_failure "duplicate-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-text" "duplicate text"
run_failure "missing-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls "$PREFIX-not-found"
