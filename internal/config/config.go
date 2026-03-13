package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const configFileName = "config.json"

type FileConfig struct {
	Host  string `json:"host"`
	Token string `json:"token"`
}

type Config struct {
	Host       string
	Token      string
	ConfigPath string
}

func Load() (Config, error) {
	filePath, err := resolveConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	fileConfig, err := loadFileConfig(filePath)
	if err != nil {
		return Config{}, err
	}

	host := strings.TrimSpace(os.Getenv("POST_HOST"))
	if host == "" {
		host = strings.TrimSpace(fileConfig.Host)
	}

	token := strings.TrimSpace(os.Getenv("POST_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(fileConfig.Token)
	}

	return Config{
		Host:       host,
		Token:      token,
		ConfigPath: filePath,
	}, nil
}

func resolveConfigFilePath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("POST_CONFIG")); path != "" {
		return path, nil
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config directory: %w", err)
	}

	return filepath.Join(configDir, "post", configFileName), nil
}

func loadFileConfig(filePath string) (FileConfig, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return FileConfig{}, nil
		}
		return FileConfig{}, fmt.Errorf("read config file %s: %w", filePath, err)
	}

	var fileConfig FileConfig
	if err := json.Unmarshal(content, &fileConfig); err != nil {
		return FileConfig{}, fmt.Errorf("parse config file %s: %w", filePath, err)
	}

	return fileConfig, nil
}
