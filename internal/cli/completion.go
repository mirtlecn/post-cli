package cli

import (
	"fmt"
	"io"
)

func (app *App) runCompletion(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: post completion <bash|zsh|powershell>")
	}

	var script string
	switch args[0] {
	case "bash":
		script = bashCompletion
	case "zsh":
		script = zshCompletion
	case "powershell":
		script = powerShellCompletion
	default:
		return fmt.Errorf("unsupported shell '%s'. Try: post completion <bash|zsh|powershell>", args[0])
	}

	_, _ = io.WriteString(app.stdout, script)
	return nil
}

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
    COMPREPLY=($(compgen -W "new text md file url html qr ls export rm version completion help" -- "${current}"))
    return 0
  fi

  case "${previous}" in
    -c|--convert)
      COMPREPLY=($(compgen -W "html md2html url text qrcode file" -- "${current}"))
      return 0
      ;;
    completion)
      COMPREPLY=($(compgen -W "bash zsh powershell" -- "${current}"))
      return 0
      ;;
    -f|--file)
      COMPREPLY=($(compgen -f -- "${current}"))
      return 0
      ;;
  esac

  case "${command}" in
    new)
      COMPREPLY=($(compgen -W "-f --file -s --slug -t --ttl -y --no-confirm -u --update -x --export -c --convert" -- "${current}"))
      ;;
    md|qr|html|text|url)
      COMPREPLY=($(compgen -W "-f --file -s --slug -t --ttl -y --no-confirm -u --update -x --export" -- "${current}"))
      ;;
    file)
      COMPREPLY=($(compgen -W "-f --file -s --slug -t --ttl -y --no-confirm -u --update -x --export" -- "${current}"))
      ;;
    ls|rm)
      COMPREPLY=($(compgen -W "-x --export" -- "${current}"))
      ;;
    export)
      COMPREPLY=()
      ;;
    completion)
      COMPREPLY=($(compgen -W "bash zsh powershell" -- "${current}"))
      ;;
    version)
      COMPREPLY=()
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
    'text:Upload text content'
    'url:Upload URL content'
    'md:Upload Markdown as HTML'
    'file:Upload a file path directly'
    'html:Upload HTML content'
    'qr:Upload text as QR code'
    'ls:List all posts or show a specific post'
    'export:Export all posts or one post with full content'
    'rm:Delete a post'
    'version:Show build version information'
    'completion:Print shell completion script'
    'help:Show help'
  )

  local -a new_options
  new_options=(
    '(-f --file)'{-f,--file}'[Read content from file]:file:_files'
    '(-s --slug)'{-s,--slug}'[Custom slug/path]:slug: '
    '(-t --ttl)'{-t,--ttl}'[Expiration time in minutes]:minutes: '
    '(-y --no-confirm)'{-y,--no-confirm}'[Skip confirmation prompt]'
    '(-u --update)'{-u,--update}'[Overwrite if slug already exists]'
    '(-x --export)'{-x,--export}'[Return full create/update response]'
    '(-c --convert)'{-c,--convert}'[Convert/type before uploading]:convert:(html md2html url text qrcode file)'
  )

  local -a shortcut_options
  shortcut_options=(
    '(-f --file)'{-f,--file}'[Read content from file]:file:_files'
    '(-s --slug)'{-s,--slug}'[Custom slug/path]:slug: '
    '(-t --ttl)'{-t,--ttl}'[Override expiration time in minutes]:minutes: '
    '(-y --no-confirm)'{-y,--no-confirm}'[Skip confirmation prompt]'
    '(-u --update)'{-u,--update}'[Overwrite if slug already exists]'
    '(-x --export)'{-x,--export}'[Return full create/update response]'
  )

  case $words[2] in
    new)
      _arguments -s $new_options '*:text: '
      ;;
    md|qr|html|text|url)
      _arguments -s $shortcut_options '*:text: '
      ;;
    file)
      _arguments -s \
        '(-s --slug)'{-s,--slug}'[Custom slug/path]:slug: ' \
        '(-t --ttl)'{-t,--ttl}'[Override expiration time in minutes]:minutes: ' \
        '(-u --update)'{-u,--update}'[Overwrite if slug already exists]' \
        '(-y --no-confirm)'{-y,--no-confirm}'[Skip confirmation prompt]' \
        '(-x --export)'{-x,--export}'[Return full create/update response]' \
        '(-f --file)'{-f,--file}'[Upload file path]:file:_files' \
        '1:file:_files'
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
      _arguments '1:shell:(bash zsh powershell)'
      ;;
    version)
      _arguments
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

compdef _post post
`

const powerShellCompletion = `Register-ArgumentCompleter -Native -CommandName post -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $tokens = $commandAst.CommandElements | ForEach-Object { $_.Extent.Text }
    $subcommands = @('new', 'text', 'md', 'file', 'url', 'html', 'qr', 'ls', 'export', 'rm', 'version', 'completion', 'help')
    $newOptions = @('-f', '--file', '-s', '--slug', '-t', '--ttl', '-y', '--no-confirm', '-u', '--update', '-x', '--export', '-c', '--convert')
    $shortcutOptions = @('-f', '--file', '-s', '--slug', '-t', '--ttl', '-y', '--no-confirm', '-u', '--update', '-x', '--export')
    $lsOptions = @('-x', '--export')
    $shells = @('bash', 'zsh', 'powershell')
    $convertValues = @('html', 'md2html', 'url', 'text', 'qrcode', 'file')

    if ($tokens.Count -le 2) {
        $subcommands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }

    $command = $tokens[1]
    $previous = if ($tokens.Count -gt 2) { $tokens[-1] } else { '' }

    if ($previous -in @('-f', '--file')) {
        Get-ChildItem -Name -Path "$wordToComplete*" -ErrorAction SilentlyContinue | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ProviderItem', $_)
        }
        return
    }

    if ($previous -in @('-c', '--convert')) {
        $convertValues | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }

    if ($command -eq 'completion') {
        $shells | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }

    $candidates = switch ($command) {
        'new' { $newOptions }
        'md' { $shortcutOptions }
        'qr' { $shortcutOptions }
        'html' { $shortcutOptions }
        'text' { $shortcutOptions }
        'url' { $shortcutOptions }
        'file' { $shortcutOptions }
        'ls' { $lsOptions }
        'rm' { $lsOptions }
        default { @() }
    }

    $candidates | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
    }
}`
