package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesConfigFileWhenEnvMissing(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"host":"https://example.com","token":"demo"}`), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("POST_CONFIG", configPath)
	t.Setenv("POST_HOST", "")
	t.Setenv("POST_TOKEN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Host != "https://example.com" {
		t.Fatalf("unexpected host: %s", cfg.Host)
	}
	if cfg.Token != "demo" {
		t.Fatalf("unexpected token: %s", cfg.Token)
	}
}

func TestLoadEnvOverridesConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"host":"https://example.com","token":"demo"}`), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("POST_CONFIG", configPath)
	t.Setenv("POST_HOST", "https://override.example.com")
	t.Setenv("POST_TOKEN", "override-token")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Host != "https://override.example.com" {
		t.Fatalf("unexpected host: %s", cfg.Host)
	}
	if cfg.Token != "override-token" {
		t.Fatalf("unexpected token: %s", cfg.Token)
	}
}

func TestLoadReturnsParseErrorForInvalidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{invalid json}`), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("POST_CONFIG", configPath)

	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadUsesDefaultDotConfigPath(t *testing.T) {
	tempDir := t.TempDir()
	defaultPath := filepath.Join(tempDir, ".config", "post")
	if err := os.MkdirAll(defaultPath, 0o755); err != nil {
		t.Fatalf("create config dir: %v", err)
	}

	configPath := filepath.Join(defaultPath, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"host":"https://example.com","token":"demo"}`), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("HOME", tempDir)
	t.Setenv("POST_CONFIG", "")
	t.Setenv("POST_HOST", "")
	t.Setenv("POST_TOKEN", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.ConfigPath != configPath {
		t.Fatalf("unexpected config path: %s", cfg.ConfigPath)
	}
	if cfg.Host != "https://example.com" || cfg.Token != "demo" {
		t.Fatalf("unexpected config values: %#v", cfg)
	}
}
