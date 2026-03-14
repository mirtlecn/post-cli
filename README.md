[Go server](https://github.com/mirtlecn/post-go) | [Node.js server](https://github.com/mirtlecn/post) | [WebUI](https://github.com/mirtlecn/post)

# Post

`post` is a cross-platform CLI for creating short links and uploading text, files, clipboard content, or piped input to a [Post-compatible server](https://github.com/mirtlecn/post).

## Features

- Native CLI binaries for Linux, macOS, and Windows
- Upload text, HTML, URLs, Markdown, QR codes, and files
- Manage topics and create topic items with titles
- Read input from arguments, files, stdin, or the system clipboard
- Optional shortcut commands for common content types
- JSON export for create, list, export, and delete flows
- Bash, Zsh, and PowerShell completion scripts

## Installation

Download a release archive from GitHub Releases, extract it, and place the `post` binary somewhere on your `PATH`.

On Windows, use `post.exe`.

## Configuration

`post` reads configuration from environment variables first, then falls back to a JSON config file.

Environment variables:

- `POST_HOST`
- `POST_TOKEN`
- `POST_CONFIG` to override the config file path

Default config file path:

```text
~/.config/post/config.json
```

Config file format:

```json
{
  "host": "https://example.com",
  "token": "your-token"
}
```

Environment variables override values from the config file.

## Commands

Core commands:

- `post new`
- `post ls`
- `post export`
- `post rm`
- `post completion`
- `post version`
- `post help`
- `post topic new`
- `post topic ls`
- `post topic rm`

Shortcut commands:

- `post md`
- `post qr`
- `post file`
- `post html`
- `post text`
- `post url`

Shortcut commands default to `ttl=10080` minutes unless `-t` is provided explicitly. `ttl=0` means no expiration.

## Input Sources

`post new`, `post md`, `post qr`, `post html`, `post text`, and `post url` accept input from:

- positional text arguments
- `-f <path>`
- stdin
- clipboard

`post file` accepts a file path only:

- `post file ./image.png`
- `post file -f ./image.png`

It does not read from stdin or clipboard.

Create-capable commands also support:

- `--type <mode>` to set the request type
- `--convert <mode>` as an alias of `--type`
- `-i, --title <title>` to set the item title
- `-p, --topic <topic>` to attach an item to a topic

When `--topic` is set, `--title` is required.

## Examples

```bash
post new hello world
post new -f ./notes.txt
echo "piped text" | post

post md -f README.md
echo '# Hello' | post md
post qr https://example.com
post html '<h1>Hello</h1>'
post text
post url https://example.com
post file ./image.png
post text -i "Quick Note" -p anime "topic item"
post file -i "Poster Pack" -p anime ./image.png

post topic new anime
post topic ls
post topic ls anime
post topic rm anime

post ls
post ls myslug
post ls -x myslug
post export myslug
post rm myslug
post rm -x myslug
```

## Shell Completion

Generate completion scripts with:

```bash
post completion bash
post completion zsh
post completion powershell
```

Examples:

```bash
source <(post completion zsh)
source <(post completion bash)
```

PowerShell:

```powershell
iex (& post completion powershell)
```

## Clipboard Behavior

Clipboard support uses platform commands instead of GUI libraries:

- macOS: `pbcopy`, `pbpaste`
- Linux: `wl-copy`, `wl-paste`, `xclip`, `xsel`
- Windows: PowerShell clipboard commands

When no clipboard read command is available, clipboard input fails with an explicit error. If no clipboard write command is available, automatic copy is skipped.

## Build

Local build:

```bash
go build -o post ./cmd/post
```

Run tests:

```bash
go test ./...
```

Run the local smoke test:

```bash
POST_HOST=http://localhost:3000 POST_TOKEN=demo ./scripts/smoke_local.sh
```

## SKILLs for AI Agents

Install "/skills/" and ask your agent to 'Publish a note' or 'Upload a file' to create a new post.

## Credits

MIT Licence

© Mirtle together with OpenAI Codex
