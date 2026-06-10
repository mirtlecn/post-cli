package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

type JSONRequest struct {
	Path    string `json:"path,omitempty"`
	URL     string `json:"url,omitempty"`
	Title   string `json:"title,omitempty"`
	Topic   string `json:"topic,omitempty"`
	Created string `json:"created,omitempty"`
	TTL     *int   `json:"ttl,omitempty"`
	Type    string `json:"type,omitempty"`
}

type APIErrorPayload struct {
	Error string `json:"error"`
	Hint  string `json:"hint"`
}

func NewClient(baseURL string, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      token,
		httpClient: httpClient,
	}
}

func (client *Client) CreateJSON(ctx context.Context, payload JSONRequest, export bool) ([]byte, error) {
	return client.postJSON(ctx, "/create", payload, export)
}

func (client *Client) UpdateJSON(ctx context.Context, payload JSONRequest, export bool) ([]byte, error) {
	return client.postJSON(ctx, "/update", payload, export)
}

func (client *Client) Query(ctx context.Context, payload JSONRequest, export bool) ([]byte, error) {
	return client.postJSON(ctx, "/query", payload, export)
}

func (client *Client) Delete(ctx context.Context, payload JSONRequest, export bool) ([]byte, error) {
	return client.postJSON(ctx, "/delete", payload, export)
}

func (client *Client) postJSON(ctx context.Context, path string, payload JSONRequest, export bool) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	request, err := client.newRequest(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	if export {
		request.Header.Set("X-Export", "true")
	}

	return client.do(request)
}

func (client *Client) CreateFile(
	ctx context.Context,
	filePath string,
	slug string,
	title string,
	topic string,
	created string,
	ttl *int,
	export bool,
) ([]byte, error) {
	return client.uploadFile(ctx, "/create", filePath, slug, title, topic, created, ttl, export)
}

func (client *Client) UpdateFile(
	ctx context.Context,
	filePath string,
	slug string,
	title string,
	topic string,
	created string,
	ttl *int,
	export bool,
) ([]byte, error) {
	return client.uploadFile(ctx, "/update", filePath, slug, title, topic, created, ttl, export)
}

func (client *Client) uploadFile(
	ctx context.Context,
	path string,
	filePath string,
	slug string,
	title string,
	topic string,
	created string,
	ttl *int,
	export bool,
) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("create multipart file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("copy file content: %w", err)
	}

	if slug != "" {
		if err := writer.WriteField("path", slug); err != nil {
			return nil, fmt.Errorf("write path field: %w", err)
		}
	}
	if title != "" {
		if err := writer.WriteField("title", title); err != nil {
			return nil, fmt.Errorf("write title field: %w", err)
		}
	}
	if topic != "" {
		if err := writer.WriteField("topic", topic); err != nil {
			return nil, fmt.Errorf("write topic field: %w", err)
		}
	}
	if created != "" {
		if err := writer.WriteField("created", created); err != nil {
			return nil, fmt.Errorf("write created field: %w", err)
		}
	}
	if ttl != nil {
		if err := writer.WriteField("ttl", strconv.Itoa(*ttl)); err != nil {
			return nil, fmt.Errorf("write ttl field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	request, err := client.newRequest(ctx, http.MethodPost, path, &body)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", writer.FormDataContentType())
	if export {
		request.Header.Set("X-Export", "true")
	}

	return client.do(request)
}

func (client *Client) newRequest(
	ctx context.Context,
	method string,
	path string,
	body io.Reader,
) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	request.Header.Set("Authorization", "Bearer "+client.token)
	return request, nil
}

func (client *Client) do(request *http.Request) ([]byte, error) {
	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request failed: %s %s (%w)", request.Method, request.URL.String(), err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if response.StatusCode >= http.StatusBadRequest {
		return nil, formatAPIError(response.StatusCode, body)
	}

	return body, nil
}

func formatAPIError(statusCode int, body []byte) error {
	var payload APIErrorPayload
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error != "" {
		if payload.Hint != "" {
			return fmt.Errorf("API error: %s - %s", payload.Error, payload.Hint)
		}
		return fmt.Errorf("API error: %s", payload.Error)
	}

	if len(body) > 0 {
		return fmt.Errorf("HTTP %d: %s", statusCode, string(body))
	}

	return fmt.Errorf("HTTP %d: empty response", statusCode)
}
