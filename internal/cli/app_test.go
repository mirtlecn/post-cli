package cli

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/mirtle/post-cli/internal/api"
	"github.com/mirtle/post-cli/internal/post"
)

func TestParseNewOptions(t *testing.T) {
	options, err := parseNewOptions([]string{"-s", "demo", "--created", "2026-03-01T08:00:00+08:00", "-t", "15", "-u", "-r", "-w", "-x", "-c", "text", "hello", "world"})
	if err != nil {
		t.Fatalf("parseNewOptions returned error: %v", err)
	}
	if options.Slug != "demo" {
		t.Fatalf("unexpected slug: %s", options.Slug)
	}
	if options.TTL == nil || *options.TTL != 15 {
		t.Fatalf("unexpected ttl: %v", options.TTL)
	}
	if options.Created != "2026-03-01T08:00:00+08:00" {
		t.Fatalf("unexpected created: %s", options.Created)
	}
	if options.Type != "text" {
		t.Fatalf("unexpected type: %s", options.Type)
	}
	if !options.Export {
		t.Fatal("expected export flag")
	}
	if !options.ReadClipboard {
		t.Fatal("expected read clipboard flag")
	}
	if !options.ReadClipboardSet {
		t.Fatal("expected read clipboard flag marker")
	}
	if !options.WriteClipboard {
		t.Fatal("expected write clipboard flag")
	}
	if !options.WriteClipboardSet {
		t.Fatal("expected write clipboard flag marker")
	}
	if len(options.Args) != 2 {
		t.Fatalf("unexpected args: %#v", options.Args)
	}
}

func TestParseNewOptionsRejectsInvalidType(t *testing.T) {
	_, err := parseNewOptions([]string{"-c", "bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseNewOptionsRejectsNegativeTTL(t *testing.T) {
	_, err := parseNewOptions([]string{"-t", "-1"})
	if err == nil || err.Error() != "option -t requires a non-negative number (minutes)" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseNewOptionsSupportsTypeTopicAndTitle(t *testing.T) {
	options, err := parseNewOptions([]string{"--type", "text", "-p", "anime", "-i", "Castle Notes", "hello"})
	if err != nil {
		t.Fatalf("parseNewOptions returned error: %v", err)
	}
	if options.Type != "text" || options.Topic != "anime" || options.Title != "Castle Notes" {
		t.Fatalf("unexpected options: %#v", options)
	}
}

func TestParseNewOptionsAllowsMatchingTypeAndConvert(t *testing.T) {
	options, err := parseNewOptions([]string{"--type", "text", "--convert", "text", "hello"})
	if err != nil {
		t.Fatalf("parseNewOptions returned error: %v", err)
	}
	if options.Type != "text" {
		t.Fatalf("unexpected type: %s", options.Type)
	}
}

func TestParseNewOptionsRejectsMismatchedTypeAndConvert(t *testing.T) {
	_, err := parseNewOptions([]string{"--type", "text", "--convert", "html", "hello"})
	if err == nil || err.Error() != "--type and --convert must match when both are provided" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseNewOptionsSupportsCombinedBooleanFlags(t *testing.T) {
	options, err := parseNewOptions([]string{"-uyx", "hello"})
	if err != nil {
		t.Fatalf("parseNewOptions returned error: %v", err)
	}
	if options.Method != http.MethodPut {
		t.Fatalf("unexpected method: %s", options.Method)
	}
	if !options.SkipConfirm {
		t.Fatal("expected no-confirm flag")
	}
	if !options.Export {
		t.Fatal("expected export flag")
	}
}

func TestParseNewOptionsSupportsCombinedFlagsWithSlug(t *testing.T) {
	options, err := parseNewOptions([]string{"-uyx", "-s", "demo", "hello"})
	if err != nil {
		t.Fatalf("parseNewOptions returned error: %v", err)
	}
	if options.Slug != "demo" {
		t.Fatalf("unexpected slug: %s", options.Slug)
	}
	if len(options.Args) != 1 || options.Args[0] != "hello" {
		t.Fatalf("unexpected args: %#v", options.Args)
	}
}

func TestParseNewOptionsSupportsCombinedClipboardFlags(t *testing.T) {
	options, err := parseNewOptions([]string{"-rw", "hello"})
	if err != nil {
		t.Fatalf("parseNewOptions returned error: %v", err)
	}
	if !options.ReadClipboard || !options.WriteClipboard {
		t.Fatalf("unexpected clipboard flags: read=%v write=%v", options.ReadClipboard, options.WriteClipboard)
	}
	if !options.ReadClipboardSet || !options.WriteClipboardSet {
		t.Fatalf("unexpected clipboard flag markers: read=%v write=%v", options.ReadClipboardSet, options.WriteClipboardSet)
	}
}

func TestParseNewOptionsRejectsCombinedValueFlags(t *testing.T) {
	_, err := parseNewOptions([]string{"-uyt", "60", "hello"})
	if err == nil || err.Error() != "option '-t' requires a value and cannot be combined" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShouldPrependNew(t *testing.T) {
	if !shouldPrependNew(nil) {
		t.Fatal("expected prepend for empty args")
	}
	if shouldPrependNew([]string{"ls"}) {
		t.Fatal("did not expect prepend for subcommand")
	}
	if !shouldPrependNew([]string{"hello"}) {
		t.Fatal("expected prepend for free text")
	}
}

func TestShouldPrependNewForCompletion(t *testing.T) {
	if shouldPrependNew([]string{"completion"}) {
		t.Fatal("did not expect prepend for completion")
	}
}

func TestShouldPrependNewForPub(t *testing.T) {
	if shouldPrependNew([]string{"pub"}) {
		t.Fatal("did not expect prepend for pub")
	}
}

func TestRunCompletionDoesNotRequireConfig(t *testing.T) {
	var stdout bytes.Buffer
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{}, BuildInfo{})

	err := app.Run(context.Background(), []string{"completion", "bash"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), "complete -F _post_completion post") {
		t.Fatalf("unexpected completion output: %q", stdout.String())
	}
}

func TestRunCreateUsesAlignedConfirmationPrompt(t *testing.T) {
	inputReader, inputWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("create input pipe: %v", err)
	}
	defer inputReader.Close()

	if _, err := inputWriter.WriteString("n\n"); err != nil {
		t.Fatalf("write confirmation input: %v", err)
	}
	if err := inputWriter.Close(); err != nil {
		t.Fatalf("close input writer: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := NewApp(inputReader, &stdout, &stderr, BuildInfo{})
	service := post.NewService(&stubCreateClient{
		postJSONFunc: func(_ context.Context, _ string, _ api.JSONRequest, _ bool) ([]byte, error) {
			return []byte(`{"surl":"https://sho.rt/abc"}`), nil
		},
	}, &stubCreateClipboard{}, bytes.NewBuffer(nil), &stderr)

	err = app.runCreate(context.Background(), service, post.NewOptions{
		Args: []string{"hello", "world"},
	}, true, "https://t.mirtle.cn")
	if err != nil {
		t.Fatalf("runCreate returned error: %v", err)
	}

	expectedStderr := "content      hello world\n\npost to      https://t.mirtle.cn\ncontinue?    [y/N] Aborted.\n"
	if stderr.String() != expectedStderr {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestBashCompletionIncludesClipboardFlagsAndFilePathCompletion(t *testing.T) {
	var stdout bytes.Buffer
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{}, BuildInfo{})

	err := app.Run(context.Background(), []string{"completion", "bash"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "--read-clipboard") || !strings.Contains(output, "--write-clipboard") {
		t.Fatalf("clipboard flags missing in bash completion: %q", output)
	}
	if !strings.Contains(output, "--type") || !strings.Contains(output, "--created") || !strings.Contains(output, "--no-confirm") || !strings.Contains(output, "pub url html qr ls export rm topic version completion help") {
		t.Fatalf("type/topic completion missing in bash completion: %q", output)
	}
	if !strings.Contains(output, "_post_topic_names()") || !strings.Contains(output, "-p|--topic)") {
		t.Fatalf("dynamic topic completion missing in bash completion: %q", output)
	}
	if !strings.Contains(output, "COMPREPLY=($(compgen -W \"new ls refresh rm\" -- \"${current}\"))") {
		t.Fatalf("topic subcommand completion missing in bash completion: %q", output)
	}
	if !strings.Contains(output, "file)\n      if [[ \"${current}\" == -* ]]; then") || !strings.Contains(output, "COMPREPLY=($(compgen -f -- \"${current}\"))") {
		t.Fatalf("file path completion missing in bash completion: %q", output)
	}
}

func TestRunPowerShellCompletionDoesNotRequireConfig(t *testing.T) {
	var stdout bytes.Buffer
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{}, BuildInfo{})

	err := app.Run(context.Background(), []string{"completion", "powershell"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), "Register-ArgumentCompleter -Native -CommandName post") {
		t.Fatalf("unexpected completion output: %q", stdout.String())
	}
}

func TestPowerShellCompletionIncludesClipboardFlagsAndFilePathCompletion(t *testing.T) {
	var stdout bytes.Buffer
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{}, BuildInfo{})

	err := app.Run(context.Background(), []string{"completion", "powershell"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "--read-clipboard") || !strings.Contains(output, "--write-clipboard") {
		t.Fatalf("clipboard flags missing in powershell completion: %q", output)
	}
	if !strings.Contains(output, "--type") || !strings.Contains(output, "--created") || !strings.Contains(output, "--no-confirm") || !strings.Contains(output, "'pub'") || !strings.Contains(output, "$topicSubcommands = @('new', 'ls', 'refresh', 'rm')") {
		t.Fatalf("type/topic completion missing in powershell completion: %q", output)
	}
	if !strings.Contains(output, "function Get-PostTopicNames") || !strings.Contains(output, "$previous -in @('-p', '--topic')") {
		t.Fatalf("dynamic topic completion missing in powershell completion: %q", output)
	}
	if !strings.Contains(output, "$fileOptions = @(") || !strings.Contains(output, "'file' {\n            if ($wordToComplete -and $wordToComplete.StartsWith('-')) {") {
		t.Fatalf("file path completion missing in powershell completion: %q", output)
	}
}

func TestCompletionPrioritizesFrequentCommands(t *testing.T) {
	var stdout bytes.Buffer
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{}, BuildInfo{})

	err := app.Run(context.Background(), []string{"completion", "zsh"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := stdout.String()
	expected := "'new:Upload text, file, stdin, or clipboard content'\n    'text:Upload text content'\n    'url:Upload URL content'\n    'md:Upload Markdown as HTML'"
	if !strings.Contains(output, expected) {
		t.Fatalf("unexpected subcommand ordering: %q", output)
	}
	if !strings.Contains(output, "--read-clipboard") || !strings.Contains(output, "--write-clipboard") {
		t.Fatalf("clipboard flags missing in zsh completion: %q", output)
	}
	if !strings.Contains(output, "'topic:Manage topics'") || !strings.Contains(output, "'pub:Publish Markdown file with inferred metadata'") || !strings.Contains(output, "--type") || !strings.Contains(output, "--created") || !strings.Contains(output, "--no-confirm") {
		t.Fatalf("type/topic completion missing in zsh completion: %q", output)
	}
	if !strings.Contains(output, "_post_topic_names()") || !strings.Contains(output, ":topic:_post_topic_names") {
		t.Fatalf("dynamic topic completion missing in zsh completion: %q", output)
	}
	if !strings.Contains(output, "'*:topic:_post_topic_names'") {
		t.Fatalf("topic rm dynamic completion missing in zsh completion: %q", output)
	}
	if !strings.Contains(output, "topic)\n      shift words\n      (( CURRENT -= 1 ))") || !strings.Contains(output, "'1:subcommand:(new ls refresh rm)'") {
		t.Fatalf("topic subcommand completion missing in zsh completion: %q", output)
	}
	if !strings.Contains(output, "shift words\n      (( CURRENT -= 1 ))\n      _arguments -s \\\n        '(-s --slug)'") {
		t.Fatalf("zsh subcommand argument shifting missing for file command: %q", output)
	}
	if !strings.Contains(output, "1:file:_files") {
		t.Fatalf("file path completion missing in zsh completion: %q", output)
	}
}

func TestRunCompletionRejectsUnsupportedShell(t *testing.T) {
	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})

	err := app.Run(context.Background(), []string{"completion", "fish"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHelpDoesNotRequireConfig(t *testing.T) {
	var stdout bytes.Buffer
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{}, BuildInfo{})

	err := app.Run(context.Background(), []string{"help"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), "post completion powershell") {
		t.Fatalf("unexpected help output: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "--read-clipboard") || !strings.Contains(stdout.String(), "post new -r") {
		t.Fatalf("help output missing clipboard usage: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "post topic new <topic>") || !strings.Contains(stdout.String(), "post pub [opts] <file.md>") || !strings.Contains(stdout.String(), "--type <mode>") || !strings.Contains(stdout.String(), "--created <time>") || !strings.Contains(stdout.String(), "-y, --no-confirm") || !strings.Contains(stdout.String(), "POST_PUB_TOPIC") {
		t.Fatalf("help output missing topic/type usage: %q", stdout.String())
	}
}

type stubCreateClient struct {
	postJSONFunc func(ctx context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error)
}

func (client *stubCreateClient) PostJSON(ctx context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error) {
	return client.postJSONFunc(ctx, method, payload, export)
}

func (client *stubCreateClient) Get(context.Context, api.JSONRequest, bool) ([]byte, error) {
	panic("unexpected Get call")
}

func (client *stubCreateClient) Delete(context.Context, api.JSONRequest, bool) ([]byte, error) {
	panic("unexpected Delete call")
}

func (client *stubCreateClient) UploadFile(context.Context, string, string, string, string, string, string, *int, bool) ([]byte, error) {
	panic("unexpected UploadFile call")
}

type stubCreateClipboard struct{}

func (clipboard *stubCreateClipboard) ReadText() (string, error) {
	return "", nil
}

func (clipboard *stubCreateClipboard) CanWriteText() bool {
	return false
}

func (clipboard *stubCreateClipboard) WriteText(string) error {
	return nil
}

func TestParseShortcutOptionsUsesDefaultTTL(t *testing.T) {
	options, err := parseShortcutOptions("md", []string{"hello"})
	if err != nil {
		t.Fatalf("parseShortcutOptions returned error: %v", err)
	}

	if options.Type != "md2html" {
		t.Fatalf("unexpected type: %s", options.Type)
	}
	if options.TTL == nil || *options.TTL != 10080 {
		t.Fatalf("unexpected ttl: %v", options.TTL)
	}
	if !options.ReadClipboard {
		t.Fatal("expected read clipboard enabled by default for shortcut command")
	}
	if !options.WriteClipboard {
		t.Fatal("expected write clipboard enabled by default for shortcut command")
	}
	if len(options.Args) != 1 || options.Args[0] != "hello" {
		t.Fatalf("unexpected args: %#v", options.Args)
	}
}

func TestParseShortcutOptionsDisablesDefaultClipboardReadWhenFlagIsSet(t *testing.T) {
	options, err := parseShortcutOptions("text", []string{"-r", "hello"})
	if err != nil {
		t.Fatalf("parseShortcutOptions returned error: %v", err)
	}

	if options.ReadClipboard {
		t.Fatal("did not expect read clipboard enabled when -r is provided for shortcut command")
	}
	if !options.WriteClipboard {
		t.Fatal("expected write clipboard to remain enabled by default")
	}
}

func TestParseShortcutOptionsDisablesDefaultClipboardWriteWhenFlagIsSet(t *testing.T) {
	options, err := parseShortcutOptions("text", []string{"-w", "hello"})
	if err != nil {
		t.Fatalf("parseShortcutOptions returned error: %v", err)
	}

	if !options.ReadClipboard {
		t.Fatal("expected read clipboard to remain enabled by default")
	}
	if options.WriteClipboard {
		t.Fatal("did not expect write clipboard enabled when -w is provided for shortcut command")
	}
}

func TestParseShortcutOptionsDisablesDefaultClipboardReadAndWriteWhenFlagsAreSet(t *testing.T) {
	options, err := parseShortcutOptions("text", []string{"-rw", "hello"})
	if err != nil {
		t.Fatalf("parseShortcutOptions returned error: %v", err)
	}

	if options.ReadClipboard || options.WriteClipboard {
		t.Fatalf("expected both clipboard defaults disabled, got read=%v write=%v", options.ReadClipboard, options.WriteClipboard)
	}
}

func TestRunTopicRejectsUnknownCommand(t *testing.T) {
	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})

	err := app.runTopic(context.Background(), post.NewService(&stubCreateClient{
		postJSONFunc: func(_ context.Context, _ string, _ api.JSONRequest, _ bool) ([]byte, error) {
			return nil, nil
		},
	}, &stubCreateClipboard{}, bytes.NewBuffer(nil), &bytes.Buffer{}), []string{"oops"})
	if err == nil || err.Error() != "unknown topic command 'oops'. Try: post help" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunTopicRefreshUsesTopicType(t *testing.T) {
	var stdout bytes.Buffer
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{}, BuildInfo{})
	service := post.NewService(&stubCreateClient{
		postJSONFunc: func(_ context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error) {
			if method != http.MethodPut || payload.Path != "anime" || payload.Title != "Anime Archive" || payload.Type != "topic" || !export {
				t.Fatalf("unexpected args: %s %#v %v", method, payload, export)
			}
			return []byte(`{"path":"anime","type":"topic","title":"anime","content":"1"}`), nil
		},
	}, &stubCreateClipboard{}, bytes.NewBuffer(nil), &bytes.Buffer{})

	err := app.runTopic(context.Background(), service, []string{"refresh", "-x", "-i", "Anime Archive", "anime"})
	if err != nil {
		t.Fatalf("runTopic returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"type": "topic"`) {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRunTopicNewUsesTitleOption(t *testing.T) {
	var stdout bytes.Buffer
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{}, BuildInfo{})
	service := post.NewService(&stubCreateClient{
		postJSONFunc: func(_ context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error) {
			if method != http.MethodPost || payload.Path != "anime" || payload.Title != "Anime Notes" || payload.Type != "topic" || !export {
				t.Fatalf("unexpected args: %s %#v %v", method, payload, export)
			}
			return []byte(`{"path":"anime","type":"topic","title":"Anime Notes","content":"0"}`), nil
		},
	}, &stubCreateClipboard{}, bytes.NewBuffer(nil), &bytes.Buffer{})

	err := app.runTopic(context.Background(), service, []string{"new", "-i", "Anime Notes", "anime"})
	if err != nil {
		t.Fatalf("runTopic returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), `"title": "Anime Notes"`) {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestParseShortcutOptionsAllowsTTLOverride(t *testing.T) {
	options, err := parseShortcutOptions("text", []string{"-t", "60", "hello"})
	if err != nil {
		t.Fatalf("parseShortcutOptions returned error: %v", err)
	}

	if options.TTL == nil || *options.TTL != 60 {
		t.Fatalf("unexpected ttl: %v", options.TTL)
	}
}

func TestParseShortcutOptionsSkipsDefaultTTLWhenTopicIsSet(t *testing.T) {
	options, err := parseShortcutOptions("text", []string{"-p", "anime", "-i", "Quick Note", "hello"})
	if err != nil {
		t.Fatalf("parseShortcutOptions returned error: %v", err)
	}

	if options.TTL != nil {
		t.Fatalf("expected ttl to stay nil for topic shortcut, got: %v", options.TTL)
	}
}

func TestParseShortcutOptionsUsesPositionalFilePath(t *testing.T) {
	options, err := parseShortcutOptions("file", []string{"./demo.txt"})
	if err != nil {
		t.Fatalf("parseShortcutOptions returned error: %v", err)
	}

	if options.FilePath != "./demo.txt" {
		t.Fatalf("unexpected file path: %s", options.FilePath)
	}
	if len(options.Args) != 0 {
		t.Fatalf("unexpected args: %#v", options.Args)
	}
	if options.Type != "file" {
		t.Fatalf("unexpected type: %s", options.Type)
	}
	if options.ReadClipboard {
		t.Fatal("did not expect read clipboard enabled for file command")
	}
	if !options.WriteClipboard {
		t.Fatal("expected write clipboard enabled for file command")
	}
}

func TestParseShortcutOptionsRejectsMissingFilePath(t *testing.T) {
	_, err := parseShortcutOptions("file", nil)
	if err == nil || err.Error() != "file command requires a file path" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseShortcutOptionsRejectsReadClipboardForFileCommand(t *testing.T) {
	_, err := parseShortcutOptions("file", []string{"-r", "a.txt"})
	if err == nil || err.Error() != "option --read-clipboard is not supported with shortcut command 'file'" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseShortcutOptionsRejectsConflictingFilePathInputs(t *testing.T) {
	_, err := parseShortcutOptions("file", []string{"-f", "a.txt", "b.txt"})
	if err == nil || err.Error() != "file command accepts either a positional file path or -f, not both" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseShortcutOptionsRejectsExtraFileArguments(t *testing.T) {
	_, err := parseShortcutOptions("file", []string{"a.txt", "b.txt"})
	if err == nil || err.Error() != "file command accepts a single file path" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunVersionPrintsBuildInfo(t *testing.T) {
	var stdout bytes.Buffer
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{}, BuildInfo{
		Version:   "v1.2.3",
		Commit:    "abc123",
		BuildDate: "2026-03-13T21:00:00Z",
	})

	err := app.Run(context.Background(), []string{"version"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "post v1.2.3") || !strings.Contains(output, "commit: abc123") || !strings.Contains(output, "built: 2026-03-13T21:00:00Z") {
		t.Fatalf("unexpected version output: %q", output)
	}
}
