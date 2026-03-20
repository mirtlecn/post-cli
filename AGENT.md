# Agent Guide

## Purpose

`post-cli` is a cross-platform Go CLI for creating short links and uploading text, HTML, URLs, Markdown, QR codes, files, clipboard content, and piped input to a Post-compatible server.

The main binary entrypoint is `cmd/post`.

## Repository Layout

- `cmd/post`
  - CLI entrypoint and top-level integration tests.
- `internal/cli`
  - Argument parsing, help output, shell completion, `post pub`, shortcut commands, and create-flow orchestration.
- `internal/post`
  - Service layer that resolves input content, validates create options, and calls the API client.
- `internal/api`
  - HTTP client for Post-compatible servers.
- `internal/config`
  - Environment/config-file loading.
- `internal/metadata`
  - Shared file metadata inference for file-based create flows.
- `internal/clipboard`
  - Platform-specific clipboard support.
- `scripts/smoke_local.sh`
  - End-to-end smoke test against a running local server.
- `scripts/bump_version.sh`
  - Version bump helper for release preparation.
- `VERSION`
  - Source of the CLI version injected at build time.

## Build And Test

- Build:
  - `make build`
- Clean rebuild:
  - `make rebuild`
- Format:
  - `make fmt`
- Run all Go tests:
  - `make test`
  - Equivalent: `go test ./...`
- Run local smoke tests:
  - `POST_HOST=http://localhost:3000 POST_TOKEN=demo make smoke-local`
  - Override `POST_HOST` and `POST_TOKEN` as needed for the local server under test.

## Versioning And Release Prep

The current release helper is `scripts/bump_version.sh`.

Expected workflow:

1. Ensure the git worktree is clean.
2. Run `./scripts/bump_version.sh vX.Y.Z`.
3. The script:
   - updates `VERSION`
   - rebuilds the binary with build metadata
   - verifies `./post version`
   - recreates the local git tag for that version

Important details:

- Version format must match `vX.Y.Z`.
- The script refuses to run with a dirty worktree.
- GitHub release automation is expected to run after pushing the created tag.

## File Metadata Inference

File-based create commands share one metadata inference path.

Covered commands:

- `post pub <file.md>`
- `post new -f <path>`
- `post md|qr|html|text|url -f <path>`
- `post file <path>`
- `post file -f <path>`

Priority rules:

- `title`
  - `--title`
  - front matter `title`
  - first H1
  - file name without extension
- `slug`
  - `--slug`
  - front matter `slug`
  - generated from final title plus current Unix time
- `created`
  - `--created`
  - front matter `created`
  - front matter `date`
  - file modified time
  - current time

Implementation constraints:

- Metadata probing reads only the first `4KB` of the file.
- File extension is not used to decide whether metadata probing is allowed.
- If the probe contains obvious binary data, content-based inference stops and the code falls back to file name and file modified time.
- Unclosed front matter in the probe is treated as "no front matter", not as an error.
- Invalid YAML is an error only when a complete front matter block is present inside the probe.

`post pub` has one extra rule:

- `topic`
  - `POST_PUB_TOPIC`
  - config `pub_topic`
  - otherwise fail

## Key Behavioral Conventions

- Explicit user input must win over inferred values.
- `post file` accepts only file paths and does not read stdin or clipboard.
- Shortcut commands default to `ttl=10080` unless overridden with `-t`.
- Shortcut clipboard defaults:
  - read is enabled by default for `md`, `qr`, `html`, `text`, and `url`
  - write is enabled by default for shortcut commands
  - `-r` or `-w` disables the corresponding default
- Topic items still require `--title` when `--topic` is set for non-file text flows.

## Testing Expectations For Changes

When changing CLI behavior:

- update or add unit tests in the closest package
- update smoke coverage for user-facing create-flow behavior
- keep help text and `README.md` aligned with current behavior

When changing file-input behavior specifically:

- check `internal/metadata/file_test.go`
- check `internal/cli/pub_test.go`
- check `internal/cli/app_test.go`
- run the local smoke test against a real server

## Notes For Future Agents

- Prefer changing shared behavior in one place instead of forking logic between `pub` and other file-based commands.
- `internal/cli/app.go` is currently the central place where file metadata is applied before calling the service layer.
- If a change affects release behavior or version output, review `Makefile`, `VERSION`, `internal/buildinfo`, and `scripts/bump_version.sh` together.
