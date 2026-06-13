package cli

import (
	"context"
	"fmt"
	"io"
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
const pubUsage = "usage: post pub [-p|--topic <topic>] [-t|--ttl <minutes>] [-s|--slug <path>] [-i|--title <title>] [-u|--update] [-y|--no-confirm] <path>"

type pubOptions struct {
	FilePath    string
	Slug        string
	Title       string
	Topic       string
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
		case "-p", "--topic":
			value, nextIndex, err := nextValue(expandedArgs, index)
			if err != nil {
				return pubOptions{}, fmt.Errorf("option %s requires a value", arg)
			}
			options.Topic = value
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
				return pubOptions{}, fmt.Errorf(pubUsage)
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
		return pubOptions{}, fmt.Errorf(pubUsage)
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

	topic, explicitTopic, err := resolvePubTopic(options, cfg)
	if err != nil {
		return err
	}

	fileInfo, err := os.Stat(options.FilePath)
	if err != nil {
		return fmt.Errorf("file not found: %s", options.FilePath)
	}

	if fileInfo.IsDir() {
		return app.runPubDirectory(ctx, service, options, stdinTTY, host, topic, explicitTopic)
	}

	if explicitTopic {
		if _, _, err := ensurePubTopic(ctx, service, topic, defaultPubTopicTitle(topic)); err != nil {
			return err
		}
	}

	return app.runCreate(ctx, service, post.NewOptions{
		FilePath:    options.FilePath,
		Slug:        options.Slug,
		Title:       options.Title,
		Topic:       topic,
		TTL:         options.TTL,
		Type:        "md2html",
		Update:      options.Update,
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
	topic string,
	explicitTopic bool,
) error {
	directoryName, err := resolvePubDirectoryName(options.FilePath)
	if err != nil {
		return err
	}

	if explicitTopic && options.Slug != "" {
		return fmt.Errorf("--slug is not supported for directory publish when --topic is set")
	}

	topicPath, topicTitle, parentTopic := resolvePubDirectoryTopic(options, topic, explicitTopic, directoryName)
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

	topicCreated, topicResult, err := ensurePubTopic(ctx, service, plan.TopicPath, plan.TopicTitle)
	if err != nil {
		return err
	}

	if err := app.uploadPubDirectoryEntries(ctx, service, plan.Entries, plan.TopicPath, options.TTL, options.Update); err != nil {
		return err
	}

	if topicCreated {
		app.writeCreateResult(topicResult)
		return nil
	}
	app.writeCreateResult(post.Result{Stdout: buildPubDirectoryTopicURL(host, plan.TopicPath) + "\n"})
	return nil
}

func resolvePubTopic(options pubOptions, cfg config.Config) (string, bool, error) {
	if options.Topic != "" {
		return options.Topic, true, nil
	}
	if cfg.PubTopic != "" {
		return cfg.PubTopic, false, nil
	}
	return "", false, fmt.Errorf("-p/--topic, POST_PUB_TOPIC, or pub_topic must be set for post pub")
}

func resolvePubDirectoryTopic(options pubOptions, topic string, explicitTopic bool, directoryName string) (string, string, string) {
	if explicitTopic {
		title := options.Title
		if title == "" {
			title = defaultPubTopicTitle(topic)
		}
		return topic, title, ""
	}

	childSlug := options.Slug
	if childSlug == "" {
		childSlug = metadata.GenerateSlugFromTitle(directoryName)
	}
	topicTitle := options.Title
	if topicTitle == "" {
		topicTitle = directoryName
	}
	return topic + "/" + childSlug, topicTitle, topic
}

func ensurePubTopic(ctx context.Context, service *post.Service, topicPath string, topicTitle string) (bool, post.Result, error) {
	topicExists, err := service.TopicExists(ctx, topicPath)
	if err != nil {
		return false, post.Result{}, err
	}
	if topicExists {
		return false, post.Result{}, nil
	}

	topicResult, err := service.New(ctx, post.NewOptions{
		Slug:        topicPath,
		Title:       topicTitle,
		Type:        "topic",
		SkipConfirm: true,
	})
	if err != nil {
		return false, post.Result{}, err
	}
	return true, topicResult, nil
}

func (app *App) uploadPubDirectoryEntries(
	ctx context.Context,
	service *post.Service,
	entries []pubDirectoryEntry,
	topicPath string,
	ttl *int,
	update bool,
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
				if err := uploadPubDirectoryEntry(ctx, service, entry, topicPath, ttl, update); err != nil {
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
	update bool,
) error {
	createOptions, err := buildPubDirectoryCreateOptions(entry, ttl, update)
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

func buildPubDirectoryCreateOptions(entry pubDirectoryEntry, ttl *int, update bool) (post.NewOptions, error) {
	createOptions := post.NewOptions{
		FilePath:    entry.FilePath,
		TTL:         ttl,
		Type:        entry.Type,
		Update:      update,
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

	createOptions, err := buildPubDirectoryCreateOptions(entry, nil, false)
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

func defaultPubTopicTitle(topicPath string) string {
	trimmedTopicPath := strings.Trim(topicPath, "/")
	if trimmedTopicPath == "" {
		return topicPath
	}
	parts := strings.Split(trimmedTopicPath, "/")
	return parts[len(parts)-1]
}

func resolvePubDirectoryName(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	absolutePath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("resolve directory path: %w", err)
	}

	return filepath.Base(absolutePath), nil
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

	if parentTopic != "" {
		writePubConfirmField(writer, "parent topic", parentTopic)
	}
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
