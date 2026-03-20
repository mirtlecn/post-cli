#!/bin/sh
set -eu

REPO_ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
VERSION_FILE="$REPO_ROOT/VERSION"
TARGET_VERSION=${1:-}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
Usage:
  ./scripts/bump_version.sh vX.Y.Z

What it does:
  1. Update VERSION
  2. Rebuild the CLI with injected build info
  3. Verify `./post version`
  4. Commit the VERSION change
  5. Create or update the local Git tag for that version

Notes:
  - The GitHub release workflow is triggered after pushing the created tag.
  - Existing local tag with the same name will be replaced.
EOF
}

validate_version() {
  printf '%s' "$1" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$' || die "version must match vX.Y.Z"
}

ensure_clean_worktree() {
  [ -z "$(git status --short)" ] || die "worktree is not clean"
}

commit_version_bump() {
  git add "$VERSION_FILE"
  git commit -m "chore: bump version to $TARGET_VERSION" >/dev/null
}

main() {
  [ -n "$TARGET_VERSION" ] || {
    usage
    exit 1
  }

  validate_version "$TARGET_VERSION"

  cd "$REPO_ROOT"
  ensure_clean_worktree

  printf '%s\n' "$TARGET_VERSION" >"$VERSION_FILE"

  make build

  VERSION_OUTPUT=$(./post version)
  printf '%s\n' "$VERSION_OUTPUT" | grep -F "post $TARGET_VERSION" >/dev/null 2>&1 || die "built binary did not report $TARGET_VERSION"

  commit_version_bump

  if git rev-parse -q --verify "refs/tags/$TARGET_VERSION" >/dev/null 2>&1; then
    git tag -d "$TARGET_VERSION" >/dev/null
  fi
  git tag -a "$TARGET_VERSION" -m "$TARGET_VERSION"

  printf 'bumped version to %s\n' "$TARGET_VERSION"
  printf 'verified build output: %s\n' "$(printf '%s\n' "$VERSION_OUTPUT" | head -n 1)"
  printf 'created commit: %s\n' "$(git rev-parse --short HEAD)"
  printf 'created local tag: %s\n' "$TARGET_VERSION"
}

main "$@"
