#!/bin/sh
set -eu

REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
SKILL_ROOT="$REPO_ROOT/skills/post-publish"

die() {
  printf 'test failed: %s\n' "$*" >&2
  exit 1
}

assert_file_contains() {
  _file_path="$1"
  _expected="$2"
  grep -F -- "$_expected" "$_file_path" >/dev/null 2>&1 || die "expected $_file_path to contain $_expected"
}

assert_equals() {
  _expected="$1"
  _actual="$2"
  [ "$_expected" = "$_actual" ] || die "expected '$(_escape "$_expected")' but got '$(_escape "$_actual")'"
}

_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/'"'"'/\\'"'"'/g'
}

make_temp_skill_copy() {
  _target_dir=$(mktemp -d)
  cp -R "$SKILL_ROOT/." "$_target_dir/"
  printf '%s\n' "$_target_dir"
}

test_configure_uses_environment_without_writing() {
  _home_dir=$(mktemp -d)
  HOME="$_home_dir" POST_HOST="https://env.example" POST_TOKEN="env-token" \
    "$SKILL_ROOT/scripts/configure_post.sh" >/tmp/post-publish-config-env.out 2>/tmp/post-publish-config-env.err

  [ ! -f "$_home_dir/.config/post/config.json" ] || die "configure_post.sh should not write config when both env vars exist"
  assert_file_contains /tmp/post-publish-config-env.err "using POST_HOST and POST_TOKEN from environment"
}

test_configure_writes_missing_config() {
  _home_dir=$(mktemp -d)
  printf 'https://config.example\nconfig-token\n' | HOME="$_home_dir" "$SKILL_ROOT/scripts/configure_post.sh" >/tmp/post-publish-config-write.out 2>/tmp/post-publish-config-write.err

  _config_path="$_home_dir/.config/post/config.json"
  [ -f "$_config_path" ] || die "configure_post.sh did not create config file"
  assert_file_contains "$_config_path" '"host": "https://config.example"'
  assert_file_contains "$_config_path" '"token": "config-token"'
}

test_install_downloads_release_and_reuses_cached_binary() {
  _skill_copy=$(make_temp_skill_copy)
  "$_skill_copy/scripts/install_post_cli.sh" >/tmp/post-publish-install-1.out 2>/tmp/post-publish-install-1.err
  _version_output=$("$_skill_copy/bin/post" version)
  printf '%s' "$_version_output" | grep -Eq '^post [0-9]+\.[0-9]+\.[0-9]+' >/dev/null 2>&1 || die "installed binary did not report a release version"

  "$_skill_copy/scripts/install_post_cli.sh" >/tmp/post-publish-install-2.out 2>/tmp/post-publish-install-2.err
  _cached_version=$(cat "$_skill_copy/bin/post.version")
  printf '%s' "$_cached_version" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$' || die "post.version was not recorded correctly"
}

write_stub_support_scripts() {
  _skill_dir="$1"
  cat >"$_skill_dir/scripts/configure_post.sh" <<'SH'
#!/bin/sh
set -eu
exit 0
SH
  cat >"$_skill_dir/scripts/install_post_cli.sh" <<'SH'
#!/bin/sh
set -eu
exit 0
SH
  chmod +x "$_skill_dir/scripts/configure_post.sh" "$_skill_dir/scripts/install_post_cli.sh"
}

write_stub_post_binary() {
  _skill_dir="$1"
  _stub_body="$2"
  mkdir -p "$_skill_dir/bin"
  cat >"$_skill_dir/bin/post" <<SH
#!/bin/sh
set -eu
$_stub_body
SH
  chmod +x "$_skill_dir/bin/post"
}

test_share_returns_first_link_line() {
  _skill_copy=$(make_temp_skill_copy)
  write_stub_support_scripts "$_skill_copy"
  write_stub_post_binary "$_skill_copy" 'printf "%s\n" "https://sho.rt/demo"'

  _output=$("$_skill_copy/scripts/share_to_post.sh" --text "hello world" --convert text)
  assert_equals "https://sho.rt/demo" "$_output"
}

test_share_retries_on_slug_conflict() {
  _skill_copy=$(make_temp_skill_copy)
  write_stub_support_scripts "$_skill_copy"
  _counter_file="$_skill_copy/bin/counter"
  write_stub_post_binary "$_skill_copy" "
counter_file='$_counter_file'
count=0
[ -f \"\$counter_file\" ] && count=\$(cat \"\$counter_file\")
count=\$((count + 1))
printf '%s' \"\$count\" >\"\$counter_file\"
if [ \"\$count\" -eq 1 ]; then
  printf '%s\n' 'error: already exists' >&2
  exit 1
fi
printf '%s\n' 'https://sho.rt/conflict-fixed'
"

  _output=$("$_skill_copy/scripts/share_to_post.sh" --text "conflict" --convert text --slug demo)
  assert_equals "https://sho.rt/conflict-fixed" "$_output"
}

test_share_rejects_file_convert_without_file() {
  _skill_copy=$(make_temp_skill_copy)
  write_stub_support_scripts "$_skill_copy"
  write_stub_post_binary "$_skill_copy" 'printf "%s\n" "unexpected"'

  if "$_skill_copy/scripts/share_to_post.sh" --text "hello" --convert file >/tmp/post-publish-share-invalid.out 2>/tmp/post-publish-share-invalid.err; then
    die "share_to_post.sh should fail when --convert file is used without --file"
  fi

  assert_file_contains /tmp/post-publish-share-invalid.err "--convert file requires --file <path>"
}

main() {
  test_configure_uses_environment_without_writing
  test_configure_writes_missing_config
  test_install_downloads_release_and_reuses_cached_binary
  test_share_returns_first_link_line
  test_share_retries_on_slug_conflict
  test_share_rejects_file_convert_without_file
  printf 'post-publish skill tests passed\n'
}

main "$@"
