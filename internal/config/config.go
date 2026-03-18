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
	Host     string `json:"host"`
	Token    string `json:"token"`
	PubTopic string `json:"pub_topic"`
}

type Config struct {
	Host       string
	Token      string
	PubTopic   string
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

	pubTopic := strings.TrimSpace(os.Getenv("POST_PUB_TOPIC"))
	if pubTopic == "" {
		pubTopic = strings.TrimSpace(fileConfig.PubTopic)
	}

	return Config{
		Host:       host,
		Token:      token,
		PubTopic:   pubTopic,
		ConfigPath: filePath,
	}, nil
}

func resolveConfigFilePath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("POST_CONFIG")); path != "" {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}

	return filepath.Join(homeDir, ".config", "post", configFileName), nil
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
