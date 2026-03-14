package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"surl":"https://sho.rt/demo"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "token", server.Client())
	_, err := client.PostJSON(context.Background(), http.MethodPost, JSONRequest{
		Path:  "demo",
		URL:   "hello",
		Title: "Demo",
		Topic: "notes",
	}, true)
	if err != nil {
		t.Fatalf("PostJSON returned error: %v", err)
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
