package cli

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestParseNewOptions(t *testing.T) {
	options, err := parseNewOptions([]string{"-s", "demo", "-t", "15", "-u", "-r", "-w", "-x", "-c", "text", "hello", "world"})
	if err != nil {
		t.Fatalf("parseNewOptions returned error: %v", err)
	}
	if options.Slug != "demo" {
		t.Fatalf("unexpected slug: %s", options.Slug)
	}
	if options.TTL == nil || *options.TTL != 15 {
		t.Fatalf("unexpected ttl: %v", options.TTL)
	}
	if options.Convert != "text" {
		t.Fatalf("unexpected convert: %s", options.Convert)
	}
	if !options.Export {
		t.Fatal("expected export flag")
	}
	if !options.ReadClipboard {
		t.Fatal("expected read clipboard flag")
	}
	if !options.WriteClipboard {
		t.Fatal("expected write clipboard flag")
	}
	if len(options.Args) != 2 {
		t.Fatalf("unexpected args: %#v", options.Args)
	}
}

func TestParseNewOptionsRejectsInvalidConvert(t *testing.T) {
	_, err := parseNewOptions([]string{"-c", "bad"})
	if err == nil {
		t.Fatal("expected error")
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

func TestParseNewOptionsSupportsCombinedClipboardFlags(t *testing.T) {
	options, err := parseNewOptions([]string{"-rw", "hello"})
	if err != nil {
		t.Fatalf("parseNewOptions returned error: %v", err)
	}
	if !options.ReadClipboard || !options.WriteClipboard {
		t.Fatalf("unexpected clipboard flags: read=%v write=%v", options.ReadClipboard, options.WriteClipboard)
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
}

func TestParseShortcutOptionsUsesDefaultTTL(t *testing.T) {
	options, err := parseShortcutOptions("md", []string{"hello"})
	if err != nil {
		t.Fatalf("parseShortcutOptions returned error: %v", err)
	}

	if options.Convert != "md2html" {
		t.Fatalf("unexpected convert: %s", options.Convert)
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

func TestParseShortcutOptionsAllowsTTLOverride(t *testing.T) {
	options, err := parseShortcutOptions("text", []string{"-t", "60", "hello"})
	if err != nil {
		t.Fatalf("parseShortcutOptions returned error: %v", err)
	}

	if options.TTL == nil || *options.TTL != 60 {
		t.Fatalf("unexpected ttl: %v", options.TTL)
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
	if options.Convert != "file" {
		t.Fatalf("unexpected convert: %s", options.Convert)
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
