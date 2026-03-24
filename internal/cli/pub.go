package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mirtle/post-cli/internal/config"
	"github.com/mirtle/post-cli/internal/metadata"
	"github.com/mirtle/post-cli/internal/post"
)

var nowFunc = time.Now

const pubDirectoryConcurrency = 5

type pubOptions struct {
	FilePath    string
	Slug        string
	Title       string
	TTL         *int
	SkipConfirm bool
	Update      bool
}

func parsePubOptions(args []string) (pubOptions, error) {
	expandedArgs, err := expandCombinedBooleanFlags(args)
	if err != nil {
		return pubOptions{}, err
	}

	options := pubOptions{}
	for index := 0; index < len(expandedArgs); {
		arg := expandedArgs[index]
		switch arg {
		case "-i", "--title":
			value, nextIndex, err := nextValue(expandedArgs, index)
			if err != nil {
				return pubOptions{}, fmt.Errorf("option %s requires a value", arg)
			}
			options.Title = value
			index = nextIndex
		case "-s", "--slug":
			value, nextIndex, err := nextValue(expandedArgs, index)
			if err != nil {
				return pubOptions{}, fmt.Errorf("option %s requires a value", arg)
			}
			options.Slug = value
			index = nextIndex
		case "-t", "--ttl":
			value, nextIndex, err := nextValue(expandedArgs, index)
			if err != nil {
				return pubOptions{}, fmt.Errorf("option %s requires a non-negative number (minutes)", arg)
			}
			ttl, convertErr := strconv.Atoi(value)
			if convertErr != nil || ttl < 0 {
				return pubOptions{}, fmt.Errorf("option %s requires a non-negative number (minutes)", arg)
			}
			options.TTL = &ttl
			index = nextIndex
		case "-u", "--update":
			options.Update = true
			index++
		case "-y", "--no-confirm":
			options.SkipConfirm = true
			index++
		case "--":
			index++
			if index >= len(expandedArgs) {
				return pubOptions{}, fmt.Errorf("usage: post pub [-t|--ttl <minutes>] [-s|--slug <path>] [-i|--title <title>] [-u|--update] [-y|--no-confirm] <path>")
			}
			if options.FilePath != "" || index+1 != len(expandedArgs) {
				return pubOptions{}, fmt.Errorf("pub command accepts a single file path")
			}
			options.FilePath = expandedArgs[index]
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
		return pubOptions{}, fmt.Errorf("usage: post pub [-t|--ttl <minutes>] [-s|--slug <path>] [-i|--title <title>] [-u|--update] [-y|--no-confirm] <path>")
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

	fileInfo, err := os.Stat(options.FilePath)
	if err != nil {
		return fmt.Errorf("file not found: %s", options.FilePath)
	}

	if fileInfo.IsDir() {
		return app.runPubDirectory(ctx, service, options, stdinTTY, host, topic)
	}

	method := http.MethodPost
	if options.Update {
		method = http.MethodPut
	}

	return app.runCreate(ctx, service, post.NewOptions{
		FilePath:    options.FilePath,
		Slug:        options.Slug,
		Title:       options.Title,
		Topic:       topic,
		TTL:         options.TTL,
		Type:        "md2html",
		Method:      method,
		SkipConfirm: options.SkipConfirm,
	}, stdinTTY, host)
}

type pubDirectoryEntry struct {
	FilePath     string
	SlugDir      string
	ItemSlug     string
	Type         string
	ResolvedSlug string
}

type pubDirectoryPlan struct {
	TopicPath  string
	TopicTitle string
	Entries    []pubDirectoryEntry
}

func (app *App) runPubDirectory(
	ctx context.Context,
	service *post.Service,
	options pubOptions,
	stdinTTY bool,
	host string,
	parentTopic string,
) error {
	childSlug := options.Slug
	if childSlug == "" {
		childSlug = metadata.GenerateSlugFromTitle(filepath.Base(options.FilePath))
	}
	topicPath := parentTopic + "/" + childSlug
	topicTitle := options.Title
	if topicTitle == "" {
		topicTitle = filepath.Base(options.FilePath)
	}

	plan, err := planPubDirectory(options.FilePath, topicPath, topicTitle)
	if err != nil {
		return err
	}

	if !options.SkipConfirm && stdinTTY {
		writePubDirectoryConfirmPreview(app.stderr, parentTopic, plan.TopicPath, plan.TopicTitle, plan.Entries, options)
		accepted, confirmErr := app.newConfirmFunc(host)("")
		if confirmErr != nil {
			return confirmErr
		}
		if !accepted {
			_, _ = fmt.Fprint(app.stderr, "Aborted.\n")
			return nil
		}
	}

	topicExists, err := service.TopicExists(ctx, plan.TopicPath)
	if err != nil {
		return err
	}
	var topicResult post.Result
	if !topicExists {
		topicResult, err = service.New(ctx, post.NewOptions{
			Slug:        plan.TopicPath,
			Title:       plan.TopicTitle,
			Type:        "topic",
			Method:      http.MethodPost,
			SkipConfirm: true,
		})
		if err != nil {
			return err
		}
	}

	method := http.MethodPost
	if options.Update {
		method = http.MethodPut
	}

	if err := app.uploadPubDirectoryEntries(ctx, service, plan.Entries, plan.TopicPath, options.TTL, method); err != nil {
		return err
	}

	if !topicExists {
		app.writeCreateResult(topicResult)
		return nil
	}
	app.writeCreateResult(post.Result{Stdout: buildPubDirectoryTopicURL(host, plan.TopicPath) + "\n"})
	return nil
}

func (app *App) uploadPubDirectoryEntries(
	ctx context.Context,
	service *post.Service,
	entries []pubDirectoryEntry,
	topicPath string,
	ttl *int,
	method string,
) error {
	workerCount := pubDirectoryConcurrency
	if len(entries) < workerCount {
		workerCount = len(entries)
	}
	if workerCount == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	entryCh := make(chan pubDirectoryEntry)
	errCh := make(chan error, 1)
	var workers sync.WaitGroup

	for index := 0; index < workerCount; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for entry := range entryCh {
				if err := uploadPubDirectoryEntry(ctx, service, entry, topicPath, ttl, method); err != nil {
					select {
					case errCh <- err:
						cancel()
					default:
					}
					return
				}
			}
		}()
	}

	for _, entry := range entries {
		select {
		case <-ctx.Done():
			workers.Wait()
			select {
			case err := <-errCh:
				return err
			default:
				return ctx.Err()
			}
		case entryCh <- entry:
		}
	}
	close(entryCh)
	workers.Wait()

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func uploadPubDirectoryEntry(
	ctx context.Context,
	service *post.Service,
	entry pubDirectoryEntry,
	topicPath string,
	ttl *int,
	method string,
) error {
	createOptions, err := buildPubDirectoryCreateOptions(entry, ttl, method)
	if err != nil {
		return err
	}

	itemSlug := createOptions.Slug
	if entry.ItemSlug != "" {
		itemSlug = entry.ItemSlug
	}
	createOptions.Slug = joinPubSlugPath(topicPath, entry.SlugDir, itemSlug)

	_, err = service.New(ctx, createOptions)
	return err
}

func buildPubDirectoryCreateOptions(entry pubDirectoryEntry, ttl *int, method string) (post.NewOptions, error) {
	createOptions := post.NewOptions{
		FilePath:    entry.FilePath,
		TTL:         ttl,
		Type:        entry.Type,
		Method:      method,
		SkipConfirm: true,
	}

	if entry.Type == "file" {
		createOptions.Title = strings.TrimSuffix(filepath.Base(entry.FilePath), filepath.Ext(entry.FilePath))
	}

	return applyAutomaticFileMetadata(createOptions)
}

func planPubDirectory(rootPath string, topicPath string, topicTitle string) (pubDirectoryPlan, error) {
	entries, err := collectPubDirectoryEntries(rootPath)
	if err != nil {
		return pubDirectoryPlan{}, err
	}
	if err := validatePubDirectoryEntries(entries, topicPath); err != nil {
		return pubDirectoryPlan{}, err
	}
	return pubDirectoryPlan{
		TopicPath:  topicPath,
		TopicTitle: topicTitle,
		Entries:    entries,
	}, nil
}

func collectPubDirectoryEntries(rootPath string) ([]pubDirectoryEntry, error) {
	entries := make([]pubDirectoryEntry, 0)
	err := filepath.WalkDir(rootPath, func(path string, dirEntry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == rootPath {
			return nil
		}

		relativePath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return err
		}
		if isHiddenPubPath(relativePath) {
			if dirEntry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if dirEntry.IsDir() {
			return nil
		}
		if !dirEntry.Type().IsRegular() {
			return nil
		}

		slugDir := filepath.Dir(relativePath)
		if slugDir == "." {
			slugDir = ""
		}
		entryType := "file"
		itemSlug := buildPubDirectorySlug(path)
		if strings.EqualFold(filepath.Ext(path), ".md") {
			entryType = "md2html"
			itemSlug = ""
		}

		entries = append(entries, pubDirectoryEntry{
			FilePath: path,
			SlugDir:  filepath.ToSlash(slugDir),
			ItemSlug: itemSlug,
			Type:     entryType,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan directory: %w", err)
	}
	return entries, nil
}

func validatePubDirectoryEntries(entries []pubDirectoryEntry, topicPath string) error {
	slugSet := make(map[string]string, len(entries))
	for index := range entries {
		resolvedSlug, err := resolvePubDirectoryEntrySlug(entries[index])
		if err != nil {
			return err
		}
		entries[index].ResolvedSlug = resolvedSlug

		slugPath := buildPubDirectoryEntryValidationPath(topicPath, entries[index], resolvedSlug)
		if existingPath, exists := slugSet[slugPath]; exists {
			return fmt.Errorf("directory publish path conflict: %s and %s both map to %s", existingPath, entries[index].FilePath, slugPath)
		}
		slugSet[slugPath] = entries[index].FilePath
	}
	return nil
}

func resolvePubDirectoryEntrySlug(entry pubDirectoryEntry) (string, error) {
	if entry.ItemSlug != "" {
		return entry.ItemSlug, nil
	}
	if entry.ResolvedSlug != "" {
		return entry.ResolvedSlug, nil
	}

	createOptions, err := buildPubDirectoryCreateOptions(entry, nil, http.MethodPost)
	if err != nil {
		return "", err
	}
	return createOptions.Slug, nil
}

func isHiddenPubPath(relativePath string) bool {
	for _, part := range strings.Split(filepath.ToSlash(relativePath), "/") {
		if strings.HasPrefix(part, ".") {
			return true
		}
	}
	return false
}

func buildPubDirectorySlug(filePath string) string {
	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	return metadata.GenerateSlugFromTitle(baseName)
}

func joinPubSlugPath(topicPath string, slugDir string, itemSlug string) string {
	if slugDir == "" {
		return topicPath + "/" + itemSlug
	}
	return topicPath + "/" + slugDir + "/" + itemSlug
}

func buildPubDirectoryEntryValidationPath(topicPath string, entry pubDirectoryEntry, resolvedSlug string) string {
	slugPath := joinPubSlugPath(topicPath, entry.SlugDir, resolvedSlug)
	if entry.Type != "file" {
		return slugPath
	}

	return slugPath + filepath.Ext(entry.FilePath)
}

func buildPubDirectoryTopicURL(host string, topicPath string) string {
	return strings.TrimRight(host, "/") + "/" + topicPath
}

func writePubDirectoryConfirmPreview(
	writer io.Writer,
	parentTopic string,
	topicPath string,
	topicTitle string,
	entries []pubDirectoryEntry,
	options pubOptions,
) {
	markdownCount := 0
	fileCount := 0
	for _, entry := range entries {
		if entry.Type == "md2html" {
			markdownCount++
			continue
		}
		fileCount++
	}

	writePubConfirmField(writer, "parent topic", parentTopic)
	writePubConfirmField(writer, "topic", topicPath)
	writePubConfirmField(writer, "title", topicTitle)
	writePubConfirmField(writer, "files", strconv.Itoa(len(entries)))
	writePubConfirmField(writer, "markdown", strconv.Itoa(markdownCount))
	writePubConfirmField(writer, "binary", strconv.Itoa(fileCount))
	if options.Update {
		writePubConfirmField(writer, "mode", "overwrite")
	}
	if options.TTL != nil {
		writePubConfirmField(writer, "ttl", fmt.Sprintf("%d min", *options.TTL))
	}
	fmt.Fprintln(writer)
}

func writePubConfirmField(writer io.Writer, key string, value string) {
	fmt.Fprintf(writer, "%-12s %s\n", key, value)
}
