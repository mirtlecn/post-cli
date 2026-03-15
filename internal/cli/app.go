package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
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
	build  BuildInfo
}

type BuildInfo struct {
	Version   string
	Commit    string
	BuildDate string
}

func NewApp(stdin *os.File, stdout io.Writer, stderr io.Writer, build BuildInfo) *App {
	return &App{stdin: stdin, stdout: stdout, stderr: stderr, build: build}
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
	case "version", "--version", "-v":
		return app.runVersion()
	case "new":
		host, token, _, err := loadRuntimeConfig()
		if err != nil {
			return err
		}
		service := newPostService(host, token, app.stdin, app.stderr)
		return app.runNew(ctx, service, args, stdinTTY, host)
	case "md", "qr", "file", "html", "text", "url":
		host, token, _, err := loadRuntimeConfig()
		if err != nil {
			return err
		}
		service := newPostService(host, token, app.stdin, app.stderr)
		return app.runShortcut(ctx, service, command, args, stdinTTY, host)
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
	case "topic":
		host, token, _, err := loadRuntimeConfig()
		if err != nil {
			return err
		}
		service := newPostService(host, token, app.stdin, app.stderr)
		return app.runTopic(ctx, service, args)
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

	return app.runCreate(ctx, service, options, stdinTTY, host)
}

func (app *App) runShortcut(
	ctx context.Context,
	service *post.Service,
	command string,
	args []string,
	stdinTTY bool,
	host string,
) error {
	options, err := parseShortcutOptions(command, args)
	if err != nil {
		return err
	}

	return app.runCreate(ctx, service, options, stdinTTY, host)
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

func (app *App) runTopic(ctx context.Context, service *post.Service, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: post topic <new|ls|refresh|rm> [args]")
	}

	switch args[0] {
	case "new":
		if len(args) != 2 {
			return fmt.Errorf("usage: post topic new <topic>")
		}
		output, err := service.CreateTopic(ctx, args[1], true)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(app.stdout, output)
		return nil
	case "ls":
		path, export, err := parsePathExportOptions(args[1:], "ls")
		if err != nil {
			return err
		}
		output, err := service.ListTopics(ctx, path, export)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(app.stdout, output)
		return nil
	case "refresh":
		path, export, err := parsePathExportOptions(args[1:], "refresh")
		if err != nil {
			return err
		}
		if path == "" {
			return fmt.Errorf("usage: post topic refresh [-x|--export] <topic>")
		}
		output, err := service.RefreshTopic(ctx, path, export)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(app.stdout, output)
		return nil
	case "rm":
		path, export, err := parsePathExportOptions(args[1:], "rm")
		if err != nil {
			return err
		}
		if path == "" {
			return fmt.Errorf("usage: post topic rm [-x|--export] <topic>")
		}
		output, err := service.RemoveTopic(ctx, path, export)
		if err != nil {
			return err
		}
		_, _ = io.WriteString(app.stdout, output)
		return nil
	default:
		return fmt.Errorf("unknown topic command '%s'. Try: post help", args[0])
	}
}

func (app *App) runVersion() error {
	_, _ = fmt.Fprintf(
		app.stdout,
		"post %s\ncommit: %s\nbuilt: %s\n",
		app.build.Version,
		app.build.Commit,
		app.build.BuildDate,
	)
	return nil
}

func (app *App) runCreate(
	ctx context.Context,
	service *post.Service,
	options post.NewOptions,
	stdinTTY bool,
	host string,
) error {
	if !stdinTTY {
		options.SkipConfirm = true
	}
	options.StdinTTY = stdinTTY
	options.Confirm = func(_ string) (bool, error) {
		fmt.Fprintf(app.stderr, "%-12s %s\n", "post to", host)
		fmt.Fprintf(app.stderr, "%-12s ", "continue?")
		fmt.Fprint(app.stderr, "[y/N] ")
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
