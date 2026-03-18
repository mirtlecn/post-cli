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
	if cfg.PubTopic != "" {
		t.Fatalf("unexpected pub topic: %s", cfg.PubTopic)
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

func TestLoadUsesPubTopicFromEnvOrConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"host":"https://example.com","token":"demo","pub_topic":"from-config"}`), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("POST_CONFIG", configPath)
	t.Setenv("POST_HOST", "")
	t.Setenv("POST_TOKEN", "")
	t.Setenv("POST_PUB_TOPIC", "from-env")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.PubTopic != "from-env" {
		t.Fatalf("unexpected pub topic: %s", cfg.PubTopic)
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
	t.Setenv("USERPROFILE", tempDir)
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
