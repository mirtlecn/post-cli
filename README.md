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
- `POST_PUB_TOPIC` for `post pub`
- `POST_CONFIG` to override the config file path

Default config file path:

```text
~/.config/post/config.json
```

Config file format:

```json
{
  "host": "https://example.com",
  "token": "your-token",
  "pub_topic": "notes"
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
- `post topic refresh`
- `post topic rm`

Shortcut commands:

- `post md`
- `post qr`
- `post file`
- `post pub`
- `post html`
- `post text`
- `post url`

Shortcut commands default to `ttl=10080` minutes unless `-t` is provided explicitly. `ttl=0` means no expiration.
For shortcut commands, automatic clipboard read/write is enabled by default. Passing `-r` or `-w` disables the corresponding default behavior.

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

File-path create commands infer metadata when explicit flags are not provided:

- `title` from `title`, then first H1, then file name
- `slug` from `slug`, then generated from the final title
- `created` from `created`, then `date`, then file modified time, then current time

This applies to:

- `post pub <path>`
- `post new -f <path>`
- `post md|qr|html|text|url -f <path>`
- `post file <path>`
- `post file -f <path>`

`post pub` still additionally infers:

- `topic` from `POST_PUB_TOPIC`, then config `pub_topic`

`post pub` fails when no topic source is configured.

When the path is a directory, `post pub`:

- creates a child topic at `<pub_topic>/<folder_name-or-slug>`
- uploads `.md` files as `md2html`
- uploads other non-hidden files as file uploads
- skips hidden files and hidden directories

Create-capable commands also support:

- `--type <mode>` to set the request type
- `--convert <mode>` as an alias of `--type`
- `-i, --title <title>` to set the item title
- `-p, --topic <topic>` to attach an item to a topic
- `--created <time>` to pass the created time to the API

When `--topic` is set, `--title` is required.
The CLI forwards `--created` as-is and lets the API validate the time format.

## Examples

```bash
post new hello world
post new -f ./notes.txt
post new --created "2026-03-01T08:00:00+08:00" "keep original time"
echo "piped text" | post

post md -f README.md
echo '# Hello' | post md
post pub ./note.md
post pub ./notes
post pub -yu ./notes
post qr https://example.com
post html '<h1>Hello</h1>'
post text
post url https://example.com
post file ./image.png
post text -i "Quick Note" -p anime "topic item"
post file -i "Poster Pack" -p anime ./image.png

post topic new -i "Anime Notes" anime
post topic ls
post topic ls anime
post topic refresh -i "Anime Archive" anime
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
make build
```

Version source:

```text
./VERSION
```

Clean rebuild:

```bash
make rebuild
```

Run tests:

```bash
make test
```

Run the local smoke test:

```bash
POST_HOST=http://localhost:3000 POST_TOKEN=demo make smoke-local
```

Bump the release version, create the version commit, and create the local Git tag:

```bash
./scripts/bump_version.sh v1.3.4
```

## SKILLs for AI Agents

Install "/skills/" and ask your agent to 'Publish a note' or 'Upload a file' to create a new post.

## Credits

MIT Licence

© Mirtle together with OpenAI Codex
