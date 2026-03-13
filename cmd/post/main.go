package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mirtle/post-cli/internal/buildinfo"
	"github.com/mirtle/post-cli/internal/cli"
)

func main() {
	currentBuild := buildinfo.Current()
	app := cli.NewApp(os.Stdin, os.Stdout, os.Stderr, cli.BuildInfo{
		Version:   currentBuild.Version,
		Commit:    currentBuild.Commit,
		BuildDate: currentBuild.BuildDate,
	})
	if err := app.Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
