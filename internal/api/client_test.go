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

func TestCreateJSONPostsToCreateAction(t *testing.T) {
	server := newJSONActionServer(t, "/create", true, func(body string) {
		if !strings.Contains(body, `"path":"demo"`) {
			t.Fatalf("unexpected body: %s", body)
		}
		if !strings.Contains(body, `"title":"Demo"`) {
			t.Fatalf("unexpected body: %s", body)
		}
		if !strings.Contains(body, `"topic":"notes"`) {
			t.Fatalf("unexpected body: %s", body)
		}
		if !strings.Contains(body, `"created":"2026-03-01T08:00:00+08:00"`) {
			t.Fatalf("unexpected body: %s", body)
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())
	_, err := client.CreateJSON(context.Background(), JSONRequest{
		Path:    "demo",
		URL:     "hello",
		Title:   "Demo",
		Topic:   "notes",
		Created: "2026-03-01T08:00:00+08:00",
	}, true)
	if err != nil {
		t.Fatalf("CreateJSON returned error: %v", err)
	}
}

func TestUpdateJSONPostsToUpdateAction(t *testing.T) {
	server := newJSONActionServer(t, "/update", false, func(body string) {
		if !strings.Contains(body, `"path":"demo"`) || !strings.Contains(body, `"url":"updated"`) {
			t.Fatalf("unexpected body: %s", body)
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())
	_, err := client.UpdateJSON(context.Background(), JSONRequest{Path: "demo", URL: "updated"}, false)
	if err != nil {
		t.Fatalf("UpdateJSON returned error: %v", err)
	}
}

func TestQueryPostsToQueryAction(t *testing.T) {
	server := newJSONActionServer(t, "/query", true, func(body string) {
		if !strings.Contains(body, `"path":"demo"`) || !strings.Contains(body, `"type":"topic"`) {
			t.Fatalf("unexpected body: %s", body)
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())
	_, err := client.Query(context.Background(), JSONRequest{Path: "demo", Type: "topic"}, true)
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
}

func TestDeletePostsToDeleteAction(t *testing.T) {
	server := newJSONActionServer(t, "/delete", false, func(body string) {
		if !strings.Contains(body, `"path":"demo"`) {
			t.Fatalf("unexpected body: %s", body)
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())
	_, err := client.Delete(context.Background(), JSONRequest{Path: "demo"}, false)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestCreateFilePostsMultipartToCreateAction(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sample.txt")
	if err := os.WriteFile(filePath, []byte("sample"), 0o644); err != nil {
		t.Fatalf("write sample file: %v", err)
	}

	server := newUploadActionServer(t, "/create", func(values map[string]string) {
		if values["path"] != "demo" || values["title"] != "Demo" || values["topic"] != "notes" {
			t.Fatalf("unexpected fields: %#v", values)
		}
		if values["created"] != "2026-03-01" {
			t.Fatalf("unexpected created field: %#v", values)
		}
		if values["ttl"] != "15" {
			t.Fatalf("unexpected ttl field: %#v", values)
		}
	})
	defer server.Close()

	ttl := 15
	client := NewClient(server.URL, "token", server.Client())
	_, err := client.CreateFile(context.Background(), filePath, "demo", "Demo", "notes", "2026-03-01", &ttl, false)
	if err != nil {
		t.Fatalf("CreateFile returned error: %v", err)
	}
}

func TestUpdateFilePostsMultipartToUpdateAction(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sample.txt")
	if err := os.WriteFile(filePath, []byte("sample"), 0o644); err != nil {
		t.Fatalf("write sample file: %v", err)
	}

	server := newUploadActionServer(t, "/update", func(values map[string]string) {
		if values["path"] != "demo" {
			t.Fatalf("unexpected fields: %#v", values)
		}
	})
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())
	_, err := client.UpdateFile(context.Background(), filePath, "demo", "", "", "", nil, false)
	if err != nil {
		t.Fatalf("UpdateFile returned error: %v", err)
	}
}

func TestQueryFormatsAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte(`{"error":"bad input","hint":"retry later"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())
	_, err := client.Query(context.Background(), JSONRequest{}, false)
	if err == nil || err.Error() != "API error: bad input - retry later" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newJSONActionServer(t *testing.T, expectedPath string, expectedExport bool, assertBody func(string)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", request.Method)
		}
		if request.URL.Path != expectedPath {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		expectedExportHeader := ""
		if expectedExport {
			expectedExportHeader = "true"
		}
		if got := request.Header.Get("X-Export"); got != expectedExportHeader {
			t.Fatalf("unexpected export header: %s", got)
		}
		if got := request.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("unexpected content type: %s", got)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		assertBody(string(body))
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"surl":"https://sho.rt/demo"}`))
	}))
}

func newUploadActionServer(t *testing.T, expectedPath string, assertValues func(map[string]string)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", request.Method)
		}
		if request.URL.Path != expectedPath {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("unexpected auth header: %s", got)
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
		assertValues(values)

		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"surl":"https://sho.rt/file"}`))
	}))
}
