package cli

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mirtle/post-cli/internal/api"
	"github.com/mirtle/post-cli/internal/config"
	"github.com/mirtle/post-cli/internal/metadata"
	"github.com/mirtle/post-cli/internal/post"
)

func TestParsePubOptions(t *testing.T) {
	options, err := parsePubOptions([]string{"-t", "60", "-s", "note", "-i", "Title", "-u", "-y", "./note.md"})
	if err != nil {
		t.Fatalf("parsePubOptions returned error: %v", err)
	}

	if options.FilePath != "./note.md" || options.Slug != "note" || options.Title != "Title" {
		t.Fatalf("unexpected options: %#v", options)
	}
	if options.TTL == nil || *options.TTL != 60 {
		t.Fatalf("unexpected ttl: %v", options.TTL)
	}
	if !options.SkipConfirm {
		t.Fatalf("expected skip confirm: %#v", options)
	}
	if !options.Update {
		t.Fatalf("expected update: %#v", options)
	}
}

func TestParsePubOptionsSupportsCombinedBooleanFlags(t *testing.T) {
	options, err := parsePubOptions([]string{"-yu", "./note.md"})
	if err != nil {
		t.Fatalf("parsePubOptions returned error: %v", err)
	}
	if !options.SkipConfirm || !options.Update {
		t.Fatalf("unexpected options: %#v", options)
	}
}

func TestParsePubOptionsRejectsMultiplePaths(t *testing.T) {
	_, err := parsePubOptions([]string{"a.md", "b.md"})
	if err == nil || err.Error() != "pub command accepts a single file path" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParsePubOptionsRejectsUnknownOption(t *testing.T) {
	_, err := parsePubOptions([]string{"-x", "a.md"})
	if err == nil || err.Error() != "unknown option '-x'. Try: post help" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunPubBuildsCreateOptionsFromMetadata(t *testing.T) {
	filePath := writeMarkdownTestFile(t, `---
title: Front Matter Title
slug: fm-slug
created: 2026-03-01
---

# Heading Title
`)

	var stdout bytes.Buffer
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{}, BuildInfo{})
	service := newStubPubService(t, func(payload api.JSONRequest) {
		if payload.Type != "md2html" {
			t.Fatalf("unexpected type: %#v", payload)
		}
		if payload.Path != "fm-slug" || payload.Title != "Front Matter Title" || payload.Topic != "quick-notes" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		if payload.Created != "2026-03-01" {
			t.Fatalf("unexpected created: %#v", payload)
		}
	}, nil)

	err := app.runPub(context.Background(), service, []string{"-y", filePath}, false, "https://example.com", config.Config{
		Host:     "https://example.com",
		Token:    "demo",
		PubTopic: "quick-notes",
	})
	if err != nil {
		t.Fatalf("runPub returned error: %v", err)
	}
}

func TestRunPubGeneratesSlugFromFinalTitle(t *testing.T) {
	filePath := writeMarkdownTestFile(t, "# Hello World\n")
	modTime := time.Date(2026, 3, 6, 7, 8, 9, 0, time.UTC)
	if err := os.Chtimes(filePath, modTime, modTime); err != nil {
		t.Fatalf("set file times: %v", err)
	}
	service := newStubPubService(t, func(payload api.JSONRequest) {
		expectedSlug := metadata.GenerateSlugFromTitle("Hello World")
		if payload.Path != expectedSlug {
			t.Fatalf("unexpected payload path: %#v", payload)
		}
		if payload.Title != "Hello World" {
			t.Fatalf("unexpected payload title: %#v", payload)
		}
		if payload.Created != modTime.Format(time.RFC3339) {
			t.Fatalf("unexpected created: %#v", payload)
		}
	}, nil)

	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	err := app.runPub(context.Background(), service, []string{"-y", filePath}, false, "https://example.com", config.Config{
		Host:     "https://example.com",
		Token:    "demo",
		PubTopic: "quick-notes",
	})
	if err != nil {
		t.Fatalf("runPub returned error: %v", err)
	}
}

func TestRunPubUsesPutWhenUpdateIsEnabled(t *testing.T) {
	filePath := writeMarkdownTestFile(t, "# Hello World\n")
	service := post.NewService(&stubPubClient{
		postJSONFunc: func(_ context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error) {
			if method != http.MethodPut || export {
				t.Fatalf("unexpected call: %s export=%v payload=%#v", method, export, payload)
			}
			return []byte(`{"surl":"https://sho.rt/pub"}`), nil
		},
	}, &stubCreateClipboard{}, bytes.NewBuffer(nil), &bytes.Buffer{})

	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	err := app.runPub(context.Background(), service, []string{"-yu", filePath}, false, "https://example.com", config.Config{
		Host:     "https://example.com",
		Token:    "demo",
		PubTopic: "quick-notes",
	})
	if err != nil {
		t.Fatalf("runPub returned error: %v", err)
	}
}

func TestRunPubAllowsOverrides(t *testing.T) {
	filePath := writeMarkdownTestFile(t, "# Heading Title\n")
	service := newStubPubService(t, func(payload api.JSONRequest) {
		if payload.Path != "manual-slug" || payload.Title != "Manual Title" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
		if payload.TTL == nil || *payload.TTL != 45 {
			t.Fatalf("unexpected ttl: %#v", payload)
		}
	}, nil)

	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	err := app.runPub(context.Background(), service, []string{"-y", "-t", "45", "-s", "manual-slug", "-i", "Manual Title", filePath}, false, "https://example.com", config.Config{
		Host:     "https://example.com",
		Token:    "demo",
		PubTopic: "quick-notes",
	})
	if err != nil {
		t.Fatalf("runPub returned error: %v", err)
	}
}

func TestGenerateSlugFromTitleUsesLowercase(t *testing.T) {
	got := metadata.GenerateSlugFromTitle("Hello World")
	if got != "hello-world" {
		t.Fatalf("unexpected slug: %s", got)
	}
}

func TestGenerateSlugFromTitleFallsBackWhenTitleIsEmpty(t *testing.T) {
	got := metadata.GenerateSlugFromTitle("")
	if got != "post" {
		t.Fatalf("unexpected slug: %s", got)
	}
}

func TestRunPubFailsWithoutTopic(t *testing.T) {
	filePath := writeMarkdownTestFile(t, "# Heading Title\n")
	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	err := app.runPub(context.Background(), newStubPubService(t, nil, nil), []string{filePath}, false, "https://example.com", config.Config{
		Host:  "https://example.com",
		Token: "demo",
	})
	if err == nil || err.Error() != "POST_PUB_TOPIC or pub_topic must be set for post pub" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunPubPassesThroughInvalidSlug(t *testing.T) {
	filePath := writeMarkdownTestFile(t, `---
slug: 中文 slug
---

# Heading Title
`)

	service := newStubPubService(t, func(payload api.JSONRequest) {
		if payload.Path != "中文 slug" {
			t.Fatalf("unexpected payload: %#v", payload)
		}
	}, func() error {
		return nil
	})

	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	err := app.runPub(context.Background(), service, []string{filePath}, false, "https://example.com", config.Config{
		Host:     "https://example.com",
		Token:    "demo",
		PubTopic: "quick-notes",
	})
	if err != nil {
		t.Fatalf("runPub returned error: %v", err)
	}
}

func TestRunPubDirectoryCreatesChildTopicAndUploadsFiles(t *testing.T) {
	rootDir := t.TempDir()
	markdownPath := filepath.Join(rootDir, "index.md")
	imagePath := filepath.Join(rootDir, "assets", "logo.png")
	nestedMarkdownPath := filepath.Join(rootDir, "nested", "guide.md")
	deepMarkdownPath := filepath.Join(rootDir, "nested", "deep", "guide-2.md")
	multiLevelFilePath := filepath.Join(rootDir, "nested", "deep", "files", "report.pdf")
	hiddenMarkdownPath := filepath.Join(rootDir, ".secret", "hidden.md")
	if err := os.MkdirAll(filepath.Dir(imagePath), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(nestedMarkdownPath), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(deepMarkdownPath), 0o755); err != nil {
		t.Fatalf("mkdir deep nested: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(multiLevelFilePath), 0o755); err != nil {
		t.Fatalf("mkdir multi-level file dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(hiddenMarkdownPath), 0o755); err != nil {
		t.Fatalf("mkdir hidden: %v", err)
	}
	if err := os.WriteFile(markdownPath, []byte("# Home\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	if err := os.WriteFile(imagePath, []byte("png"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	if err := os.WriteFile(nestedMarkdownPath, []byte("---\nslug: custom-guide\n---\n\n# Guide\n"), 0o644); err != nil {
		t.Fatalf("write nested markdown: %v", err)
	}
	if err := os.WriteFile(deepMarkdownPath, []byte("# Deep Guide\n"), 0o644); err != nil {
		t.Fatalf("write deep markdown: %v", err)
	}
	if err := os.WriteFile(multiLevelFilePath, []byte("pdf"), 0o644); err != nil {
		t.Fatalf("write multi-level file: %v", err)
	}
	if err := os.WriteFile(hiddenMarkdownPath, []byte("# Hidden\n"), 0o644); err != nil {
		t.Fatalf("write hidden markdown: %v", err)
	}

	client := &recordingPubClient{}
	service := post.NewService(client, &stubCreateClipboard{}, bytes.NewBuffer(nil), &bytes.Buffer{})
	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	err := app.runPub(context.Background(), service, []string{"-yu", rootDir}, false, "https://example.com", config.Config{
		Host:     "https://example.com",
		Token:    "demo",
		PubTopic: "notes",
	})
	if err != nil {
		t.Fatalf("runPub returned error: %v", err)
	}

	if len(client.postJSONCalls) != 4 {
		t.Fatalf("unexpected post JSON call count: %d", len(client.postJSONCalls))
	}
	topicCall := client.postJSONCalls[0]
	if topicCall.method != http.MethodPost || topicCall.payload.Type != "topic" || topicCall.payload.Path != "notes/"+filepath.Base(rootDir) {
		t.Fatalf("unexpected topic call: %#v", topicCall)
	}

	itemCalls := append([]recordingPostJSONCall(nil), client.postJSONCalls[1:]...)
	sort.Slice(itemCalls, func(i, j int) bool {
		return itemCalls[i].payload.Path < itemCalls[j].payload.Path
	})
	for _, itemCall := range itemCalls {
		if itemCall.method != http.MethodPut {
			t.Fatalf("expected PUT calls for markdown items: %#v", itemCalls)
		}
	}
	expectedMarkdownPaths := map[string]bool{
		"notes/" + filepath.Base(rootDir) + "/home":                   false,
		"notes/" + filepath.Base(rootDir) + "/nested/custom-guide":    false,
		"notes/" + filepath.Base(rootDir) + "/nested/deep/deep-guide": false,
	}
	for _, itemCall := range itemCalls {
		if _, exists := expectedMarkdownPaths[itemCall.payload.Path]; exists {
			expectedMarkdownPaths[itemCall.payload.Path] = true
		}
	}
	for expectedPath, seen := range expectedMarkdownPaths {
		if !seen {
			t.Fatalf("missing markdown upload for %s: %#v", expectedPath, itemCalls)
		}
	}

	if len(client.uploadFileCalls) != 2 {
		t.Fatalf("unexpected upload file call count: %d", len(client.uploadFileCalls))
	}
	expectedFilePaths := map[string]bool{
		"notes/" + filepath.Base(rootDir) + "/assets/logo":              false,
		"notes/" + filepath.Base(rootDir) + "/nested/deep/files/report": false,
	}
	for _, uploadCall := range client.uploadFileCalls {
		if uploadCall.method != http.MethodPut {
			t.Fatalf("unexpected upload file method: %#v", uploadCall)
		}
		if _, exists := expectedFilePaths[uploadCall.slug]; exists {
			expectedFilePaths[uploadCall.slug] = true
		}
	}
	for expectedPath, seen := range expectedFilePaths {
		if !seen {
			t.Fatalf("missing file upload for %s: %#v", expectedPath, client.uploadFileCalls)
		}
	}
}

func TestRunPubDirectoryUsesCustomChildSlug(t *testing.T) {
	rootDir := t.TempDir()
	filePath := filepath.Join(rootDir, "index.md")
	if err := os.WriteFile(filePath, []byte("# Home\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	client := &recordingPubClient{}
	service := post.NewService(client, &stubCreateClipboard{}, bytes.NewBuffer(nil), &bytes.Buffer{})
	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	err := app.runPub(context.Background(), service, []string{"-y", "-s", "week-12", rootDir}, false, "https://example.com", config.Config{
		Host:     "https://example.com",
		Token:    "demo",
		PubTopic: "notes",
	})
	if err != nil {
		t.Fatalf("runPub returned error: %v", err)
	}
	if len(client.postJSONCalls) == 0 || client.postJSONCalls[0].payload.Path != "notes/week-12" {
		t.Fatalf("unexpected topic path: %#v", client.postJSONCalls)
	}
}

func TestPlanPubDirectoryRejectsConflictingSlugs(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootDir, "hello!.md"), []byte("# Hello\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "hello?.md"), []byte("# Hello\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	_, err := planPubDirectory(rootDir, "notes/demo", "Demo")
	if err == nil || !strings.Contains(err.Error(), "directory publish path conflict") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanPubDirectoryAllowsFilesWithSameBaseNameDifferentExtensions(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(rootDir, "report.pdf"), []byte("pdf"), 0o644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "report.png"), []byte("png"), 0o644); err != nil {
		t.Fatalf("write png: %v", err)
	}

	plan, err := planPubDirectory(rootDir, "notes/demo", "Demo")
	if err != nil {
		t.Fatalf("planPubDirectory returned error: %v", err)
	}
	if len(plan.Entries) != 2 {
		t.Fatalf("unexpected entry count: %d", len(plan.Entries))
	}
}

func TestCollectPubDirectoryEntriesSkipsHiddenPaths(t *testing.T) {
	rootDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(rootDir, ".secret"), 0o755); err != nil {
		t.Fatalf("mkdir hidden: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(rootDir, "visible"), 0o755); err != nil {
		t.Fatalf("mkdir visible: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, ".hidden.md"), []byte("# hidden\n"), 0o644); err != nil {
		t.Fatalf("write hidden file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, ".secret", "nested.md"), []byte("# hidden\n"), 0o644); err != nil {
		t.Fatalf("write hidden nested file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "visible", "note.md"), []byte("# visible\n"), 0o644); err != nil {
		t.Fatalf("write visible file: %v", err)
	}

	entries, err := collectPubDirectoryEntries(rootDir)
	if err != nil {
		t.Fatalf("collectPubDirectoryEntries returned error: %v", err)
	}
	if len(entries) != 1 || !strings.HasSuffix(entries[0].FilePath, filepath.Join("visible", "note.md")) {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestCollectPubDirectoryEntriesSupportsNestedDirectories(t *testing.T) {
	rootDir := t.TempDir()
	rootFile := filepath.Join(rootDir, "root.md")
	singleLevelFile := filepath.Join(rootDir, "docs", "guide.md")
	doubleLevelFile := filepath.Join(rootDir, "docs", "v1", "guide.md")
	multiLevelFile := filepath.Join(rootDir, "docs", "v1", "deep", "guide.md")

	for _, path := range []string{rootFile, singleLevelFile, doubleLevelFile, multiLevelFile} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", path, err)
		}
		content := "# Guide\n"
		if path == rootFile {
			content = "# Root\n"
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	plan, err := planPubDirectory(rootDir, "notes/demo", "Demo")
	if err != nil {
		t.Fatalf("planPubDirectory returned error: %v", err)
	}

	gotPaths := make([]string, 0, len(plan.Entries))
	for _, entry := range plan.Entries {
		gotPaths = append(gotPaths, joinPubSlugPath(plan.TopicPath, entry.SlugDir, entry.ResolvedSlug))
	}
	sort.Strings(gotPaths)

	expectedPaths := []string{
		"notes/demo/docs/guide",
		"notes/demo/docs/v1/deep/guide",
		"notes/demo/docs/v1/guide",
		"notes/demo/root",
	}
	if strings.Join(gotPaths, "\n") != strings.Join(expectedPaths, "\n") {
		t.Fatalf("unexpected nested paths:\n got: %v\nwant: %v", gotPaths, expectedPaths)
	}
}

func TestRunPubDirectoryPropagatesTTLToAllEntries(t *testing.T) {
	rootDir := t.TempDir()
	markdownPath := filepath.Join(rootDir, "index.md")
	filePath := filepath.Join(rootDir, "assets", "logo.png")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(markdownPath, []byte("# Home\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("png"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	client := &recordingPubClient{}
	service := post.NewService(client, &stubCreateClipboard{}, bytes.NewBuffer(nil), &bytes.Buffer{})
	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	err := app.runPub(context.Background(), service, []string{"-y", "-u", "-t", "30", rootDir}, false, "https://example.com", config.Config{
		Host:     "https://example.com",
		Token:    "demo",
		PubTopic: "notes",
	})
	if err != nil {
		t.Fatalf("runPub returned error: %v", err)
	}

	for _, itemCall := range client.postJSONCalls[1:] {
		if itemCall.payload.TTL == nil || *itemCall.payload.TTL != 30 {
			t.Fatalf("expected markdown ttl propagation: %#v", client.postJSONCalls)
		}
	}
	for _, uploadCall := range client.uploadFileCalls {
		if uploadCall.ttl == nil || *uploadCall.ttl != 30 {
			t.Fatalf("expected file ttl propagation: %#v", client.uploadFileCalls)
		}
	}
}

func TestRunPubDirectoryCreatesTopicForEmptyDirectory(t *testing.T) {
	rootDir := t.TempDir()

	client := &recordingPubClient{}
	service := post.NewService(client, &stubCreateClipboard{}, bytes.NewBuffer(nil), &bytes.Buffer{})
	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	err := app.runPub(context.Background(), service, []string{"-y", rootDir}, false, "https://example.com", config.Config{
		Host:     "https://example.com",
		Token:    "demo",
		PubTopic: "notes",
	})
	if err != nil {
		t.Fatalf("runPub returned error: %v", err)
	}
	if len(client.postJSONCalls) != 1 || client.postJSONCalls[0].payload.Type != "topic" {
		t.Fatalf("expected only topic creation call: %#v", client.postJSONCalls)
	}
	if len(client.uploadFileCalls) != 0 {
		t.Fatalf("did not expect file uploads: %#v", client.uploadFileCalls)
	}
}

func TestRunPubDirectoryCreatesTopicForHiddenOnlyDirectory(t *testing.T) {
	rootDir := t.TempDir()
	hiddenFile := filepath.Join(rootDir, ".secret", "note.md")
	if err := os.MkdirAll(filepath.Dir(hiddenFile), 0o755); err != nil {
		t.Fatalf("mkdir hidden: %v", err)
	}
	if err := os.WriteFile(hiddenFile, []byte("# Hidden\n"), 0o644); err != nil {
		t.Fatalf("write hidden file: %v", err)
	}

	client := &recordingPubClient{}
	service := post.NewService(client, &stubCreateClipboard{}, bytes.NewBuffer(nil), &bytes.Buffer{})
	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})
	err := app.runPub(context.Background(), service, []string{"-y", rootDir}, false, "https://example.com", config.Config{
		Host:     "https://example.com",
		Token:    "demo",
		PubTopic: "notes",
	})
	if err != nil {
		t.Fatalf("runPub returned error: %v", err)
	}
	if len(client.postJSONCalls) != 1 || client.postJSONCalls[0].payload.Type != "topic" {
		t.Fatalf("expected only topic creation call: %#v", client.postJSONCalls)
	}
	if len(client.uploadFileCalls) != 0 {
		t.Fatalf("did not expect file uploads: %#v", client.uploadFileCalls)
	}
}

func TestRunPubDirectorySkipsCreateWhenTopicExists(t *testing.T) {
	rootDir := t.TempDir()
	filePath := filepath.Join(rootDir, "index.md")
	if err := os.WriteFile(filePath, []byte("# Home\n"), 0o644); err != nil {
		t.Fatalf("write markdown: %v", err)
	}

	client := &recordingPubClient{
		getFunc: func(_ context.Context, payload api.JSONRequest, export bool) ([]byte, error) {
			if payload.Path != "notes/"+filepath.Base(rootDir) || payload.Type != "topic" || !export {
				t.Fatalf("unexpected topic lookup: %#v export=%v", payload, export)
			}
			return []byte(fmt.Sprintf(`{"path":"%s","type":"topic","title":"Demo"}`, payload.Path)), nil
		},
	}
	service := post.NewService(client, &stubCreateClipboard{}, bytes.NewBuffer(nil), &bytes.Buffer{})
	var stdout bytes.Buffer
	app := NewApp(os.Stdin, &stdout, &bytes.Buffer{}, BuildInfo{})
	err := app.runPub(context.Background(), service, []string{"-yu", rootDir}, false, "https://example.com", config.Config{
		Host:     "https://example.com",
		Token:    "demo",
		PubTopic: "notes",
	})
	if err != nil {
		t.Fatalf("runPub returned error: %v", err)
	}
	if len(client.getCalls) != 1 {
		t.Fatalf("unexpected get call count: %d", len(client.getCalls))
	}
	if len(client.postJSONCalls) != 1 {
		t.Fatalf("unexpected post JSON call count: %d", len(client.postJSONCalls))
	}
	if client.postJSONCalls[0].payload.Type == "topic" {
		t.Fatalf("did not expect topic creation call: %#v", client.postJSONCalls)
	}
	if got := stdout.String(); got != "https://example.com/notes/"+filepath.Base(rootDir)+"\n" {
		t.Fatalf("unexpected stdout: %q", got)
	}
}

func TestRunPubDirectoryCancelsPendingUploadsAfterFailure(t *testing.T) {
	rootDir := t.TempDir()
	paths := []string{
		filepath.Join(rootDir, "a.md"),
		filepath.Join(rootDir, "b.md"),
		filepath.Join(rootDir, "c.md"),
		filepath.Join(rootDir, "d.md"),
		filepath.Join(rootDir, "e.md"),
		filepath.Join(rootDir, "f.md"),
	}
	for _, path := range paths {
		if err := os.WriteFile(path, []byte("# "+strings.TrimSuffix(filepath.Base(path), ".md")+"\n"), 0o644); err != nil {
			t.Fatalf("write markdown %s: %v", path, err)
		}
	}

	client := &recordingPubClient{
		postJSONFunc: func(ctx context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error) {
			if payload.Type == "topic" {
				return []byte(`{"surl":"https://sho.rt/topic"}`), nil
			}
			if strings.HasSuffix(payload.Path, "/c") {
				return nil, fmt.Errorf("boom")
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(50 * time.Millisecond):
			}
			return []byte(`{"surl":"https://sho.rt/item"}`), nil
		},
	}
	service := post.NewService(client, &stubCreateClipboard{}, bytes.NewBuffer(nil), &bytes.Buffer{})
	app := NewApp(os.Stdin, &bytes.Buffer{}, &bytes.Buffer{}, BuildInfo{})

	err := app.runPub(context.Background(), service, []string{"-yu", rootDir}, false, "https://example.com", config.Config{
		Host:     "https://example.com",
		Token:    "demo",
		PubTopic: "notes",
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.postJSONCalls) >= len(paths)+1 {
		t.Fatalf("expected cancellation to stop some uploads, got %d calls", len(client.postJSONCalls))
	}
}

func writeMarkdownTestFile(t *testing.T, content string) string {
	t.Helper()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "note.md")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write markdown file: %v", err)
	}
	return filePath
}

type stubPubClient struct {
	postJSONFunc func(ctx context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error)
}

func (client *stubPubClient) PostJSON(ctx context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error) {
	return client.postJSONFunc(ctx, method, payload, export)
}

func (client *stubPubClient) Get(context.Context, api.JSONRequest, bool) ([]byte, error) {
	panic("unexpected Get call")
}

func (client *stubPubClient) Delete(context.Context, api.JSONRequest, bool) ([]byte, error) {
	panic("unexpected Delete call")
}

func (client *stubPubClient) UploadFile(context.Context, string, string, string, string, string, string, *int, bool) ([]byte, error) {
	panic("unexpected UploadFile call")
}

func newStubPubService(t *testing.T, assertPayload func(payload api.JSONRequest), afterCall func() error) *post.Service {
	t.Helper()
	return post.NewService(&stubPubClient{
		postJSONFunc: func(_ context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error) {
			if method != http.MethodPost || export {
				t.Fatalf("unexpected call: %s export=%v", method, export)
			}
			if assertPayload != nil {
				assertPayload(payload)
			}
			if afterCall != nil {
				if err := afterCall(); err != nil {
					return nil, err
				}
			}
			return []byte(`{"surl":"https://sho.rt/pub"}`), nil
		},
	}, &stubCreateClipboard{}, bytes.NewBuffer(nil), &bytes.Buffer{})
}

type recordingPostJSONCall struct {
	method  string
	payload api.JSONRequest
	export  bool
}

type recordingUploadFileCall struct {
	method   string
	filePath string
	slug     string
	title    string
	topic    string
	created  string
	ttl      *int
	export   bool
}

type recordingPubClient struct {
	postJSONCalls   []recordingPostJSONCall
	uploadFileCalls []recordingUploadFileCall
	getCalls        []api.JSONRequest
	postJSONFunc    func(ctx context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error)
	getFunc         func(ctx context.Context, payload api.JSONRequest, export bool) ([]byte, error)
	mu              sync.Mutex
}

func (client *recordingPubClient) PostJSON(ctx context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error) {
	client.mu.Lock()
	client.postJSONCalls = append(client.postJSONCalls, recordingPostJSONCall{
		method:  method,
		payload: payload,
		export:  export,
	})
	postJSONFunc := client.postJSONFunc
	client.mu.Unlock()
	if postJSONFunc != nil {
		return postJSONFunc(ctx, method, payload, export)
	}
	return []byte(`{"surl":"https://sho.rt/pub"}`), nil
}

func (client *recordingPubClient) Get(ctx context.Context, payload api.JSONRequest, export bool) ([]byte, error) {
	client.mu.Lock()
	client.getCalls = append(client.getCalls, payload)
	getFunc := client.getFunc
	client.mu.Unlock()
	if getFunc != nil {
		return getFunc(ctx, payload, export)
	}
	return nil, fmt.Errorf("HTTP 404: URL not found")
}

func (client *recordingPubClient) Delete(context.Context, api.JSONRequest, bool) ([]byte, error) {
	panic("unexpected Delete call")
}

func (client *recordingPubClient) UploadFile(_ context.Context, method string, filePath string, slug string, title string, topic string, created string, ttl *int, export bool) ([]byte, error) {
	client.mu.Lock()
	client.uploadFileCalls = append(client.uploadFileCalls, recordingUploadFileCall{
		method:   method,
		filePath: filePath,
		slug:     slug,
		title:    title,
		topic:    topic,
		created:  created,
		ttl:      ttl,
		export:   export,
	})
	client.mu.Unlock()
	return []byte(`{"surl":"https://sho.rt/file"}`), nil
}
