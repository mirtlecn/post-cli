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
TOPIC_NAME="$PREFIX-topic"
TOPIC_EXPORT_NAME="$PREFIX-topic-export"

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
run_success "version" ./post version
run_success "completion-bash" ./post completion bash
run_success "completion-zsh" ./post completion zsh
run_success "completion-powershell" ./post completion powershell
run_success "ls-all" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls
run_success "topic-new" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic new "$TOPIC_NAME"
run_success "topic-new-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic new "$TOPIC_EXPORT_NAME"
run_success "topic-ls-all" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic ls
run_success "topic-ls-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic ls -x "$TOPIC_NAME"
run_success "topic-ls-one" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic ls "$TOPIC_NAME"
run_success "new-text" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-text" "hello text"
run_success "new-write-clipboard" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -w -s "$PREFIX-write-clipboard" "clipboard write text"
run_success "new-combined-uyx" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -uyx -s "$PREFIX-combined-uyx" "combined flags text"
run_success "new-combined-rw" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -yrw -s "$PREFIX-combined-rw" "combined clipboard flags text"
run_success "new-topic-text" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -p "$TOPIC_NAME" -i "Topic Note" -s "$TOPIC_NAME/note" "topic text"
run_success "new-file-content" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-file-content" -f "$SAMPLE_FILE"
run_success "new-file-upload" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-file-upload" -c file -f "$SAMPLE_FILE"
run_success "new-topic-file-upload" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y -p "$TOPIC_NAME" -i "Topic File" -s "$TOPIC_NAME/upload" "$SAMPLE_FILE"
run_success "new-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -x -s "$PREFIX-export" "export text"
run_success "new-update-initial" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-update" "before update"
run_success "new-update-overwrite" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -u -s "$PREFIX-update" "after update"
run_success "new-ttl-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -x -s "$PREFIX-ttl" -t 60 "ttl text"
run_success "new-ttl-zero" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -x -s "$PREFIX-ttl-zero" -t 0 "ttl zero text"
run_success "new-type-long-option" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y --type text -s "$PREFIX-type-long" "typed via long option"
run_success "new-type-convert-match" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y --type text --convert text -s "$PREFIX-type-match" "type and convert match"
run_success "new-type-text" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-type-text" -c text "typed text"
run_success "new-type-html" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-type-html" -c html "<h1>Hello</h1>"
run_success "new-type-url" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-type-url" -c url "https://example.com"
run_success "new-md2html" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-md2html" -c md2html "# Hello"
run_success "new-qrcode" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-qrcode" -c qrcode "https://example.com/qr"
run_success "shortcut-md-text" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post md -y -s "$PREFIX-shortcut-md" "# Shortcut"
run_success "shortcut-md-file" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post md -y -s "$PREFIX-shortcut-md-file" -f "$SAMPLE_FILE"
run_success "shortcut-qr" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post qr -y -s "$PREFIX-shortcut-qr" "https://example.com/qr-shortcut"
run_success "shortcut-html" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post html -y -s "$PREFIX-shortcut-html" "<p>hi</p>"
run_success "shortcut-text" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post text -y -s "$PREFIX-shortcut-text" "shortcut text"
run_success "shortcut-url" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post url -y -s "$PREFIX-shortcut-url" "https://example.com/shortcut"
run_success "shortcut-file-positional" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y -s "$PREFIX-shortcut-file" "$SAMPLE_FILE"
run_success "shortcut-file-flag" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y -s "$PREFIX-shortcut-file-flag" -f "$SAMPLE_FILE"
run_success "shortcut-ttl-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post text -y -x -s "$PREFIX-shortcut-ttl" "shortcut ttl"
run_success "shortcut-ttl-override" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post text -y -x -t 60 -s "$PREFIX-shortcut-ttl-override" "shortcut ttl override"
run_success "ls-one" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls "$PREFIX-text"
run_success "ls-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls -x "$PREFIX-text"
run_success "export-all" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post export
run_success "export-one" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post export "$PREFIX-update"
run_success "export-topic-item" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post export "$TOPIC_NAME/note"
run_success "config-file" env POST_HOST= POST_TOKEN= POST_CONFIG="$CONFIG_FILE" ./post ls "$PREFIX-text"
run_success "rm-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post rm -x "$PREFIX-file-content"
run_success "rm" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post rm "$PREFIX-export"
run_success "topic-rm-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic rm -x "$TOPIC_EXPORT_NAME"
run_success "topic-rm" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic rm "$TOPIC_NAME"

run_failure "missing-config" env POST_HOST= POST_TOKEN= POST_CONFIG="$TMP_DIR/not-found-config.json" ./post ls
run_failure "invalid-convert" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -c bad value
run_failure "combined-value-flag" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -uyt 60 text
run_failure "clipboard-read-disabled" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y
run_failure "missing-file-flag" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -c file
run_failure "missing-file" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -f "$TMP_DIR/not-found.txt"
run_failure "missing-rm-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post rm
run_failure "topic-new-missing-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic new
run_failure "topic-rm-missing-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic rm
run_failure "topic-unknown-command" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic oops
run_failure "shortcut-file-missing-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y
run_failure "shortcut-file-read-clipboard" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y -r -f "$SAMPLE_FILE"
run_failure "shortcut-file-conflicting-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y -f "$SAMPLE_FILE" "$SAMPLE_FILE"
run_failure "unknown-command" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post oops
run_failure "unknown-option" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -z text
run_failure "invalid-ttl" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -t nope text
run_failure "negative-ttl" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -t -1 text
run_failure "shortcut-invalid-ttl" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post text -y -t nope text
run_failure "type-convert-mismatch" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y --type text --convert html text
run_failure "topic-missing-title" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -p "$PREFIX-missing-title" "topic text"
run_failure "topic-path-mismatch" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -p "$PREFIX-topic-a" -i "Mismatch" -s "$PREFIX-topic-b/item" "topic text"
run_failure "topic-not-found" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -p "$PREFIX-missing-topic" -i "Missing Topic" -s "$PREFIX-missing-topic/item" "topic text"
run_failure "duplicate-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-text" "duplicate text"
run_failure "missing-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls "$PREFIX-not-found"
