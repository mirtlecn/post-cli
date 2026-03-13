package post

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/mirtle/post-cli/internal/api"
)

type stubClient struct {
	postJSONFunc   func(ctx context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error)
	getFunc        func(ctx context.Context, path string, export bool) ([]byte, error)
	deleteFunc     func(ctx context.Context, path string, export bool) ([]byte, error)
	uploadFileFunc func(ctx context.Context, method string, filePath string, slug string, ttl *int, export bool) ([]byte, error)
}

func (client *stubClient) PostJSON(ctx context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error) {
	return client.postJSONFunc(ctx, method, payload, export)
}

func (client *stubClient) Get(ctx context.Context, path string, export bool) ([]byte, error) {
	return client.getFunc(ctx, path, export)
}

func (client *stubClient) Delete(ctx context.Context, path string, export bool) ([]byte, error) {
	return client.deleteFunc(ctx, path, export)
}

func (client *stubClient) UploadFile(ctx context.Context, method string, filePath string, slug string, ttl *int, export bool) ([]byte, error) {
	return client.uploadFileFunc(ctx, method, filePath, slug, ttl, export)
}

type stubClipboard struct {
	readValue string
	readErr   error
	canWrite  bool
	writeErr  error
	written   string
}

func (clipboard *stubClipboard) ReadText() (string, error) {
	if clipboard.readErr != nil {
		return "", clipboard.readErr
	}
	return clipboard.readValue, nil
}

func (clipboard *stubClipboard) CanWriteText() bool {
	return clipboard.canWrite
}

func (clipboard *stubClipboard) WriteText(text string) error {
	clipboard.written = text
	return clipboard.writeErr
}

func TestNewUsesArguments(t *testing.T) {
	stderr := &bytes.Buffer{}
	clipboard := &stubClipboard{canWrite: true}
	service := NewService(&stubClient{
		postJSONFunc: func(_ context.Context, method string, payload api.JSONRequest, export bool) ([]byte, error) {
			if method != http.MethodPost {
				t.Fatalf("unexpected method: %s", method)
			}
			if payload.URL != "hello world" {
				t.Fatalf("unexpected url: %s", payload.URL)
			}
			if export {
				t.Fatal("unexpected export flag")
			}
			return []byte(`{"surl":"https://sho.rt/abc"}`), nil
		},
	}, clipboard, bytes.NewBuffer(nil), stderr)

	result, err := service.New(context.Background(), NewOptions{
		Args:        []string{"hello", "world"},
		Method:      http.MethodPost,
		SkipConfirm: true,
		StdinTTY:    true,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if result.Stdout != "https://sho.rt/abc\n" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
	if clipboard.written != "https://sho.rt/abc" {
		t.Fatalf("unexpected clipboard write: %q", clipboard.written)
	}
}

func TestNewAcceptsStandardURLContent(t *testing.T) {
	service := NewService(&stubClient{
		postJSONFunc: func(_ context.Context, _ string, payload api.JSONRequest, _ bool) ([]byte, error) {
			if payload.Type != "url" {
				t.Fatalf("unexpected type: %s", payload.Type)
			}
			if payload.URL != "https://example.com/docs" {
				t.Fatalf("unexpected url: %s", payload.URL)
			}
			return []byte(`{"surl":"https://sho.rt/url"}`), nil
		},
	}, &stubClipboard{}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	result, err := service.New(context.Background(), NewOptions{
		Args:        []string{"https://example.com/docs"},
		Convert:     "url",
		Method:      http.MethodPost,
		SkipConfirm: true,
		StdinTTY:    true,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if result.Stdout != "https://sho.rt/url\n" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}

func TestNewAcceptsCustomURIScheme(t *testing.T) {
	service := NewService(&stubClient{
		postJSONFunc: func(_ context.Context, _ string, payload api.JSONRequest, _ bool) ([]byte, error) {
			if payload.URL != "obsidian://open?vault=demo" {
				t.Fatalf("unexpected url: %s", payload.URL)
			}
			return []byte(`{"surl":"https://sho.rt/custom"}`), nil
		},
	}, &stubClipboard{}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	_, err := service.New(context.Background(), NewOptions{
		Args:        []string{"obsidian://open?vault=demo"},
		Convert:     "url",
		Method:      http.MethodPost,
		SkipConfirm: true,
		StdinTTY:    true,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
}

func TestValidateURLContentAcceptsCommonSchemes(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{name: "http", content: "http://example.com"},
		{name: "https", content: "https://example.com/docs?q=1#part"},
		{name: "mailto", content: "mailto:test@example.com"},
		{name: "ftp", content: "ftp://ftp.example.com/pub/file.txt"},
		{name: "custom plus", content: "obsidian+note://open?vault=demo"},
		{name: "custom dot", content: "web+demo.v1://open/item"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateURLContent("url", testCase.content)
			if err != nil {
				t.Fatalf("validateURLContent returned error: %v", err)
			}
		})
	}
}

func TestNewRejectsURLWithoutScheme(t *testing.T) {
	service := NewService(&stubClient{}, &stubClipboard{}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	_, err := service.New(context.Background(), NewOptions{
		Args:        []string{"example.com/path"},
		Convert:     "url",
		Method:      http.MethodPost,
		SkipConfirm: true,
		StdinTTY:    true,
	})
	if err == nil || err.Error() != "invalid URL: missing or invalid URI scheme" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRejectsURLWithInvalidScheme(t *testing.T) {
	service := NewService(&stubClient{}, &stubClipboard{}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	_, err := service.New(context.Background(), NewOptions{
		Args:        []string{"1demo://open"},
		Convert:     "url",
		Method:      http.MethodPost,
		SkipConfirm: true,
		StdinTTY:    true,
	})
	if err == nil || err.Error() != "invalid URL: missing or invalid URI scheme" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateURLContentRejectsInvalidInputs(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{name: "missing scheme", content: "example.com/path"},
		{name: "whitespace only", content: "   "},
		{name: "scheme starts with digit", content: "1demo://open"},
		{name: "scheme starts with symbol", content: "+demo://open"},
		{name: "scheme missing body", content: "://open"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			err := validateURLContent("url", testCase.content)
			if err == nil || err.Error() != "invalid URL: missing or invalid URI scheme" {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewUsesClipboardWhenNoArgs(t *testing.T) {
	service := NewService(&stubClient{
		postJSONFunc: func(_ context.Context, _ string, payload api.JSONRequest, _ bool) ([]byte, error) {
			if payload.URL != "clipboard text" {
				t.Fatalf("unexpected clipboard payload: %s", payload.URL)
			}
			return []byte(`{"surl":"https://sho.rt/from-clipboard"}`), nil
		},
	}, &stubClipboard{readValue: "clipboard text"}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	result, err := service.New(context.Background(), NewOptions{
		Method:      http.MethodPost,
		SkipConfirm: true,
		StdinTTY:    true,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if result.Stdout != "https://sho.rt/from-clipboard\n" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}

func TestNewUsesStdinWhenPiped(t *testing.T) {
	service := NewService(&stubClient{
		postJSONFunc: func(_ context.Context, _ string, payload api.JSONRequest, _ bool) ([]byte, error) {
			if payload.URL != "piped content" {
				t.Fatalf("unexpected piped payload: %s", payload.URL)
			}
			return []byte(`{"surl":"https://sho.rt/piped"}`), nil
		},
	}, &stubClipboard{}, bytes.NewBufferString("piped content"), bytes.NewBuffer(nil))

	result, err := service.New(context.Background(), NewOptions{
		Method:      http.MethodPost,
		SkipConfirm: true,
		StdinTTY:    false,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if result.Stdout != "https://sho.rt/piped\n" {
		t.Fatalf("unexpected stdout: %q", result.Stdout)
	}
}

func TestNewUploadsFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "sample.txt")
	if err := os.WriteFile(filePath, []byte("sample"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	service := NewService(&stubClient{
		uploadFileFunc: func(_ context.Context, method string, uploadPath string, slug string, ttl *int, export bool) ([]byte, error) {
			if method != http.MethodPut {
				t.Fatalf("unexpected method: %s", method)
			}
			if uploadPath != filePath {
				t.Fatalf("unexpected file path: %s", uploadPath)
			}
			if slug != "demo" {
				t.Fatalf("unexpected slug: %s", slug)
			}
			if ttl == nil || *ttl != 60 {
				t.Fatalf("unexpected ttl: %v", ttl)
			}
			if !export {
				t.Fatal("expected export flag")
			}
			return []byte(`{"surl":"https://sho.rt/file","path":"demo"}`), nil
		},
	}, &stubClipboard{}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	ttl := 60
	result, err := service.New(context.Background(), NewOptions{
		FilePath:    filePath,
		Convert:     "file",
		Method:      http.MethodPut,
		Export:      true,
		Slug:        "demo",
		TTL:         &ttl,
		SkipConfirm: true,
		StdinTTY:    true,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if result.Stdout == "" || result.Stdout[0] != '{' {
		t.Fatalf("unexpected export output: %q", result.Stdout)
	}
}

func TestNewReturnsClipboardWriteWarning(t *testing.T) {
	service := NewService(&stubClient{
		postJSONFunc: func(_ context.Context, _ string, _ api.JSONRequest, _ bool) ([]byte, error) {
			return []byte(`{"surl":"https://sho.rt/abc"}`), nil
		},
	}, &stubClipboard{canWrite: true, writeErr: errors.New("clipboard unavailable")}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	result, err := service.New(context.Background(), NewOptions{
		Args:        []string{"hello"},
		Method:      http.MethodPost,
		SkipConfirm: true,
		StdinTTY:    true,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if result.Stderr == "" {
		t.Fatal("expected clipboard warning")
	}
}

func TestNewSkipsClipboardWriteWhenUnavailable(t *testing.T) {
	service := NewService(&stubClient{
		postJSONFunc: func(_ context.Context, _ string, _ api.JSONRequest, _ bool) ([]byte, error) {
			return []byte(`{"surl":"https://sho.rt/abc"}`), nil
		},
	}, &stubClipboard{}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	result, err := service.New(context.Background(), NewOptions{
		Args:        []string{"hello"},
		Method:      http.MethodPost,
		SkipConfirm: true,
		StdinTTY:    true,
	})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if result.Stderr != "" {
		t.Fatalf("expected no clipboard stderr, got: %q", result.Stderr)
	}
}

func TestNewFailsWhenClipboardReadFails(t *testing.T) {
	service := NewService(&stubClient{}, &stubClipboard{readErr: errors.New("clipboard unavailable")}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	_, err := service.New(context.Background(), NewOptions{
		Method:      http.MethodPost,
		SkipConfirm: true,
		StdinTTY:    true,
	})
	if err == nil || err.Error() != "clipboard unavailable; provide text, -f, or pipe stdin instead" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListFormatsJSON(t *testing.T) {
	service := NewService(&stubClient{
		getFunc: func(_ context.Context, path string, export bool) ([]byte, error) {
			if path != "demo" || !export {
				t.Fatalf("unexpected args: %s %v", path, export)
			}
			return []byte(`{"path":"demo","url":"hello"}`), nil
		},
	}, &stubClipboard{}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	output, err := service.List(context.Background(), "demo", true)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if output == "" || output[0] != '{' {
		t.Fatalf("unexpected output: %q", output)
	}
}

func TestRemoveUsesDelete(t *testing.T) {
	service := NewService(&stubClient{
		deleteFunc: func(_ context.Context, path string, export bool) ([]byte, error) {
			if path != "demo" || export {
				t.Fatalf("unexpected args: %s %v", path, export)
			}
			return []byte(`{"ok":true}`), nil
		},
	}, &stubClipboard{}, bytes.NewBuffer(nil), bytes.NewBuffer(nil))

	output, err := service.Remove(context.Background(), "demo", false)
	if err != nil {
		t.Fatalf("Remove returned error: %v", err)
	}
	if output == "" {
		t.Fatal("expected output")
	}
}
