package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mirtle/post-cli/internal/api"
	"github.com/mirtle/post-cli/internal/clipboard"
	"github.com/mirtle/post-cli/internal/config"
	"github.com/mirtle/post-cli/internal/post"
)

type App struct {
	stdin  *os.File
	stdout io.Writer
	stderr io.Writer
}

func NewApp(stdin *os.File, stdout io.Writer, stderr io.Writer) *App {
	return &App{stdin: stdin, stdout: stdout, stderr: stderr}
}

func (app *App) Run(ctx context.Context, args []string) error {
	stdinTTY := isTerminal(app.stdin)
	if !stdinTTY && shouldPrependNew(args) {
		args = append([]string{"new"}, args...)
	}

	command := "help"
	if len(args) > 0 {
		command = args[0]
		args = args[1:]
	}

	switch command {
	case "completion":
		return app.runCompletion(args)
	case "new":
		host, token, _, err := loadRuntimeConfig()
		if err != nil {
			return err
		}
		service := newPostService(host, token, app.stdin, app.stderr)
		return app.runNew(ctx, service, args, stdinTTY, host)
	case "ls":
		host, token, _, err := loadRuntimeConfig()
		if err != nil {
			return err
		}
		service := newPostService(host, token, app.stdin, app.stderr)
		return app.runList(ctx, service, args)
	case "export":
		host, token, _, err := loadRuntimeConfig()
		if err != nil {
			return err
		}
		service := newPostService(host, token, app.stdin, app.stderr)
		return app.runExport(ctx, service, args)
	case "rm":
		host, token, _, err := loadRuntimeConfig()
		if err != nil {
			return err
		}
		service := newPostService(host, token, app.stdin, app.stderr)
		return app.runRemove(ctx, service, args)
	case "help", "-h", "--help":
		_, _ = io.WriteString(app.stdout, helpText)
		return nil
	default:
		return fmt.Errorf("unknown command '%s'. Try: post help", command)
	}
}

func loadRuntimeConfig() (string, string, string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", "", "", err
	}

	if cfg.Host == "" || cfg.Token == "" {
		return "", "", cfg.ConfigPath, fmt.Errorf(
			"POST_HOST and POST_TOKEN must be set via environment variables or config file (%s)",
			cfg.ConfigPath,
		)
	}

	return cfg.Host, cfg.Token, cfg.ConfigPath, nil
}

func newPostService(host string, token string, stdin *os.File, stderr io.Writer) *post.Service {
	client := api.NewClient(host, token, http.DefaultClient)
	return post.NewService(client, clipboard.NewSystemService(), stdin, stderr)
}

func (app *App) runNew(
	ctx context.Context,
	service *post.Service,
	args []string,
	stdinTTY bool,
	host string,
) error {
	options, err := parseNewOptions(args)
	if err != nil {
		return err
	}

	if !stdinTTY {
		options.SkipConfirm = true
	}
	options.StdinTTY = stdinTTY
	options.Confirm = func(_ string) (bool, error) {
		fmt.Fprintf(app.stderr, "[Post on]: %s? (y/N) ", host)
		reader := bufio.NewReader(app.stdin)
		answer, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return false, fmt.Errorf("read confirmation: %w", readErr)
		}

		trimmed := strings.TrimSpace(answer)
		return trimmed == "y" || trimmed == "Y" || strings.EqualFold(trimmed, "yes"), nil
	}

	result, err := service.New(ctx, options)
	if err != nil {
		return err
	}

	if result.Stderr != "" {
		_, _ = io.WriteString(app.stderr, result.Stderr)
	}
	if result.Stdout != "" {
		_, _ = io.WriteString(app.stdout, result.Stdout)
	}
	return nil
}

func (app *App) runList(ctx context.Context, service *post.Service, args []string) error {
	path, export, err := parsePathExportOptions(args, "ls")
	if err != nil {
		return err
	}
	output, err := service.List(ctx, path, export)
	if err != nil {
		return err
	}
	_, _ = io.WriteString(app.stdout, output)
	return nil
}

func (app *App) runExport(ctx context.Context, service *post.Service, args []string) error {
	path := ""
	if len(args) > 0 {
		path = args[0]
	}
	output, err := service.Export(ctx, path)
	if err != nil {
		return err
	}
	_, _ = io.WriteString(app.stdout, output)
	return nil
}

func (app *App) runRemove(ctx context.Context, service *post.Service, args []string) error {
	path, export, err := parsePathExportOptions(args, "rm")
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("usage: post rm [-x|--export] <path>")
	}

	output, err := service.Remove(ctx, path, export)
	if err != nil {
		return err
	}
	_, _ = io.WriteString(app.stdout, output)
	return nil
}

func (app *App) runCompletion(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: post completion <bash|zsh>")
	}

	var script string
	switch args[0] {
	case "bash":
		script = bashCompletion
	case "zsh":
		script = zshCompletion
	default:
		return fmt.Errorf("unsupported shell '%s'. Try: post completion <bash|zsh>", args[0])
	}

	_, _ = io.WriteString(app.stdout, script)
	return nil
}

func parseNewOptions(args []string) (post.NewOptions, error) {
	options := post.NewOptions{
		Method: http.MethodPost,
	}

	for index := 0; index < len(args); {
		arg := args[index]
		switch arg {
		case "-s", "--slug":
			value, nextIndex, err := nextValue(args, index)
			if err != nil {
				return options, fmt.Errorf("option %s requires a value", arg)
			}
			options.Slug = value
			index = nextIndex
		case "-t", "--ttl":
			value, nextIndex, err := nextValue(args, index)
			if err != nil {
				return options, fmt.Errorf("option %s requires a number (minutes)", arg)
			}
			ttl, convertErr := strconv.Atoi(value)
			if convertErr != nil {
				return options, fmt.Errorf("option %s requires a number (minutes)", arg)
			}
			options.TTL = &ttl
			index = nextIndex
		case "-u", "--update":
			options.Method = http.MethodPut
			index++
		case "-y", "--no-confirm":
			options.SkipConfirm = true
			index++
		case "-x", "--export":
			options.Export = true
			index++
		case "-f", "--file":
			value, nextIndex, err := nextValue(args, index)
			if err != nil {
				return options, fmt.Errorf("option %s requires a file path", arg)
			}
			options.FilePath = value
			index = nextIndex
		case "-c", "--convert":
			value, nextIndex, err := nextValue(args, index)
			if err != nil {
				return options, fmt.Errorf("option %s requires a value: html|md2html|url|text|qrcode|file", arg)
			}
			if !isValidConvert(value) {
				return options, fmt.Errorf("invalid convert value '%s'. Must be one of: html, md2html, url, text, qrcode, file", value)
			}
			options.Convert = value
			index = nextIndex
		case "--":
			options.Args = append(options.Args, args[index+1:]...)
			return options, nil
		default:
			if strings.HasPrefix(arg, "-") {
				return options, fmt.Errorf("unknown option '%s'. Try: post help", arg)
			}
			options.Args = append(options.Args, args[index:]...)
			return options, nil
		}
	}

	return options, nil
}

func parsePathExportOptions(args []string, command string) (string, bool, error) {
	export := false
	index := 0
	for index < len(args) {
		arg := args[index]
		switch arg {
		case "-x", "--export":
			export = true
			index++
		case "--":
			index++
			if index < len(args) {
				return args[index], export, nil
			}
			return "", export, nil
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, fmt.Errorf("unknown option '%s'. Try: post help", arg)
			}
			return arg, export, nil
		}
	}

	if command == "rm" {
		return "", export, nil
	}
	return "", export, nil
}

func shouldPrependNew(args []string) bool {
	if len(args) == 0 {
		return true
	}

	switch args[0] {
	case "new", "ls", "export", "rm", "help", "completion", "--help", "-h":
		return false
	default:
		return true
	}
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func nextValue(args []string, index int) (string, int, error) {
	if index+1 >= len(args) {
		return "", index, io.EOF
	}
	return args[index+1], index + 2, nil
}

func isValidConvert(value string) bool {
	switch value {
	case "html", "md2html", "url", "text", "qrcode", "file":
		return true
	default:
		return false
	}
}

const helpText = `post - paste & short-URL manager

Usage:
  post new [opts] <text...>    Upload text
  post new [opts] -f <file>    Upload file contents
  post new [opts]              Upload clipboard contents (no -f, no text, no stdin)
  echo "..." | post [new]      Upload from stdin
  post ls                      List all posts (truncated text)
  post ls <path>               Show a specific post
  post ls -x <path>            Show a specific post with full content
  post export                  Export all posts with full text
  post export <path>           Export one post with full content
  post rm <path>               Delete a post
  post rm -x <path>            Delete a post and return full content
  post completion <shell>      Print shell completion script
  post help | -h | --help      Show this help

Options for 'new':
  -f, --file <path>              Read content from file
  -s, --slug <path>              Custom slug/path (default: auto-generated)
  -t, --ttl <minutes>            Expiration time in minutes (default: never)
  -u, --update                   Overwrite if slug already exists (uses PUT)
  -y, --no-confirm               Skip confirmation prompt
  -x, --export                   Return full create/update response
  -c, --convert <mode>           Convert/type before uploading:
                                   html    -> set type to html
                                   md2html -> convert Markdown to HTML (type: html)
                                   url     -> set type to url
                                   text    -> set type to text
                                   qrcode  -> convert content to QR code
                                   file    -> upload binary file (requires -f)

Options for 'ls' and 'rm':
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
  post new hello world
  post new -f ~/notes.txt
  post new -s mycode -f script.sh
  post new -t 60 "expires in 1 hour"
  post new -y "quick note"
  post new                          # uploads clipboard
  echo "piped" | post
  echo "piped" | post new -s myslug
  post ls
  post ls myslug
  post ls -x myslug
  post rm myslug
  post rm -x myslug
  post export myslug
`

const bashCompletion = `# bash completion for post
_post_completion() {
  local current previous command
  COMPREPLY=()
  current="${COMP_WORDS[COMP_CWORD]}"
  previous="${COMP_WORDS[COMP_CWORD-1]}"
  command=""

  if [[ ${#COMP_WORDS[@]} -gt 1 ]]; then
    command="${COMP_WORDS[1]}"
  fi

  if [[ ${COMP_CWORD} -eq 1 ]]; then
    COMPREPLY=($(compgen -W "new ls export rm completion help" -- "${current}"))
    return 0
  fi

  case "${previous}" in
    -c|--convert)
      COMPREPLY=($(compgen -W "html md2html url text qrcode file" -- "${current}"))
      return 0
      ;;
    completion)
      COMPREPLY=($(compgen -W "bash zsh" -- "${current}"))
      return 0
      ;;
    -f|--file)
      COMPREPLY=($(compgen -f -- "${current}"))
      return 0
      ;;
  esac

  case "${command}" in
    new)
      COMPREPLY=($(compgen -W "-s --slug -t --ttl -u --update -y --no-confirm -x --export -f --file -c --convert" -- "${current}"))
      ;;
    ls|rm)
      COMPREPLY=($(compgen -W "-x --export" -- "${current}"))
      ;;
    export)
      COMPREPLY=()
      ;;
    completion)
      COMPREPLY=($(compgen -W "bash zsh" -- "${current}"))
      ;;
    help)
      COMPREPLY=()
      ;;
  esac
}

complete -F _post_completion post
`

const zshCompletion = `#compdef post

_post() {
  local -a subcommands
  subcommands=(
    'new:Upload text, file, stdin, or clipboard content'
    'ls:List all posts or show a specific post'
    'export:Export all posts or one post with full content'
    'rm:Delete a post'
    'completion:Print shell completion script'
    'help:Show help'
  )

  local -a new_options
  new_options=(
    '(-s --slug)'{-s,--slug}'[Custom slug/path]:slug: '
    '(-t --ttl)'{-t,--ttl}'[Expiration time in minutes]:minutes: '
    '(-u --update)'{-u,--update}'[Overwrite if slug already exists]'
    '(-y --no-confirm)'{-y,--no-confirm}'[Skip confirmation prompt]'
    '(-x --export)'{-x,--export}'[Return full create/update response]'
    '(-f --file)'{-f,--file}'[Read content from file]:file:_files'
    '(-c --convert)'{-c,--convert}'[Convert/type before uploading]:convert:(html md2html url text qrcode file)'
  )

  case $words[2] in
    new)
      _arguments -s $new_options '*:text: '
      ;;
    ls)
      _arguments -s '(-x --export)'{-x,--export}'[Return full content]' '*:path: '
      ;;
    export)
      _arguments -s '*:path: '
      ;;
    rm)
      _arguments -s '(-x --export)'{-x,--export}'[Return full content]' '1:path: '
      ;;
    completion)
      _arguments '1:shell:(bash zsh)'
      ;;
    *)
      _arguments \
        '1:subcommand:->subcommand' \
        '*::arg:->args'
      case $state in
        subcommand)
          _describe 'subcommand' subcommands
          ;;
      esac
      ;;
  esac
}

_post "$@"
`
