package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mirtle/post-cli/internal/config"
	"github.com/mirtle/post-cli/internal/post"
	"gopkg.in/yaml.v3"
)

var nowFunc = time.Now

type pubOptions struct {
	FilePath    string
	Slug        string
	Title       string
	TTL         *int
	SkipConfirm bool
}

type markdownMetadata struct {
	Title   string `yaml:"title"`
	Slug    string `yaml:"slug"`
	Created string `yaml:"created"`
	Date    string `yaml:"date"`
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

	metadata, err := readMarkdownMetadata(options.FilePath)
	if err != nil {
		return err
	}

	topic := cfg.PubTopic
	if topic == "" {
		return fmt.Errorf("POST_PUB_TOPIC or pub_topic must be set for post pub")
	}

	title := options.Title
	if title == "" {
		title = metadata.Title
	}

	slug := options.Slug
	if slug == "" {
		slug = metadata.Slug
	}

	created := metadata.Created
	if created == "" {
		created = nowFunc().Format(time.RFC3339)
	}

	return app.runCreate(ctx, service, post.NewOptions{
		FilePath:    options.FilePath,
		Slug:        slug,
		Title:       title,
		Topic:       topic,
		Created:     created,
		TTL:         options.TTL,
		Type:        "md2html",
		Method:      "POST",
		SkipConfirm: options.SkipConfirm,
	}, stdinTTY, host)
}

func readMarkdownMetadata(filePath string) (markdownMetadata, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return markdownMetadata{}, fmt.Errorf("read markdown file: %w", err)
	}

	metadata, body, err := parseMarkdownFrontMatter(content)
	if err != nil {
		return markdownMetadata{}, err
	}

	if metadata.Title == "" {
		metadata.Title = extractFirstHeading(body)
	}
	if metadata.Title == "" {
		metadata.Title = strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	}
	if metadata.Created == "" {
		metadata.Created = metadata.Date
	}

	return metadata, nil
}

func parseMarkdownFrontMatter(content []byte) (markdownMetadata, []byte, error) {
	trimmedContent := bytes.TrimPrefix(content, []byte("\xef\xbb\xbf"))
	if !bytes.HasPrefix(trimmedContent, []byte("---\n")) && !bytes.Equal(trimmedContent, []byte("---")) && !bytes.HasPrefix(trimmedContent, []byte("---\r\n")) {
		return markdownMetadata{}, trimmedContent, nil
	}

	lines := splitLines(trimmedContent)
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return markdownMetadata{}, trimmedContent, nil
	}

	closingIndex := -1
	for index := 1; index < len(lines); index++ {
		line := strings.TrimSpace(lines[index])
		if line == "---" || line == "..." {
			closingIndex = index
			break
		}
	}
	if closingIndex == -1 {
		return markdownMetadata{}, nil, fmt.Errorf("parse front matter: missing closing delimiter")
	}

	var metadata markdownMetadata
	frontMatter := strings.Join(lines[1:closingIndex], "\n")
	if err := yaml.Unmarshal([]byte(frontMatter), &metadata); err != nil {
		return markdownMetadata{}, nil, fmt.Errorf("parse front matter: %w", err)
	}

	body := strings.Join(lines[closingIndex+1:], "\n")
	return metadata, []byte(body), nil
}

func splitLines(content []byte) []string {
	normalized := strings.ReplaceAll(string(content), "\r\n", "\n")
	return strings.Split(normalized, "\n")
}

func extractFirstHeading(content []byte) string {
	for _, line := range splitLines(content) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
		break
	}
	return ""
}
