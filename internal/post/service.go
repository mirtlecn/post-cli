package post

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/mirtle/post-cli/internal/api"
	"github.com/mirtle/post-cli/internal/clipboard"
)

type APIClient interface {
	PostJSON(ctx context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error)
	Get(ctx context.Context, payload api.JSONRequest, export bool) ([]byte, error)
	Delete(ctx context.Context, payload api.JSONRequest, export bool) ([]byte, error)
	UploadFile(ctx context.Context, method string, filePath string, slug string, title string, topic string, ttl *int, export bool) ([]byte, error)
}

type Service struct {
	client    APIClient
	clipboard clipboard.Service
	stdin     io.Reader
	stderr    io.Writer
}

type NewOptions struct {
	Slug           string
	Title          string
	Topic          string
	TTL            *int
	SkipConfirm    bool
	ReadClipboard  bool
	WriteClipboard bool
	FilePath       string
	Type           string
	Method         string
	Export         bool
	Args           []string
	StdinTTY       bool
	Confirm        func(label string) (bool, error)
}

type Result struct {
	Stdout string
	Stderr string
}

type createResponse struct {
	ShortURL string `json:"surl"`
}

var uriSchemePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*$`)

func NewService(client APIClient, clipboardService clipboard.Service, stdin io.Reader, stderr io.Writer) *Service {
	return &Service{
		client:    client,
		clipboard: clipboardService,
		stdin:     stdin,
		stderr:    stderr,
	}
}

func (service *Service) New(ctx context.Context, options NewOptions) (Result, error) {
	if err := validateNewOptions(options); err != nil {
		return Result{}, err
	}

	if isTopicCreation(options) {
		return service.createTopicFromNew(ctx, options)
	}

	content, label, err := service.resolveContent(options)
	if err != nil {
		return Result{}, err
	}

	requestType := resolveRequestType(options.Type)
	if err := validateURLContent(requestType, content); err != nil {
		return Result{}, err
	}

	if !options.SkipConfirm && options.StdinTTY && options.Confirm != nil {
		writeConfirmPreview(service.stderr, label, content, options, requestType)
		accepted, confirmErr := options.Confirm(label)
		if confirmErr != nil {
			return Result{}, confirmErr
		}
		if !accepted {
			return Result{Stderr: "Aborted.\n"}, nil
		}
	}

	var responseBody []byte
	if options.Type == "file" {
		responseBody, err = service.client.UploadFile(
			ctx,
			options.Method,
			options.FilePath,
			options.Slug,
			options.Title,
			options.Topic,
			options.TTL,
			options.Export,
		)
	} else {
		payload := api.JSONRequest{
			Path:  options.Slug,
			URL:   content,
			Title: options.Title,
			Topic: options.Topic,
			TTL:   options.TTL,
			Type:  requestType,
		}
		responseBody, err = service.client.PostJSON(ctx, options.Method, payload, options.Export)
	}
	if err != nil {
		return Result{}, err
	}

	var response createResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return Result{}, fmt.Errorf("parse create response: %w", err)
	}
	if response.ShortURL == "" {
		return Result{}, fmt.Errorf("no URL returned from API; response: %s", string(responseBody))
	}

	result := Result{}
	if options.WriteClipboard {
		if !service.clipboard.CanWriteText() {
			result.Stderr += "warning: clipboard write is unavailable\n"
		} else if err := service.clipboard.WriteText(response.ShortURL); err != nil {
			result.Stderr += fmt.Sprintf("warning: failed to copy to clipboard: %s\n", err)
		} else {
			result.Stderr += fmt.Sprintf("Copied to clipboard: %s\n", response.ShortURL)
		}
	}

	if options.Export {
		formatted, formatErr := formatJSON(responseBody)
		if formatErr != nil {
			return Result{}, formatErr
		}
		result.Stdout = formatted
		return result, nil
	}

	result.Stdout = response.ShortURL + "\n"
	return result, nil
}

func (service *Service) createTopicFromNew(ctx context.Context, options NewOptions) (Result, error) {
	if !options.SkipConfirm && options.StdinTTY && options.Confirm != nil {
		writeConfirmPreview(service.stderr, "", "", options, "topic")
		accepted, confirmErr := options.Confirm("")
		if confirmErr != nil {
			return Result{}, confirmErr
		}
		if !accepted {
			return Result{Stderr: "Aborted.\n"}, nil
		}
	}

	responseBody, err := service.client.PostJSON(ctx, options.Method, api.JSONRequest{
		Path: options.Slug,
		Type: "topic",
	}, options.Export)
	if err != nil {
		return Result{}, err
	}

	var response createResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return Result{}, fmt.Errorf("parse create response: %w", err)
	}
	if response.ShortURL == "" {
		return Result{}, fmt.Errorf("no URL returned from API; response: %s", string(responseBody))
	}

	result := Result{}
	if options.WriteClipboard {
		if !service.clipboard.CanWriteText() {
			result.Stderr += "warning: clipboard write is unavailable\n"
		} else if err := service.clipboard.WriteText(response.ShortURL); err != nil {
			result.Stderr += fmt.Sprintf("warning: failed to copy to clipboard: %s\n", err)
		} else {
			result.Stderr += fmt.Sprintf("Copied to clipboard: %s\n", response.ShortURL)
		}
	}

	if options.Export {
		formatted, formatErr := formatJSON(responseBody)
		if formatErr != nil {
			return Result{}, formatErr
		}
		result.Stdout = formatted
		return result, nil
	}

	result.Stdout = response.ShortURL + "\n"
	return result, nil
}

func (service *Service) List(ctx context.Context, path string, export bool) (string, error) {
	body, err := service.client.Get(ctx, api.JSONRequest{Path: path}, export)
	if err != nil {
		return "", err
	}
	return formatJSON(body)
}

func (service *Service) Export(ctx context.Context, path string) (string, error) {
	return service.List(ctx, path, true)
}

func (service *Service) Remove(ctx context.Context, path string, export bool) (string, error) {
	body, err := service.client.Delete(ctx, api.JSONRequest{Path: path}, export)
	if err != nil {
		return "", err
	}
	return formatJSON(body)
}

func (service *Service) ListTopics(ctx context.Context, path string, export bool) (string, error) {
	payload := api.JSONRequest{Type: "topic"}
	if path != "" {
		payload.Path = path
	}
	body, err := service.client.Get(ctx, payload, export)
	if err != nil {
		return "", err
	}
	return formatJSON(body)
}

func (service *Service) CreateTopic(ctx context.Context, path string, export bool) (string, error) {
	body, err := service.client.PostJSON(ctx, http.MethodPost, api.JSONRequest{
		Path: path,
		Type: "topic",
	}, export)
	if err != nil {
		return "", err
	}
	return formatJSON(body)
}

func (service *Service) RefreshTopic(ctx context.Context, path string, export bool) (string, error) {
	body, err := service.client.PostJSON(ctx, http.MethodPut, api.JSONRequest{
		Path: path,
		Type: "topic",
	}, export)
	if err != nil {
		return "", err
	}
	return formatJSON(body)
}

func (service *Service) RemoveTopic(ctx context.Context, path string, export bool) (string, error) {
	body, err := service.client.Delete(ctx, api.JSONRequest{
		Path: path,
		Type: "topic",
	}, export)
	if err != nil {
		return "", err
	}
	return formatJSON(body)
}

func (service *Service) resolveContent(options NewOptions) (string, string, error) {
	if options.Type == "file" {
		if options.FilePath == "" {
			return "", "", fmt.Errorf("--type file requires -f <path>")
		}
		if _, err := os.Stat(options.FilePath); err != nil {
			return "", "", fmt.Errorf("file not found: %s", options.FilePath)
		}
		return options.FilePath, "[File upload]: " + options.FilePath, nil
	}

	if options.FilePath != "" {
		content, err := os.ReadFile(options.FilePath)
		if err != nil {
			return "", "", fmt.Errorf("file not found: %s", options.FilePath)
		}
		if len(content) == 0 {
			return "", "", fmt.Errorf("content is empty")
		}
		return string(content), "[File]: " + options.FilePath, nil
	}

	if len(options.Args) > 0 {
		content := strings.Join(options.Args, " ")
		if content == "" {
			return "", "", fmt.Errorf("content is empty")
		}
		return content, "[Text&Link]: " + content, nil
	}

	if !options.StdinTTY {
		content, err := io.ReadAll(service.stdin)
		if err != nil {
			return "", "", fmt.Errorf("read stdin: %w", err)
		}
		if len(content) == 0 {
			return "", "", fmt.Errorf("content is empty")
		}
		return string(content), "[Pipe]", nil
	}

	if !options.ReadClipboard {
		return "", "", fmt.Errorf("clipboard read is disabled; use -r/--read-clipboard, provide text, -f, or pipe stdin instead")
	}

	content, err := service.clipboard.ReadText()
	if err != nil {
		return "", "", fmt.Errorf("%w; provide text, -f, or pipe stdin instead", err)
	}
	if content == "" {
		return "", "", fmt.Errorf("content is empty")
	}
	return content, "[Clipboard]", nil
}

func resolveRequestType(value string) string {
	return value
}

func validateURLContent(requestType string, content string) error {
	if requestType != "url" {
		return nil
	}

	trimmedContent := strings.TrimSpace(content)
	if !hasValidURIScheme(trimmedContent) {
		return fmt.Errorf("invalid URL: missing or invalid URI scheme")
	}

	parsedURL, err := url.Parse(trimmedContent)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme == "" {
		return fmt.Errorf("invalid URL: missing or invalid URI scheme")
	}
	return nil
}

func hasValidURIScheme(content string) bool {
	schemeSeparatorIndex := strings.Index(content, ":")
	if schemeSeparatorIndex <= 0 {
		return false
	}
	return uriSchemePattern.MatchString(content[:schemeSeparatorIndex])
}

func writeConfirmPreview(writer io.Writer, label string, content string, options NewOptions, requestType string) {
	writeConfirmContentField(writer, "content", resolveConfirmContent(label, content))
	if options.Slug != "" {
		writeConfirmField(writer, "slug", options.Slug)
	}
	if options.Topic != "" {
		writeConfirmField(writer, "topic", options.Topic)
	}
	if options.Title != "" {
		writeConfirmField(writer, "title", options.Title)
	}
	if options.TTL != nil {
		writeConfirmField(writer, "ttl", fmt.Sprintf("%d min", *options.TTL))
	}
	if typeLabel := formatConfirmType(options.Type, requestType); typeLabel != "" {
		writeConfirmField(writer, "type", typeLabel)
	}
	if options.Export {
		writeConfirmField(writer, "export", "full response")
	}
	if options.Method == http.MethodPut {
		writeConfirmField(writer, "mode", "overwrite")
	}
	fmt.Fprintln(writer)
}

func writeConfirmField(writer io.Writer, key string, value string) {
	fmt.Fprintf(writer, "%-12s %s\n", key, value)
}

func writeConfirmContentField(writer io.Writer, key string, value string) {
	lines := formatConfirmContentLines(value)
	if len(lines) == 0 {
		writeConfirmField(writer, key, "")
		return
	}

	fmt.Fprintf(writer, "%-12s %s\n", key, lines[0])
	indent := strings.Repeat(" ", 13)
	for _, line := range lines[1:] {
		fmt.Fprintf(writer, "%s%s\n", indent, line)
	}
}

func resolveConfirmContent(label string, content string) string {
	switch {
	case strings.HasPrefix(label, "[File upload]: "):
		return strings.TrimPrefix(label, "[File upload]: ")
	case strings.HasPrefix(label, "[File]: "):
		return strings.TrimPrefix(label, "[File]: ")
	case label == "[Clipboard]", label == "[Pipe]", strings.HasPrefix(label, "[Text&Link]: "):
		return content
	default:
		return label
	}
}

func formatConfirmContentLines(value string) []string {
	const maxConfirmLines = 3
	const maxConfirmLineLength = 27

	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	rawLines := strings.Split(normalized, "\n")
	if len(rawLines) == 0 {
		return []string{""}
	}

	lines := make([]string, 0, maxConfirmLines+1)
	for index, rawLine := range rawLines {
		if index == maxConfirmLines {
			lines = append(lines, "...")
			break
		}
		lines = append(lines, truncateConfirmLine(rawLine, maxConfirmLineLength))
	}

	return lines
}

func truncateConfirmLine(value string, maxLength int) string {
	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}

	return string(runes[:maxLength]) + "..."
}

func formatConfirmType(value string, requestType string) string {
	switch value {
	case "":
		return requestType
	case "md2html":
		return "markdown -> html"
	case "qrcode":
		return "text -> qrcode"
	case "file":
		return "file"
	}
	if requestType != "" {
		return requestType
	}
	return value
}

func validateNewOptions(options NewOptions) error {
	if isTopicCreation(options) {
		return validateTopicCreationOptions(options)
	}

	if options.TTL != nil && *options.TTL < 0 {
		return fmt.Errorf("ttl must be a non-negative number")
	}
	if options.Topic == "" {
		return nil
	}
	if options.Title == "" {
		return fmt.Errorf("--title is required when --topic is set")
	}
	if options.Slug == "" {
		return nil
	}
	topicPrefix := options.Topic + "/"
	if strings.Contains(options.Slug, "/") && !strings.HasPrefix(options.Slug, topicPrefix) {
		return fmt.Errorf("slug must start with '%s' when --topic %s is set", topicPrefix, options.Topic)
	}
	return nil
}

func isTopicCreation(options NewOptions) bool {
	return options.Type == "topic"
}

func validateTopicCreationOptions(options NewOptions) error {
	if options.Slug == "" {
		return fmt.Errorf("--slug is required when --type topic is set")
	}
	if options.TTL != nil {
		return fmt.Errorf("--ttl is not supported when --type topic is set")
	}
	if options.FilePath != "" {
		return fmt.Errorf("--file is not supported when --type topic is set")
	}
	if options.Topic != "" {
		return fmt.Errorf("--topic is not supported when --type topic is set")
	}
	if options.Title != "" {
		return fmt.Errorf("--title is not supported when --type topic is set")
	}
	if len(options.Args) > 0 {
		return fmt.Errorf("content is not supported when --type topic is set")
	}
	if options.ReadClipboard {
		return fmt.Errorf("--read-clipboard is not supported when --type topic is set")
	}
	return nil
}

func formatJSON(body []byte) (string, error) {
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, body, "", "  "); err != nil {
		return "", fmt.Errorf("format JSON: %w", err)
	}
	if err := formatted.WriteByte('\n'); err != nil {
		return "", fmt.Errorf("write trailing newline: %w", err)
	}
	return formatted.String(), nil
}
