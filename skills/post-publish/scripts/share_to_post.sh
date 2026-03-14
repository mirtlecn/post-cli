#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
SKILL_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd)
POST_BIN="$SKILL_DIR/bin/post"

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

random_suffix() {
  LC_ALL=C tr -dc 'a-z0-9' </dev/urandom 2>/dev/null | awk '{ print substr($0, 1, 6); exit }'
}

slugify() {
  _value=$(printf '%s' "$1" | tr '[:upper:]' '[:lower:]')
  _value=$(printf '%s' "$_value" | sed 's#https\{0,1\}://##g')
  _value=$(printf '%s' "$_value" | sed 's/[^a-z0-9]/-/g; s/-\{2,\}/-/g; s/^-//; s/-$//')
  _value=$(printf '%s' "$_value" | cut -c 1-48)
  _value=$(printf '%s' "$_value" | sed 's/-$//')
  printf '%s' "$_value"
}

first_nonempty_line() {
  printf '%s\n' "$1" | sed -n '/[^[:space:]]/ { s/^[[:space:]]*//; s/[[:space:]]*$//; p; q; }'
}

guess_slug_source() {
  _content="$1"
  _convert="$2"

  case "$mode" in
    file)
      if [ "$_convert" = "file" ]; then
        basename "$file"
        return 0
      fi
      ;;
  esac

  if [ "$_convert" = "url" ] || printf '%s' "$_content" | grep -Eq '^https?://'; then
    _url=$(printf '%s' "$_content" | head -n 1 | sed 's/[?#].*$//')
    _host=$(printf '%s' "$_url" | sed -E 's#^https?://([^/]+).*$#\1#; s/^www\.//')
    _path=$(printf '%s' "$_url" | sed -E 's#^https?://[^/]+/?##; s#/$##')
    _last=$(printf '%s' "$_path" | awk -F/ 'NF { print $NF }')
    [ -n "$_host" ] && [ -n "$_last" ] && {
      printf '%s %s' "$_host" "$_last"
      return 0
    }
    [ -n "$_host" ] && {
      printf '%s' "$_host"
      return 0
    }
  fi

  if [ "$_convert" = "md2html" ] || printf '%s\n' "$_content" | grep -Eq '^[[:space:]]*#'; then
    _md_title=$(printf '%s\n' "$_content" | sed -n 's/^[[:space:]]*#\{1,6\}[[:space:]]*//p' | sed -n '/[^[:space:]]/ { p; q; }')
    [ -n "$_md_title" ] && {
      printf '%s' "$_md_title"
      return 0
    }
  fi

  if [ "$_convert" = "html" ] || printf '%s' "$_content" | grep -Eqi '<(html|head|body|title|h1)\b'; then
    _html_title=$(printf '%s' "$_content" | sed -n 's/.*<[Tt][Ii][Tt][Ll][Ee]>\(.*\)<\/[Tt][Ii][Tt][Ll][Ee]>.*/\1/p' | sed -n '/[^[:space:]]/ { p; q; }')
    [ -z "$_html_title" ] && _html_title=$(printf '%s' "$_content" | sed -n 's/.*<[Hh]1[^>]*>\(.*\)<\/[Hh]1>.*/\1/p' | sed -n '/[^[:space:]]/ { p; q; }')
    [ -n "$_html_title" ] && {
      printf '%s' "$_html_title"
      return 0
    }
  fi

  first_nonempty_line "$_content"
}

build_auto_slug() {
  _content="$1"
  _convert="$2"
  _source=$(guess_slug_source "$_content" "$_convert")
  _slug=$(slugify "$_source")
  if [ -n "$_slug" ]; then
    printf '%s' "$_slug"
    return 0
  fi
  random_suffix
}

is_conflict_error() {
  printf '%s' "$1" | grep -Eqi 'already exists|conflict|choose another path|overwrite'
}

infer_convert() {
  _text="$1"

  if printf '%s' "$_text" | grep -Eq '^https?://'; then
    printf 'url'
    return 0
  fi

  if printf '%s\n' "$_text" | grep -Eq '^[[:space:]]*#'; then
    printf 'md2html'
    return 0
  fi

  if printf '%s' "$_text" | grep -Eqi '<(html|head|body|title|h1)\b'; then
    printf 'html'
    return 0
  fi

  printf 'text'
}

run_post() {
  _stdout=$(mktemp) || die "failed to create temp file"
  _stderr=$(mktemp) || die "failed to create temp file"
  if "$@" >"$_stdout" 2>"$_stderr"; then
    RUN_POST_STDOUT=$(cat "$_stdout")
    RUN_POST_STDERR=$(cat "$_stderr")
    rm -f "$_stdout" "$_stderr"
    return 0
  fi
  RUN_POST_STDOUT=$(cat "$_stdout")
  RUN_POST_STDERR=$(cat "$_stderr")
  rm -f "$_stdout" "$_stderr"
  return 1
}

mode=""
text=""
file=""
slug=""
ttl="10080"
convert=""
update=0
export_json=0

while [ $# -gt 0 ]; do
  case "$1" in
    --text)
      [ $# -ge 2 ] || die "--text requires a value"
      mode="text"
      text="$2"
      shift 2
      ;;
    --file)
      [ $# -ge 2 ] || die "--file requires a path"
      mode="file"
      file="$2"
      shift 2
      ;;
    --clipboard)
      mode="clipboard"
      shift
      ;;
    --slug)
      [ $# -ge 2 ] || die "--slug requires a value"
      slug="$2"
      shift 2
      ;;
    --ttl)
      [ $# -ge 2 ] || die "--ttl requires a value in minutes"
      ttl="$2"
      shift 2
      ;;
    --convert)
      [ $# -ge 2 ] || die "--convert requires a value"
      convert="$2"
      shift 2
      ;;
    --update)
      update=1
      shift
      ;;
    --export)
      export_json=1
      shift
      ;;
    *)
      die "unknown option: $1"
      ;;
  esac
done

"$SCRIPT_DIR/configure_post.sh"
"$SCRIPT_DIR/install_post_cli.sh"
[ -x "$POST_BIN" ] || die "post binary is not available at $POST_BIN"

case "$mode" in
  text)
    content="$text"
    ;;
  file)
    [ -f "$file" ] || die "file not found: $file"
    if [ "$convert" = "file" ]; then
      content=$(basename "$file")
    else
      content=$(cat "$file")
    fi
    ;;
  clipboard)
    content=""
    ;;
  *)
    die "one of --text, --file, or --clipboard is required"
    ;;
esac

if [ -z "$convert" ]; then
  case "$mode" in
    file)
      if printf '%s' "$file" | grep -Eqi '\.md$'; then
        convert="md2html"
      else
        convert="text"
      fi
      ;;
    clipboard)
      convert="text"
      ;;
    *)
      convert=$(infer_convert "$content")
      ;;
  esac
fi

if [ "$convert" = "file" ] && [ "$mode" != "file" ]; then
  die "--convert file requires --file <path>"
fi

[ -n "$slug" ] || {
  case "$mode" in
    clipboard) slug=$(random_suffix) ;;
    *) slug=$(build_auto_slug "$content" "$convert") ;;
  esac
}

base_slug="$slug"
attempt=1
max_attempts=5

while :; do
  set -- "$POST_BIN" new -y
  [ -n "$slug" ] && set -- "$@" -s "$slug"
  [ -n "$ttl" ] && set -- "$@" -t "$ttl"
  [ "$update" -eq 1 ] && set -- "$@" -u
  [ "$export_json" -eq 1 ] && set -- "$@" -x
  [ -n "$convert" ] && set -- "$@" -c "$convert"

  rc=0
  case "$mode" in
    text)
      if run_post "$@" "$text"; then
        rc=0
      else
        rc=1
      fi
      ;;
    file)
      if run_post "$@" -f "$file"; then
        rc=0
      else
        rc=1
      fi
      ;;
    clipboard)
      if run_post "$@" -r; then
        rc=0
      else
        rc=1
      fi
      ;;
  esac

  if [ "$rc" -eq 0 ]; then
    if [ "$export_json" -eq 1 ]; then
      printf '%s\n' "$RUN_POST_STDOUT"
    else
      printf '%s\n' "$RUN_POST_STDOUT" | sed -n '/./ { p; q; }'
    fi
    exit 0
  fi

  if [ -n "${slug:-}" ] && [ "$update" -eq 0 ] && [ "$attempt" -lt "$max_attempts" ] && is_conflict_error "$RUN_POST_STDERR"; then
    attempt=$((attempt + 1))
    if [ "$attempt" -le 3 ]; then
      slug="${base_slug}-${attempt}"
    else
      slug="${base_slug}-$(random_suffix)"
    fi
    continue
  fi

  if [ -n "$RUN_POST_STDERR" ]; then
    printf '%s\n' "$RUN_POST_STDERR" >&2
  fi
  break
done

exit 1
