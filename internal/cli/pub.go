package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mirtle/post-cli/internal/config"
	"github.com/mirtle/post-cli/internal/post"
)

var nowFunc = time.Now

type pubOptions struct {
	FilePath    string
	Slug        string
	Title       string
	TTL         *int
	SkipConfirm bool
}

func parsePubOptions(args []string) (pubOptions, error) {
	options := pubOptions{}
	for index := 0; index < len(args); {
		arg := args[index]
		switch arg {
		case "-i", "--title":
			value, nextIndex, err := nextValue(args, index)
			if err != nil {
				return pubOptions{}, fmt.Errorf("option %s requires a value", arg)
			}
			options.Title = value
			index = nextIndex
		case "-s", "--slug":
			value, nextIndex, err := nextValue(args, index)
			if err != nil {
				return pubOptions{}, fmt.Errorf("option %s requires a value", arg)
			}
			options.Slug = value
			index = nextIndex
		case "-t", "--ttl":
			value, nextIndex, err := nextValue(args, index)
			if err != nil {
				return pubOptions{}, fmt.Errorf("option %s requires a non-negative number (minutes)", arg)
			}
			ttl, convertErr := strconv.Atoi(value)
			if convertErr != nil || ttl < 0 {
				return pubOptions{}, fmt.Errorf("option %s requires a non-negative number (minutes)", arg)
			}
			options.TTL = &ttl
			index = nextIndex
		case "-y", "--no-confirm":
			options.SkipConfirm = true
			index++
		case "--":
			index++
			if index >= len(args) {
				return pubOptions{}, fmt.Errorf("usage: post pub [-t|--ttl <minutes>] [-s|--slug <path>] [-i|--title <title>] [-y|--no-confirm] <file.md>")
			}
			if options.FilePath != "" || index+1 != len(args) {
				return pubOptions{}, fmt.Errorf("pub command accepts a single file path")
			}
			options.FilePath = args[index]
			index++
		default:
			if strings.HasPrefix(arg, "-") {
				return pubOptions{}, fmt.Errorf("unknown option '%s'. Try: post help", arg)
			}
			if options.FilePath != "" {
				return pubOptions{}, fmt.Errorf("pub command accepts a single file path")
			}
			options.FilePath = arg
			index++
		}
	}

	if options.FilePath == "" {
		return pubOptions{}, fmt.Errorf("usage: post pub [-t|--ttl <minutes>] [-s|--slug <path>] [-i|--title <title>] [-y|--no-confirm] <file.md>")
	}

	return options, nil
}

func (app *App) runPub(
	ctx context.Context,
	service *post.Service,
	args []string,
	stdinTTY bool,
	host string,
	cfg config.Config,
) error {
	options, err := parsePubOptions(args)
	if err != nil {
		return err
	}

	topic := cfg.PubTopic
	if topic == "" {
		return fmt.Errorf("POST_PUB_TOPIC or pub_topic must be set for post pub")
	}

	return app.runCreate(ctx, service, post.NewOptions{
		FilePath:    options.FilePath,
		Slug:        options.Slug,
		Title:       options.Title,
		Topic:       topic,
		TTL:         options.TTL,
		Type:        "md2html",
		Method:      "POST",
		SkipConfirm: options.SkipConfirm,
	}, stdinTTY, host)
}
