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
_post_topic_names() {
  post topic ls 2>/dev/null | sed -n 's/^[[:space:]]*"path": "\(.*\)",$/\1/p'
}

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
    COMPREPLY=($(compgen -W "new text md file pub url html qr ls export rm topic version completion help" -- "${current}"))
    return 0
  fi

  case "${previous}" in
    -c|--convert|--type)
      COMPREPLY=($(compgen -W "html md2html url text qrcode file topic" -- "${current}"))
      return 0
      ;;
    topic)
      COMPREPLY=($(compgen -W "new ls refresh rm" -- "${current}"))
      return 0
      ;;
    completion)
      COMPREPLY=($(compgen -W "bash zsh powershell" -- "${current}"))
      return 0
      ;;
    -p|--topic)
      COMPREPLY=($(compgen -W "$(_post_topic_names)" -- "${current}"))
      return 0
      ;;
    -f|--file)
      COMPREPLY=($(compgen -f -- "${current}"))
      return 0
      ;;
  esac

  case "${command}" in
    new)
      COMPREPLY=($(compgen -W "-f --file -s --slug -i --title -p --topic --created -t --ttl -y --no-confirm -u --update -x --export -r --read-clipboard -w --write-clipboard -c --convert --type" -- "${current}"))
      ;;
    md|qr|html|text|url)
      COMPREPLY=($(compgen -W "-f --file -s --slug -i --title -p --topic --created -t --ttl -y --no-confirm -u --update -x --export -r --read-clipboard -w --write-clipboard" -- "${current}"))
      ;;
    pub)
      if [[ "${current}" == -* ]]; then
        COMPREPLY=($(compgen -W "-s --slug -i --title -t --ttl -y --no-confirm" -- "${current}"))
      else
        COMPREPLY=($(compgen -f -- "${current}"))
      fi
      ;;
    file)
      if [[ "${current}" == -* ]]; then
        COMPREPLY=($(compgen -W "-f --file -s --slug -i --title -p --topic --created -t --ttl -y --no-confirm -u --update -x --export -w --write-clipboard" -- "${current}"))
      else
        COMPREPLY=($(compgen -f -- "${current}"))
      fi
      ;;
    ls|rm)
      COMPREPLY=($(compgen -W "-x --export" -- "${current}"))
      ;;
    topic)
      case "${COMP_WORDS[2]}" in
        new)
          if [[ "${current}" == -* ]]; then
            COMPREPLY=($(compgen -W "-i --title -x --export" -- "${current}"))
          fi
          ;;
        ls|refresh|rm)
          if [[ "${COMP_WORDS[2]}" == "refresh" && "${current}" == -* ]]; then
            COMPREPLY=($(compgen -W "-i --title -x --export" -- "${current}"))
          elif [[ "${current}" == -* ]]; then
            COMPREPLY=($(compgen -W "-x --export" -- "${current}"))
          else
            COMPREPLY=($(compgen -W "$(_post_topic_names)" -- "${current}"))
          fi
          ;;
        *)
          COMPREPLY=($(compgen -W "new ls refresh rm" -- "${current}"))
          ;;
      esac
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

_post_topic_names() {
  local -a topic_names
  topic_names=(${(f)"$(post topic ls 2>/dev/null | sed -n 's/^[[:space:]]*\"path\": \"\(.*\)\",$/\1/p')"})
  _describe 'topic' topic_names
}

_post() {
  local -a subcommands
  subcommands=(
    'new:Upload text, file, stdin, or clipboard content'
    'text:Upload text content'
    'url:Upload URL content'
    'md:Upload Markdown as HTML'
    'file:Upload a file path directly'
    'pub:Publish Markdown file with inferred metadata'
    'html:Upload HTML content'
    'qr:Upload text as QR code'
    'ls:List all posts or show a specific post'
    'export:Export all posts or one post with full content'
    'rm:Delete a post'
    'topic:Manage topics'
    'version:Show build version information'
    'completion:Print shell completion script'
    'help:Show help'
  )

  local -a new_options
  new_options=(
    '(-f --file)'{-f,--file}'[Read content from file]:file:_files'
    '(-s --slug)'{-s,--slug}'[Custom slug/path]:slug: '
    '(-i --title)'{-i,--title}'[Set item title]:title: '
    '(-p --topic)'{-p,--topic}'[Attach item to a topic]:topic:_post_topic_names'
    '(--created)'--created'[Set created time and send it to the API]:time: '
    '(-t --ttl)'{-t,--ttl}'[Expiration time in minutes (0 means never)]:minutes: '
    '(-y --no-confirm)'{-y,--no-confirm}'[Skip confirmation prompt]'
    '(-u --update)'{-u,--update}'[Overwrite if slug already exists]'
    '(-x --export)'{-x,--export}'[Return full create/update response]'
    '(-r --read-clipboard)'{-r,--read-clipboard}'[Read content from clipboard]'
    '(-w --write-clipboard)'{-w,--write-clipboard}'[Copy result URL to clipboard]'
    '(--type)'--type'[Set request type]:type:(html md2html url text qrcode file topic)'
    '(-c --convert)'{-c,--convert}'[Alias of --type]:type:(html md2html url text qrcode file topic)'
  )

  local -a shortcut_options
  shortcut_options=(
    '(-f --file)'{-f,--file}'[Read content from file]:file:_files'
    '(-s --slug)'{-s,--slug}'[Custom slug/path]:slug: '
    '(-i --title)'{-i,--title}'[Set item title]:title: '
    '(-p --topic)'{-p,--topic}'[Attach item to a topic]:topic:_post_topic_names'
    '(--created)'--created'[Set created time and send it to the API]:time: '
    '(-t --ttl)'{-t,--ttl}'[Override expiration time in minutes]:minutes: '
    '(-y --no-confirm)'{-y,--no-confirm}'[Skip confirmation prompt]'
    '(-u --update)'{-u,--update}'[Overwrite if slug already exists]'
    '(-x --export)'{-x,--export}'[Return full create/update response]'
    '(-r --read-clipboard)'{-r,--read-clipboard}'[Read content from clipboard]'
    '(-w --write-clipboard)'{-w,--write-clipboard}'[Copy result URL to clipboard]'
  )

  case $words[2] in
    new)
      shift words
      (( CURRENT -= 1 ))
      _arguments -s $new_options '*:text: '
      ;;
    md|qr|html|text|url)
      shift words
      (( CURRENT -= 1 ))
      _arguments -s $shortcut_options '*:text: '
      ;;
    pub)
      shift words
      (( CURRENT -= 1 ))
      _arguments -s \
        '(-s --slug)'{-s,--slug}'[Override front matter slug]:slug: ' \
        '(-i --title)'{-i,--title}'[Override inferred title]:title: ' \
        '(-t --ttl)'{-t,--ttl}'[Optional TTL override]:minutes: ' \
        '(-y --no-confirm)'{-y,--no-confirm}'[Skip confirmation prompt]' \
        '1:file:_files'
      ;;
    file)
      shift words
      (( CURRENT -= 1 ))
      _arguments -s \
        '(-s --slug)'{-s,--slug}'[Custom slug/path]:slug: ' \
        '(-i --title)'{-i,--title}'[Set item title]:title: ' \
        '(-p --topic)'{-p,--topic}'[Attach item to a topic]:topic:_post_topic_names' \
        '(--created)'--created'[Set created time and send it to the API]:time: ' \
        '(-t --ttl)'{-t,--ttl}'[Override expiration time in minutes]:minutes: ' \
        '(-u --update)'{-u,--update}'[Overwrite if slug already exists]' \
        '(-y --no-confirm)'{-y,--no-confirm}'[Skip confirmation prompt]' \
        '(-x --export)'{-x,--export}'[Return full create/update response]' \
        '(-w --write-clipboard)'{-w,--write-clipboard}'[Copy result URL to clipboard]' \
        '(-f --file)'{-f,--file}'[Upload file path]:file:_files' \
        '1:file:_files'
      ;;
    ls)
      shift words
      (( CURRENT -= 1 ))
      _arguments -s '(-x --export)'{-x,--export}'[Return full content]' '*:path: '
      ;;
    export)
      shift words
      (( CURRENT -= 1 ))
      _arguments -s '*:path: '
      ;;
    rm)
      shift words
      (( CURRENT -= 1 ))
      _arguments -s '(-x --export)'{-x,--export}'[Return full content]' '1:path: '
      ;;
    topic)
      shift words
      (( CURRENT -= 1 ))
      case $words[2] in
        new)
          _arguments -s \
            '(-i --title)'{-i,--title}'[Set topic title]:title: ' \
            '(-x --export)'{-x,--export}'[Return full content]' \
            '1:topic: '
          ;;
        ls)
          _arguments -s '(-x --export)'{-x,--export}'[Return full content]' '*:topic:_post_topic_names'
          ;;
        refresh)
          _arguments -s \
            '(-i --title)'{-i,--title}'[Set topic title]:title: ' \
            '(-x --export)'{-x,--export}'[Return full content]' \
            '*:topic:_post_topic_names'
          ;;
        rm)
          _arguments -s '(-x --export)'{-x,--export}'[Return full content]' '*:topic:_post_topic_names'
          ;;
        *)
          _arguments '1:subcommand:(new ls refresh rm)' '*::arg: '
          ;;
      esac
      ;;
    completion)
      shift words
      (( CURRENT -= 1 ))
      _arguments '1:shell:(bash zsh powershell)'
      ;;
    version)
      shift words
      (( CURRENT -= 1 ))
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

    function Get-PostTopicNames {
        try {
            $result = post topic ls 2>$null | ConvertFrom-Json
            $result | ForEach-Object { $_.path }
        } catch {
            @()
        }
    }

    $tokens = $commandAst.CommandElements | ForEach-Object { $_.Extent.Text }
    $subcommands = @('new', 'text', 'md', 'file', 'pub', 'url', 'html', 'qr', 'ls', 'export', 'rm', 'topic', 'version', 'completion', 'help')
    $newOptions = @('-f', '--file', '-s', '--slug', '-i', '--title', '-p', '--topic', '--created', '-t', '--ttl', '-y', '--no-confirm', '-u', '--update', '-x', '--export', '-r', '--read-clipboard', '-w', '--write-clipboard', '-c', '--convert', '--type')
    $shortcutOptions = @('-f', '--file', '-s', '--slug', '-i', '--title', '-p', '--topic', '--created', '-t', '--ttl', '-y', '--no-confirm', '-u', '--update', '-x', '--export', '-r', '--read-clipboard', '-w', '--write-clipboard')
    $fileOptions = @('-f', '--file', '-s', '--slug', '-i', '--title', '-p', '--topic', '--created', '-t', '--ttl', '-y', '--no-confirm', '-u', '--update', '-x', '--export', '-w', '--write-clipboard')
    $pubOptions = @('-s', '--slug', '-i', '--title', '-t', '--ttl', '-y', '--no-confirm')
    $lsOptions = @('-x', '--export')
    $topicSubcommands = @('new', 'ls', 'refresh', 'rm')
    $shells = @('bash', 'zsh', 'powershell')
    $convertValues = @('html', 'md2html', 'url', 'text', 'qrcode', 'file', 'topic')

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

    if ($previous -in @('-c', '--convert', '--type')) {
        $convertValues | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }

    if ($previous -in @('-p', '--topic')) {
        Get-PostTopicNames | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
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

    if ($command -eq 'topic') {
        if ($tokens.Count -le 3) {
            $topicSubcommands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
            return
        }

        $topicCommand = $tokens[2]
        if ($topicCommand -in @('ls', 'refresh', 'rm') -and -not ($wordToComplete -and $wordToComplete.StartsWith('-'))) {
            Get-PostTopicNames | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
                [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
            }
            return
        }

        $candidates = switch ($topicCommand) {
            'ls' { $lsOptions }
            'refresh' { $lsOptions }
            'rm' { $lsOptions }
            default { @() }
        }

        $candidates | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
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
        'file' {
            if ($wordToComplete -and $wordToComplete.StartsWith('-')) {
                $fileOptions
            } else {
                Get-ChildItem -Name -Path "$wordToComplete*" -ErrorAction SilentlyContinue | ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new($_, $_, 'ProviderItem', $_)
                }
                return
            }
        }
        'pub' {
            if ($wordToComplete -and $wordToComplete.StartsWith('-')) {
                $pubOptions
            } else {
                Get-ChildItem -Name -Path "$wordToComplete*" -ErrorAction SilentlyContinue | ForEach-Object {
                    [System.Management.Automation.CompletionResult]::new($_, $_, 'ProviderItem', $_)
                }
                return
            }
        }
        'ls' { $lsOptions }
        'rm' { $lsOptions }
        default { @() }
    }

    $candidates | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterName', $_)
    }
}`
