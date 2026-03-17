---
name: post-publish
description: Use when publishing text, Markdown, HTML, URLs, QR codes, files, or clipboard content through the local post CLI and returning a shareable link. Trigger on requests to upload content, create a short link, publish a file, publish Markdown as HTML, publish a QR code, or post content to a configured Post service.
---

# Post Publish

Use this skill to publish content with the local `post` CLI and return the final shareable URL.

## Workflow

1. Resolve the CLI in this order: `post` in PATH, then `./post` in the skill directory.
2. Resolve config in this order: `POST_HOST` and `POST_TOKEN`, then `~/.config/post/config.json`.
3. If the CLI is missing, stop and report that `post` is unavailable. Ask the user to download it from https://github.com/mirtlecn/post-cli manually.
4. If config is missing, ask the user for host and token. Do not invent credentials.
5. Use the shortest non-interactive command that matches the request. Always pass `-y` for create/update commands.
6. Return only the final URL on success unless the user explicitly asks for full output.

## Default Rules

- When running in sandboxed agents (such as Codex), directly request user elevation to gain access to the network and file system; otherwise, the operation may fail due to insufficient permissions.
- Default ttl (`-t` parameter) is `10080` minutes unless the user specifies another value.
- Do not use delete flows in this skill.
- Use `-f ./path/to/file` for uploading file contents for file itself
- Use pipe input for command output, e.g. `echo "hello" | post new -c text -y -t 10080`
- If a command fails, return the exact error instead of guessing.

## Common Commands

Prepare environment when needed:

```bash
export POST_HOST="https://your-post-host"
export POST_TOKEN="your-token"
```

Create text, Markdown, URL, HTML, or QR:

```bash
post new -c text -y -t 10080 "hello world"
post new -c md2html -y -t 10080 -f ./README.md
post new -c url -y -t 10080 "https://example.com"
post new -c html -y -t 10080 -f <(echo '<h1>Hello</h1>')
post new -c qr -y -t 10080 "https://example.com"
```

Upload files or file contents:

```bash
post new -c file -y -t 10080 -f ./note.txt
```

## Notes

- `post help` is the fallback when a flag combination is unclear.
- Config file format:

```json
// ~/.config/post/config.json
{
  "host": "https://example.com",
  "token": "your-token"
}
```
