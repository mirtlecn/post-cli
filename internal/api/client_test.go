package api

import (
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPostJSONAddsHeadersAndBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", request.Method)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		if got := request.Header.Get("X-Export"); got != "true" {
			t.Fatalf("unexpected export header: %s", got)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(string(body), `"path":"demo"`) {
			t.Fatalf("unexpected body: %s", string(body))
		}
		if !strings.Contains(string(body), `"title":"Demo"`) {
			t.Fatalf("unexpected body: %s", string(body))
		}
		if !strings.Contains(string(body), `"topic":"notes"`) {
			t.Fatalf("unexpected body: %s", string(body))
		}
		if !strings.Contains(string(body), `"created":"2026-03-01T08:00:00+08:00"`) {
			t.Fatalf("unexpected body: %s", string(body))
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"surl":"https://sho.rt/demo"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())
	_, err := client.PostJSON(context.Background(), http.MethodPost, JSONRequest{
		Path:    "demo",
		URL:     "hello",
		Title:   "Demo",
		Topic:   "notes",
		Created: "2026-03-01T08:00:00+08:00",
	}, true)
	if err != nil {
		t.Fatalf("PostJSON returned error: %v", err)
	}
}

func TestUploadFileSendsCreatedField(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sample.txt")
	if err := os.WriteFile(filePath, []byte("sample"), 0o644); err != nil {
		t.Fatalf("write sample file: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", request.Method)
		}

		mediaType, params, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("parse media type: %v", err)
		}
		if mediaType != "multipart/form-data" {
			t.Fatalf("unexpected media type: %s", mediaType)
		}

		reader := multipart.NewReader(request.Body, params["boundary"])
		values := map[string]string{}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("read part: %v", err)
			}
			body, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("read part body: %v", err)
			}
			values[part.FormName()] = string(body)
		}

		if values["path"] != "demo" || values["title"] != "Demo" || values["topic"] != "notes" {
			t.Fatalf("unexpected fields: %#v", values)
		}
		if values["created"] != "2026-03-01" {
			t.Fatalf("unexpected created field: %#v", values)
		}
		if values["ttl"] != "15" {
			t.Fatalf("unexpected ttl field: %#v", values)
		}

		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"surl":"https://sho.rt/file"}`))
	}))
	defer server.Close()

	ttl := 15
	client := NewClient(server.URL, "token", server.Client())
	_, err := client.UploadFile(context.Background(), http.MethodPost, filePath, "demo", "Demo", "notes", "2026-03-01", &ttl, false)
	if err != nil {
		t.Fatalf("UploadFile returned error: %v", err)
	}
}

func TestGetSendsJSONPayloadWhenProvided(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", request.Method)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(string(body), `"path":"demo"`) || !strings.Contains(string(body), `"type":"topic"`) {
			t.Fatalf("unexpected body: %s", string(body))
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"path":"demo"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())
	_, err := client.Get(context.Background(), JSONRequest{Path: "demo", Type: "topic"}, false)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
}

func TestGetFormatsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"error":"bad input","hint":"retry later"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())
	_, err := client.Get(context.Background(), JSONRequest{}, false)
	if err == nil || err.Error() != "API error: bad input - retry later" {
		t.Fatalf("unexpected error: %v", err)
	}
}
