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
AUTO_TEXT_FILE="$TMP_DIR/auto-note.md"
AUTO_UPLOAD_FILE="$TMP_DIR/upload-note.txt"
PUB_FILE="$TMP_DIR/pub.md"
PUB_HEADING_FILE="$TMP_DIR/pub-heading.md"
PUB_AUTO_SLUG_FILE="$TMP_DIR/pub-auto-slug.md"
PUB_INVALID_SLUG_FILE="$TMP_DIR/pub-invalid-slug.md"
CONFIG_FILE="$TMP_DIR/config.json"
PREFIX="smoke-$(date +%s)"
TOPIC_NAME="$PREFIX-topic"
TOPIC_EXPORT_NAME="$PREFIX-topic-export"
TOPIC_VIA_NEW_NAME="$PREFIX-topic-via-new"

printf 'file payload\n' > "$SAMPLE_FILE"
printf '# Auto Text Title\n\nbody\n' > "$AUTO_TEXT_FILE"
printf 'upload body\n' > "$AUTO_UPLOAD_FILE"
cat > "$PUB_FILE" <<EOF
---
title: Smoke Pub Title
slug: $PREFIX-pub
created: 2026-03-01
---

# Ignored Heading
EOF
cat > "$PUB_HEADING_FILE" <<'EOF'

# Heading Pub Title

content
EOF
cat > "$PUB_AUTO_SLUG_FILE" <<'EOF'

# Smoke Auto Slug Title

content
EOF
cat > "$PUB_INVALID_SLUG_FILE" <<'EOF'
---
slug: bad slug ?
---

# Invalid Slug Title
EOF
cat > "$CONFIG_FILE" <<EOF
{"host":"$POST_HOST","token":"$POST_TOKEN"}
EOF

TZ=UTC touch -t 202603060708.09 "$SAMPLE_FILE"
TZ=UTC touch -t 202603070809.10 "$AUTO_TEXT_FILE"
TZ=UTC touch -t 202603080910.11 "$AUTO_UPLOAD_FILE"
TZ=UTC touch -t 202603090910.11 "$PUB_HEADING_FILE"
TZ=UTC touch -t 202603100910.11 "$PUB_AUTO_SLUG_FILE"

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

assert_contains() {
  haystack="$1"
  needle="$2"
  if printf '%s' "$haystack" | grep -F -- "$needle" >/dev/null 2>&1; then
    return 0
  fi

  printf 'FAIL\tassert-contains\texpected output to contain: %s\n' "$needle"
  exit 1
}

run_success "help" ./post help
run_success "version" ./post version
run_success "completion-bash" ./post completion bash
run_success "completion-zsh" ./post completion zsh
run_success "completion-powershell" ./post completion powershell
run_success "ls-all" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls
run_success "topic-new" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic new -i "Smoke Topic" "$TOPIC_NAME"
run_success "topic-new-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic new "$TOPIC_EXPORT_NAME"
run_success "topic-new-via-type" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y --type topic -i "Smoke Via Type" -s "$TOPIC_VIA_NEW_NAME"
run_success "topic-ls-all" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic ls
run_success "topic-ls-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic ls -x "$TOPIC_NAME"
run_success "topic-ls-one" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic ls "$TOPIC_NAME"
run_success "new-text" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-text" "hello text"
run_success "new-write-clipboard" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -w -s "$PREFIX-write-clipboard" "clipboard write text"
run_success "new-combined-uyx" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -uyx -s "$PREFIX-combined-uyx" "combined flags text"
run_success "new-combined-rw" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -yrw -s "$PREFIX-combined-rw" "combined clipboard flags text"
run_success "new-topic-text" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -p "$TOPIC_NAME" -i "Topic Note" -s "$TOPIC_NAME/note" "topic text"
run_success "new-file-content" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-file-content" -f "$SAMPLE_FILE"
new_auto_text_output=$(env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -x -f "$AUTO_TEXT_FILE" 2>&1) || {
  printf 'FAIL\tnew-file-auto-metadata\t%s\n' "$new_auto_text_output"
  exit 1
}
printf 'PASS\tnew-file-auto-metadata\t%s\n' "$new_auto_text_output"
assert_contains "$new_auto_text_output" "\"title\": \"Auto Text Title\""
assert_contains "$new_auto_text_output" "\"created\": \"2026-03-07T08:09:10Z\""
assert_contains "$new_auto_text_output" "\"path\": \"auto-text-title-"
run_success "new-file-upload" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-file-upload" -c file -f "$SAMPLE_FILE"
auto_upload_output=$(env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y -x "$AUTO_UPLOAD_FILE" 2>&1) || {
  printf 'FAIL\tshortcut-file-auto-metadata\t%s\n' "$auto_upload_output"
  exit 1
}
printf 'PASS\tshortcut-file-auto-metadata\t%s\n' "$auto_upload_output"
assert_contains "$auto_upload_output" "\"title\": \"upload-note\""
assert_contains "$auto_upload_output" "\"created\": \"2026-03-08T09:10:11Z\""
assert_contains "$auto_upload_output" "\"path\": \"upload-note-"
run_success "new-topic-file-upload" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y -p "$TOPIC_NAME" -i "Topic File" -s "$TOPIC_NAME/upload" "$SAMPLE_FILE"
run_success "new-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -x -s "$PREFIX-export" "export text"
run_success "new-update-initial" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-update" "before update"
run_success "new-update-overwrite" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -u -s "$PREFIX-update" "after update"
run_success "new-ttl-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -x -s "$PREFIX-ttl" -t 60 "ttl text"
run_success "new-ttl-zero" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -x -s "$PREFIX-ttl-zero" -t 0 "ttl zero text"
run_success "new-type-long-option" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y --type text -s "$PREFIX-type-long" "typed via long option"
run_success "new-created-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -x --created "2026-03-01T08:00:00+08:00" -s "$PREFIX-created" "created text"
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
run_success "shortcut-text-created-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post text -y -x --created "2026-03-01 08:00:00" -s "$PREFIX-shortcut-created" "shortcut created text"
run_success "shortcut-topic-no-ttl" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post text -y -x -p "$TOPIC_NAME" -i "Shortcut Topic Note" -s "$TOPIC_NAME/shortcut-topic" "shortcut topic text"
run_success "shortcut-url" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post url -y -s "$PREFIX-shortcut-url" "https://example.com/shortcut"
run_success "shortcut-file-positional" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y -s "$PREFIX-shortcut-file" "$SAMPLE_FILE"
run_success "shortcut-file-flag" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y -s "$PREFIX-shortcut-file-flag" -f "$SAMPLE_FILE"
run_success "shortcut-file-created-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y -x --created "2026-03-01" -s "$PREFIX-shortcut-file-created" "$SAMPLE_FILE"
run_success "pub-basic" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" POST_PUB_TOPIC="$TOPIC_NAME" ./post pub -y "$PUB_FILE"
run_success "pub-title-from-frontmatter" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls "$TOPIC_NAME/$PREFIX-pub"
run_success "pub-title-from-heading" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" POST_PUB_TOPIC="$TOPIC_NAME" ./post pub -y -s "$PREFIX-pub-heading" "$PUB_HEADING_FILE"
pub_created_auto_output=$(env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls "$TOPIC_NAME/$PREFIX-pub-heading" 2>&1) || {
  printf 'FAIL\tpub-created-auto\t%s\n' "$pub_created_auto_output"
  exit 1
}
printf 'PASS\tpub-created-auto\t%s\n' "$pub_created_auto_output"
assert_contains "$pub_created_auto_output" "\"created\": \"2026-03-09T09:10:11Z\""
auto_slug_output=$(env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" POST_PUB_TOPIC="$TOPIC_NAME" ./post pub -y "$PUB_AUTO_SLUG_FILE" 2>&1) || {
  printf 'FAIL\tpub-auto-slug\t%s\n' "$auto_slug_output"
  exit 1
}
printf 'PASS\tpub-auto-slug\t%s\n' "$auto_slug_output"
assert_contains "$auto_slug_output" "$TOPIC_NAME/smoke-auto-slug-title-"
auto_slug_path=${auto_slug_output#"$POST_HOST/"}
pub_auto_slug_list_output=$(env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls "$auto_slug_path" 2>&1) || {
  printf 'FAIL\tpub-auto-slug-created\t%s\n' "$pub_auto_slug_list_output"
  exit 1
}
printf 'PASS\tpub-auto-slug-created\t%s\n' "$pub_auto_slug_list_output"
assert_contains "$pub_auto_slug_list_output" "\"created\": \"2026-03-10T09:10:11Z\""
run_success "shortcut-ttl-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post text -y -x -s "$PREFIX-shortcut-ttl" "shortcut ttl"
run_success "shortcut-ttl-override" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post text -y -x -t 60 -s "$PREFIX-shortcut-ttl-override" "shortcut ttl override"
run_success "ls-one" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls "$PREFIX-text"
run_success "ls-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls -x "$PREFIX-text"
run_success "export-all" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post export
run_success "export-one" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post export "$PREFIX-update"
run_success "export-topic-item" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post export "$TOPIC_NAME/note"
run_success "topic-refresh" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic refresh -x -i "Smoke Topic Refreshed" "$TOPIC_NAME"
run_success "config-file" env POST_HOST= POST_TOKEN= POST_CONFIG="$CONFIG_FILE" ./post ls "$PREFIX-text"
run_success "rm-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post rm -x "$PREFIX-file-content"
run_success "rm" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post rm "$PREFIX-export"
run_success "topic-rm-export" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic rm -x "$TOPIC_EXPORT_NAME"
run_success "topic-rm" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic rm "$TOPIC_NAME"
run_success "topic-rm-via-new" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic rm "$TOPIC_VIA_NEW_NAME"

run_failure "missing-config" env POST_HOST= POST_TOKEN= POST_CONFIG="$TMP_DIR/not-found-config.json" ./post ls
run_failure "invalid-convert" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -c bad value
run_failure "combined-value-flag" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -uyt 60 text
run_failure "clipboard-read-disabled" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y
run_failure "missing-file-flag" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -c file
run_failure "missing-file" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -f "$TMP_DIR/not-found.txt"
run_failure "missing-rm-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post rm
run_failure "topic-new-missing-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic new
run_failure "topic-refresh-missing-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic refresh
run_failure "topic-rm-missing-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic rm
run_failure "topic-unknown-command" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post topic oops
run_failure "shortcut-file-missing-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y
run_failure "shortcut-file-read-clipboard" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y -r -f "$SAMPLE_FILE"
run_failure "shortcut-file-conflicting-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post file -y -f "$SAMPLE_FILE" "$SAMPLE_FILE"
run_failure "pub-missing-topic" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" POST_PUB_TOPIC= ./post pub -y "$PUB_HEADING_FILE"
run_failure "pub-invalid-slug" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" POST_PUB_TOPIC="$TOPIC_NAME" ./post pub -y "$PUB_INVALID_SLUG_FILE"
run_failure "unknown-command" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post oops
run_failure "unknown-option" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -z text
run_failure "invalid-ttl" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -t nope text
run_failure "negative-ttl" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -t -1 text
run_failure "shortcut-invalid-ttl" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post text -y -t nope text
run_failure "type-convert-mismatch" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y --type text --convert html text
run_failure "topic-missing-title" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -p "$PREFIX-missing-title" "topic text"
run_failure "topic-type-with-content" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y --type topic -s "$PREFIX-topic-type-content" "# hi"
run_failure "topic-path-mismatch" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -p "$PREFIX-topic-a" -i "Mismatch" -s "$PREFIX-topic-b/item" "topic text"
run_failure "topic-not-found" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -p "$PREFIX-missing-topic" -i "Missing Topic" -s "$PREFIX-missing-topic/item" "topic text"
run_failure "duplicate-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post new -y -s "$PREFIX-text" "duplicate text"
run_failure "missing-path" env POST_HOST="$POST_HOST" POST_TOKEN="$POST_TOKEN" ./post ls "$PREFIX-not-found"
