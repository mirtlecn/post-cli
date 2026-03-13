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
	TTL     *int   `json:"ttl,omitempty"`
	Type    string `json:"type,omitempty"`
	Convert string `json:"convert,omitempty"`
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

func (client *Client) PostJSON(ctx context.Context, method string, payload JSONRequest, export bool) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	request, err := client.newRequest(ctx, method, "/", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	if export {
		request.Header.Set("X-Export", "true")
	}

	return client.do(request)
}

func (client *Client) Get(ctx context.Context, path string, export bool) ([]byte, error) {
	request, err := client.newRequest(ctx, http.MethodGet, "/", nil)
	if err != nil {
		return nil, err
	}

	if export {
		request.Header.Set("X-Export", "true")
	}

	if path != "" {
		payload := JSONRequest{Path: path}
		body, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return nil, fmt.Errorf("marshal request: %w", marshalErr)
		}
		request.Body = io.NopCloser(bytes.NewReader(body))
		request.ContentLength = int64(len(body))
		request.Header.Set("Content-Type", "application/json")
	}

	return client.do(request)
}

func (client *Client) Delete(ctx context.Context, path string, export bool) ([]byte, error) {
	payload := JSONRequest{Path: path}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	request, err := client.newRequest(ctx, http.MethodDelete, "/", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	if export {
		request.Header.Set("X-Export", "true")
	}

	return client.do(request)
}

func (client *Client) UploadFile(
	ctx context.Context,
	method string,
	filePath string,
	slug string,
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
	if ttl != nil {
		if err := writer.WriteField("ttl", strconv.Itoa(*ttl)); err != nil {
			return nil, fmt.Errorf("write ttl field: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	request, err := client.newRequest(ctx, method, "/", &body)
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
