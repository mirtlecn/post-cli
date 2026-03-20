package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadFileMetadataUsesFrontMatterAndHeading(t *testing.T) {
	filePath := writeMetadataTestFile(t, `---
title: Front Matter Title
slug: front-matter-slug
created: 2026-03-01
---

# Heading Title
`)

	metadata, err := ReadFileMetadata(filePath, time.Date(2026, 3, 20, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ReadFileMetadata returned error: %v", err)
	}

	if metadata.Title != "Front Matter Title" || metadata.Slug != "front-matter-slug" || metadata.Created != "2026-03-01" {
		t.Fatalf("unexpected metadata: %#v", metadata)
	}
}

func TestReadFileMetadataFallsBackToFileNameAndModifiedTime(t *testing.T) {
	filePath := writeMetadataTestFile(t, "plain text\n")
	modTime := time.Date(2026, 3, 5, 6, 7, 8, 0, time.UTC)
	if err := os.Chtimes(filePath, modTime, modTime); err != nil {
		t.Fatalf("set file times: %v", err)
	}

	metadata, err := ReadFileMetadata(filePath, time.Date(2026, 3, 20, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ReadFileMetadata returned error: %v", err)
	}

	if metadata.Title != "note" {
		t.Fatalf("unexpected title: %#v", metadata)
	}
	if metadata.Slug != "" {
		t.Fatalf("unexpected slug: %#v", metadata)
	}
	if metadata.Created != modTime.Format(time.RFC3339) {
		t.Fatalf("unexpected created: %#v", metadata)
	}
}

func TestReadFileMetadataIgnoresIncompleteFrontMatter(t *testing.T) {
	filePath := writeMetadataTestFile(t, "---\ntitle: Front Matter Title\n")

	metadata, err := ReadFileMetadata(filePath, time.Date(2026, 3, 20, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ReadFileMetadata returned error: %v", err)
	}

	if metadata.Title != "note" {
		t.Fatalf("unexpected title: %#v", metadata)
	}
}

func TestReadFileMetadataFallsBackForBinaryFile(t *testing.T) {
	filePath := writeMetadataTestFileBytes(t, []byte{0x00, 0x01, 0x02})

	metadata, err := ReadFileMetadata(filePath, time.Date(2026, 3, 20, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ReadFileMetadata returned error: %v", err)
	}

	if metadata.Title != "note" {
		t.Fatalf("unexpected title: %#v", metadata)
	}
	if metadata.Slug != "" {
		t.Fatalf("unexpected slug: %#v", metadata)
	}
}

func writeMetadataTestFile(t *testing.T, content string) string {
	t.Helper()
	return writeMetadataTestFileBytes(t, []byte(content))
}

func writeMetadataTestFileBytes(t *testing.T, content []byte) string {
	t.Helper()
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "note.md")
	if err := os.WriteFile(filePath, content, 0o644); err != nil {
		t.Fatalf("write metadata file: %v", err)
	}
	return filePath
}
