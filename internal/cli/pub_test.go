package cli

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mirtle/post-cli/internal/api"
	"github.com/mirtle/post-cli/internal/config"
	"github.com/mirtle/post-cli/internal/metadata"
	"github.com/mirtle/post-cli/internal/post"
)

func TestParsePubOptions(t *testing.T) {
	options, err := parsePubOptions([]string{"-t", "60", "-s", "note", "-i", "Title", "-y", "./note.md"})
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

func TestReadMarkdownMetadataUsesFrontMatterTitleSlugAndCreated(t *testing.T) {
	filePath := writeMarkdownTestFile(t, `---
title: Front Matter Title
slug: fm-slug
created: 2026-03-01
---

# Heading Title
`)

	metadata, err := readMarkdownMetadata(filePath)
	if err != nil {
		t.Fatalf("readMarkdownMetadata returned error: %v", err)
	}

	if metadata.Title != "Front Matter Title" || metadata.Slug != "fm-slug" || metadata.Created != "2026-03-01" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
}

func TestReadMarkdownMetadataFallsBackToHeadingAndDate(t *testing.T) {
	filePath := writeMarkdownTestFile(t, `---
date: 2026-03-01T08:00:00+08:00
---

# Heading Title
`)

	metadata, err := readMarkdownMetadata(filePath)
	if err != nil {
		t.Fatalf("readMarkdownMetadata returned error: %v", err)
	}

	if metadata.Title != "Heading Title" {
		t.Fatalf("unexpected title: %#v", metadata)
	}
	if metadata.Created != "2026-03-01T08:00:00+08:00" {
		t.Fatalf("unexpected created: %#v", metadata)
	}
}

func TestReadMarkdownMetadataFallsBackToFileName(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "hello-world.md")
	if err := os.WriteFile(filePath, []byte("\ncontent\n"), 0o644); err != nil {
		t.Fatalf("write markdown file: %v", err)
	}

	metadata, err := readMarkdownMetadata(filePath)
	if err != nil {
		t.Fatalf("readMarkdownMetadata returned error: %v", err)
	}

	if metadata.Title != "hello-world" {
		t.Fatalf("unexpected title: %#v", metadata)
	}
}

func TestRunPubBuildsCreateOptionsFromMetadata(t *testing.T) {
	originalNowFunc := nowFunc
	nowFunc = func() time.Time {
		return time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		nowFunc = originalNowFunc
	})

	filePath := writeMarkdownTestFile(t, `---
title: Front Matter Title
slug: fm-slug
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
		if payload.Created != "2026-03-01T00:00:00Z" {
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

func TestRunPubGeneratesSlugFromFinalTitleAndUnixTime(t *testing.T) {
	originalNowFunc := nowFunc
	nowFunc = func() time.Time {
		return time.Unix(1760000000, 0).UTC()
	}
	t.Cleanup(func() {
		nowFunc = originalNowFunc
	})

	filePath := writeMarkdownTestFile(t, "# Hello World\n")
	service := newStubPubService(t, func(payload api.JSONRequest) {
		expectedSlug := metadata.GenerateSlugFromTitle("Hello World", 1760000000)
		if payload.Path != expectedSlug {
			t.Fatalf("unexpected payload path: %#v", payload)
		}
		if payload.Title != "Hello World" {
			t.Fatalf("unexpected payload title: %#v", payload)
		}
		if payload.Created != "2025-10-09T08:53:20Z" {
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

func TestGenerateSlugFromTitleUsesLowercaseAndUnixTime(t *testing.T) {
	got := metadata.GenerateSlugFromTitle("Hello World", 1760000000)
	if got != "hello-world-1760000000" {
		t.Fatalf("unexpected slug: %s", got)
	}
}

func TestGenerateSlugFromTitleFallsBackWhenTitleIsEmpty(t *testing.T) {
	got := metadata.GenerateSlugFromTitle("", 1760000000)
	if got != "post-1760000000" {
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
