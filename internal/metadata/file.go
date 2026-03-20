package metadata

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const metadataProbeLimit = 4 * 1024

type FileMetadata struct {
	Title   string
	Slug    string
	Created string
}

type frontMatterMetadata struct {
	Title   string `yaml:"title"`
	Slug    string `yaml:"slug"`
	Created string `yaml:"created"`
	Date    string `yaml:"date"`
}

func ReadFileMetadata(filePath string, now time.Time) (FileMetadata, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return FileMetadata{}, fmt.Errorf("stat file metadata: %w", err)
	}

	probe, err := readProbeBytes(filePath, metadataProbeLimit)
	if err != nil {
		return FileMetadata{}, fmt.Errorf("read file metadata: %w", err)
	}

	metadata := FileMetadata{
		Title:   strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)),
		Created: now.UTC().Format(time.RFC3339),
	}
	if !fileInfo.ModTime().IsZero() {
		metadata.Created = fileInfo.ModTime().UTC().Format(time.RFC3339)
	}

	if len(probe) == 0 || isLikelyBinary(probe) {
		return metadata, nil
	}

	frontMatter, body, err := parseFrontMatterProbe(probe)
	if err != nil {
		return FileMetadata{}, err
	}

	if frontMatter.Title != "" {
		metadata.Title = frontMatter.Title
	} else if heading := extractFirstHeading(body); heading != "" {
		metadata.Title = heading
	}

	metadata.Slug = frontMatter.Slug

	switch {
	case frontMatter.Created != "":
		metadata.Created = frontMatter.Created
	case frontMatter.Date != "":
		metadata.Created = frontMatter.Date
	}

	return metadata, nil
}

func readProbeBytes(filePath string, limit int64) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	probe, err := io.ReadAll(io.LimitReader(file, limit))
	if err != nil {
		return nil, err
	}
	return probe, nil
}

func isLikelyBinary(content []byte) bool {
	return bytes.IndexByte(content, 0) >= 0
}

func parseFrontMatterProbe(content []byte) (frontMatterMetadata, []byte, error) {
	trimmedContent := bytes.TrimPrefix(content, []byte("\xef\xbb\xbf"))
	if !bytes.HasPrefix(trimmedContent, []byte("---\n")) && !bytes.Equal(trimmedContent, []byte("---")) && !bytes.HasPrefix(trimmedContent, []byte("---\r\n")) {
		return frontMatterMetadata{}, trimmedContent, nil
	}

	lines := splitLines(trimmedContent)
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return frontMatterMetadata{}, trimmedContent, nil
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
		return frontMatterMetadata{}, trimmedContent, nil
	}

	var metadata frontMatterMetadata
	frontMatter := strings.Join(lines[1:closingIndex], "\n")
	if err := yaml.Unmarshal([]byte(frontMatter), &metadata); err != nil {
		return frontMatterMetadata{}, nil, fmt.Errorf("parse front matter: %w", err)
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
