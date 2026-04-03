package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds infrastructure configuration loaded from config.yaml.
type Config struct {
	TelegramBotToken string `yaml:"telegram_bot_token"`
	SQLitePath       string `yaml:"sqlite_path"`
	LLMAPIKey        string `yaml:"llm_api_key"`
	ListenAddress    string `yaml:"listen_address"`
	LogLevel         string `yaml:"log_level"`
	EncryptionKey    string `yaml:"encryption_key"`
}

// LoadConfig reads and parses a YAML config file.
// Applies defaults for optional fields (log_level=info, listen_address=:8080).
// Returns error if file doesn't exist or required fields are missing.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	// Apply defaults for optional fields.
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = ":8080"
	}

	// Validate required fields.
	if cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("config: required field telegram_bot_token is missing or empty")
	}
	if cfg.SQLitePath == "" {
		return nil, fmt.Errorf("config: required field sqlite_path is missing or empty")
	}
	if cfg.EncryptionKey == "" {
		return nil, fmt.Errorf("config: required field encryption_key is missing or empty")
	}

	return &cfg, nil
}
