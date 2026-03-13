package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/mirtle/post-cli/internal/buildinfo"
	"github.com/mirtle/post-cli/internal/cli"
)

type appRunner interface {
	Run(ctx context.Context, args []string) error
}

var newApp = func(stdin io.Reader, stdout io.Writer, stderr io.Writer, info cli.BuildInfo) appRunner {
	return cli.NewApp(stdin.(*os.File), stdout, stderr, info)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin *os.File, stdout io.Writer, stderr io.Writer) (exitCode int) {
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Fprintf(stderr, "error: internal panic: %v\n", recovered)
			exitCode = 1
		}
	}()

	currentBuild := buildinfo.Current()
	app := newApp(stdin, stdout, stderr, cli.BuildInfo{
		Version:   currentBuild.Version,
		Commit:    currentBuild.Commit,
		BuildDate: currentBuild.BuildDate,
	})
	if err := app.Run(context.Background(), args); err != nil {
		fmt.Fprintf(stderr, "error: %s\n", err)
		return 1
	}

	return 0
}
