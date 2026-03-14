package cli

const helpText = `post - paste & short-URL manager

Usage:
  post new [opts] <text...>    Upload text
  post md [opts] [text...]     Upload Markdown as HTML (default ttl: 10080)
  post qr [opts] [text...]     Upload text as QR code (default ttl: 10080)
  post file [opts] <file>      Upload a file path directly (default ttl: 10080)
  post html [opts] [text...]   Upload HTML content (default ttl: 10080)
  post text [opts] [text...]   Upload text content (default ttl: 10080)
  post url [opts] [text...]    Upload URL content (default ttl: 10080)
  post topic new <topic>       Create a topic
  post topic ls                List all topics
  post topic ls <topic>        Show one topic
  post topic rm <topic>        Delete a topic
  post new [opts] -f <file>    Upload file contents
  post new -r [opts]           Upload clipboard contents (explicit read)
  echo "..." | post [new]      Upload from stdin
  post ls                      List all posts (truncated text)
  post ls <path>               Show a specific post
  post ls -x <path>            Show a specific post with full content
  post export                  Export all posts with full text
  post export <path>           Export one post with full content
  post rm <path>               Delete a post
  post rm -x <path>            Delete a post and return full content
  post completion <shell>      Print shell completion script
  post version | -v            Show build version information
  post help | -h | --help      Show this help

Options for 'new':
  -f, --file <path>              Read content from file
  -s, --slug <path>              Custom slug/path (default: auto-generated)
  -i, --title <title>            Set item title
  -p, --topic <topic>            Attach item to a topic (requires --title)
  -t, --ttl <minutes>            Expiration time in minutes (0 means never)
  -u, --update                   Overwrite if slug already exists (uses PUT)
  -y, --no-confirm               Skip confirmation prompt
  -x, --export                   Return full create/update response
  -r, --read-clipboard           Read content from clipboard when no text/-f/stdin
  -w, --write-clipboard          Copy created short URL to clipboard
  --type <mode>                  Set request type before uploading:
                                   html    -> set type to html
                                   md2html -> convert Markdown to HTML (type: html)
                                   url     -> set type to url
                                   text    -> set type to text
                                   qrcode  -> convert content to QR code
                                   file    -> upload binary file (requires -f)
                                   topic   -> create a topic resource
  -c, --convert <mode>           Alias of --type

Options for 'ls' and 'rm':
  -x, --export                   Return full content

Options for shortcut commands:
  -s, --slug <path>              Custom slug/path
  -i, --title <title>            Set item title
  -p, --topic <topic>            Attach item to a topic (requires --title)
  -t, --ttl <minutes>            Override default 10080-minute expiration
  -u, --update                   Overwrite if slug already exists
  -y, --no-confirm               Skip confirmation prompt
  -x, --export                   Return full create/update response
  -f, --file <path>              Read content from file (not for post file)
  -r, --read-clipboard           Enabled by default for md/qr/html/text/url (not for post file)
  -w, --write-clipboard          Enabled by default for shortcut commands

Options for 'topic ls' and 'topic rm':
  -x, --export                   Return full content

Environment variables:
  POST_HOST    Base endpoint URL (e.g. https://example.com)
  POST_TOKEN   Bearer token
  POST_CONFIG  Optional config file path override

Config file:
  Default path: ~/.config/post/config.json
  JSON format:
    {
      "host": "https://example.com",
      "token": "your-token"
    }
  Environment variables override config file values

Examples:
  post completion bash
  post completion zsh
  post completion powershell
  post version
  post new hello world
  post md -f README.md
  echo '# Hello' | post md
  post qr https://example.com
  post file ./image.png
  post file -f ./image.png
  post html '<h1>Hello</h1>'
  post html -f snippet.html
  post text -p anime -i "Quick Note" "topic item"
  post file -p anime -i "Poster Pack" ./poster.png
  post text
  post text -f note.txt
  post url https://example.com
  post topic new anime
  post topic ls
  post topic ls anime
  post topic rm anime
  post new -f ~/notes.txt
  post new -s mycode -f script.sh
  post new -t 60 "expires in 1 hour"
  post new -t 0 "never expires"
  post new -y "quick note"
  post new -r                       # uploads clipboard
  post new -w "copy this short URL"
  post new -rw "explicit read/write clipboard mode"
  echo "piped" | post
  echo "piped" | post new -s myslug
  post ls
  post ls myslug
  post ls -x myslug
  post rm myslug
  post rm -x myslug
  post export myslug
`
