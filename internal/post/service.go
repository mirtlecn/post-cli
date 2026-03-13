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
	Get(ctx context.Context, path string, export bool) ([]byte, error)
	Delete(ctx context.Context, path string, export bool) ([]byte, error)
	UploadFile(ctx context.Context, method string, filePath string, slug string, ttl *int, export bool) ([]byte, error)
}

type Service struct {
	client    APIClient
	clipboard clipboard.Service
	stdin     io.Reader
	stderr    io.Writer
}

type NewOptions struct {
	Slug        string
	TTL         *int
	SkipConfirm bool
	FilePath    string
	Convert     string
	Method      string
	Export      bool
	Args        []string
	StdinTTY    bool
	Confirm     func(label string) (bool, error)
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
	content, label, err := service.resolveContent(options)
	if err != nil {
		return Result{}, err
	}

	requestType := resolveRequestType(options.Convert)
	if err := validateURLContent(requestType, content); err != nil {
		return Result{}, err
	}

	if !options.SkipConfirm && options.StdinTTY && options.Confirm != nil {
		writeConfirmPreview(service.stderr, label, options, requestType)
		accepted, confirmErr := options.Confirm(label)
		if confirmErr != nil {
			return Result{}, confirmErr
		}
		if !accepted {
			return Result{Stderr: "Aborted.\n"}, nil
		}
	}

	var responseBody []byte
	if options.Convert == "file" {
		responseBody, err = service.client.UploadFile(ctx, options.Method, options.FilePath, options.Slug, options.TTL, options.Export)
	} else {
		payload := api.JSONRequest{
			Path:    options.Slug,
			URL:     content,
			TTL:     options.TTL,
			Type:    requestType,
			Convert: options.Convert,
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
	if service.clipboard.CanWriteText() {
		if err := service.clipboard.WriteText(response.ShortURL); err != nil {
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
	body, err := service.client.Get(ctx, path, export)
	if err != nil {
		return "", err
	}
	return formatJSON(body)
}

func (service *Service) Export(ctx context.Context, path string) (string, error) {
	return service.List(ctx, path, true)
}

func (service *Service) Remove(ctx context.Context, path string, export bool) (string, error) {
	body, err := service.client.Delete(ctx, path, export)
	if err != nil {
		return "", err
	}
	return formatJSON(body)
}

func (service *Service) resolveContent(options NewOptions) (string, string, error) {
	if options.Convert == "file" {
		if options.FilePath == "" {
			return "", "", fmt.Errorf("-c file requires -f <path>")
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

	content, err := service.clipboard.ReadText()
	if err != nil {
		return "", "", fmt.Errorf("%w; provide text, -f, or pipe stdin instead", err)
	}
	if content == "" {
		return "", "", fmt.Errorf("content is empty")
	}
	return content, "[Clipboard]", nil
}

func resolveRequestType(convert string) string {
	switch convert {
	case "html", "url", "text":
		return convert
	default:
		return ""
	}
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

func writeConfirmPreview(writer io.Writer, label string, options NewOptions, requestType string) {
	fmt.Fprintln(writer, label)
	if options.Slug != "" {
		fmt.Fprintf(writer, "[Slug]: %s\n", options.Slug)
	}
	if options.TTL != nil {
		fmt.Fprintf(writer, "[Expire after]: %d min\n", *options.TTL)
	}
	if options.Convert != "" {
		fmt.Fprintf(writer, "[Convert]: %s\n", options.Convert)
	}
	if requestType != "" {
		fmt.Fprintf(writer, "[Type]: %s\n", requestType)
	}
	if options.Export {
		fmt.Fprintln(writer, "[Export]: full response")
	}
	if options.Method == http.MethodPut {
		fmt.Fprintln(writer, "[Mode]: overwrite")
	}
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
