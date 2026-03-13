package cli

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

func TestParseNewOptions(t *testing.T) {
	options, err := parseNewOptions([]string{"-s", "demo", "-t", "15", "-u", "-x", "-c", "text", "hello", "world"})
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
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{})

	err := app.Run(context.Background(), []string{"completion", "bash"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), "complete -F _post_completion post") {
		t.Fatalf("unexpected completion output: %q", stdout.String())
	}
}

func TestRunCompletionRejectsUnsupportedShell(t *testing.T) {
	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{})

	err := app.Run(context.Background(), []string{"completion", "fish"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestHelpDoesNotRequireConfig(t *testing.T) {
	var stdout bytes.Buffer
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{})

	err := app.Run(context.Background(), []string{"help"})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.Contains(stdout.String(), "post completion <shell>") {
		t.Fatalf("unexpected help output: %q", stdout.String())
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
}

func TestParseShortcutOptionsRejectsMissingFilePath(t *testing.T) {
	_, err := parseShortcutOptions("file", nil)
	if err == nil || err.Error() != "file command requires a file path" {
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
