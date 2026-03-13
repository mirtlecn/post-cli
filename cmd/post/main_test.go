package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/mirtle/post-cli/internal/cli"
)

type stubApp struct {
	runFunc func(ctx context.Context, args []string) error
}

func (app *stubApp) Run(ctx context.Context, args []string) error {
	return app.runFunc(ctx, args)
}

func TestRunPrintsApplicationError(t *testing.T) {
	originalNewApp := newApp
	t.Cleanup(func() {
		newApp = originalNewApp
	})

	newApp = func(stdin io.Reader, stdout io.Writer, stderr io.Writer, info cli.BuildInfo) appRunner {
		return &stubApp{
			runFunc: func(ctx context.Context, args []string) error {
				return errors.New("boom")
			},
		}
	}

	var stderr bytes.Buffer
	exitCode := run([]string{"ls"}, os.Stdin, &bytes.Buffer{}, &stderr)
	if exitCode != 1 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr.String() != "error: boom\n" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunRecoversPanicWithoutStackTrace(t *testing.T) {
	originalNewApp := newApp
	t.Cleanup(func() {
		newApp = originalNewApp
	})

	newApp = func(stdin io.Reader, stdout io.Writer, stderr io.Writer, info cli.BuildInfo) appRunner {
		return &stubApp{
			runFunc: func(ctx context.Context, args []string) error {
				panic("panic payload")
			},
		}
	}

	var stderr bytes.Buffer
	exitCode := run([]string{"new"}, os.Stdin, &bytes.Buffer{}, &stderr)
	if exitCode != 1 {
		t.Fatalf("unexpected exit code: %d", exitCode)
	}
	if stderr.String() != "error: internal panic: panic payload\n" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}
