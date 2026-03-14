---
name: post-publish
description: Use when publishing links, text, HTML, webpages, Markdown, QR codes, files, or clipboard content to a Post service and returning an accessible link. Trigger on requests like publishing content to the internet, uploading a file and returning a link, creating a short link for a URL, sharing Markdown or HTML online, or turning text into a QR code and publishing it.
---

# Post Publish

Use this skill to publish content with `post-cli` and return the final accessible link by default.

Before publishing, ensure configuration exists and ensure the skill-local release build of `post-cli` is installed.

## Workflow

1. Run `./scripts/configure_post.sh`.
2. Run `./scripts/install_post_cli.sh`.
3. Choose the correct publish mode and call `./scripts/share_to_post.sh`.

Do not compile from source unless the user explicitly asks for it. Prefer the latest GitHub Releases build.

## Rules

- Return only the final link on success unless the user explicitly asks for the full response.
- Use the skill-local binary at `./bin/post`.
- Download the release binary only once. If `./bin/post` already exists, reuse it
- Prefer `~/.config/post/config.json` for persisted configuration.
- If `POST_HOST` and `POST_TOKEN` are both present in the environment, use them without overwriting the config file.
- If configuration is missing, collect `host` and `token`, then write them to `~/.config/post/config.json`.
- If macOS blocks the unsigned binary, first remove the quarantine attribute. If execution still fails, return the exact error and tell the user to allow the app manually in System Settings.
- Do not include delete flows in this skill.

## Publish Mapping

- Plain text:
  - `./scripts/share_to_post.sh --text "content"`
- URL short link:
  - `./scripts/share_to_post.sh --text "https://example.com" --convert url`
- Markdown file:
  - `./scripts/share_to_post.sh --file /abs/path/doc.md --convert md2html`
- HTML content:
  - `./scripts/share_to_post.sh --text "<h1>Hello</h1>" --convert html`
- QR code:
  - `./scripts/share_to_post.sh --text "content" --convert qrcode`
- Binary file upload:
  - `./scripts/share_to_post.sh --file /abs/path/image.png --convert file`
- Clipboard:
  - `./scripts/share_to_post.sh --clipboard`

## Parameter Defaults

- `ttl`: default to `10080` minutes. Omit it only when the user explicitly asks for a permanent link.
- `slug`: use the user-provided slug first. Otherwise generate a readable slug from the content, then retry with suffixes on conflict.
- `convert`: infer from the request when not specified:
  - URL or short-link intent -> `url`
  - Markdown -> `md2html`
  - HTML -> `html`
  - QR code intent -> `qrcode`
  - File upload intent -> `file`
  - Otherwise -> `text`
- `update`: pass only when the user explicitly asks to overwrite an existing slug.
- `export`: pass only when the user wants the full JSON response.

## Failure Handling

- Missing config: ask the user for `POST_HOST` and `POST_TOKEN`, then write `~/.config/post/config.json`.
- Invalid config JSON: return the parse error and stop.
- Missing file for `--convert file`: return a direct error.
- GitHub API or download failure: return the exact install failure and stop.
- Service-side publish failure: return the CLI error without inventing causes.
