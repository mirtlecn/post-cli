package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mirtle/post-cli/internal/cli"
)

func main() {
	app := cli.NewApp(os.Stdin, os.Stdout, os.Stderr)
	if err := app.Run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
