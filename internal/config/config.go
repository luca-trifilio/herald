package config

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/luca-trifilio/herald/internal/models"
)

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "newsdigest", "config.json"), nil
}

func Load() (*models.Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	cfg := &models.Config{Theme: "dark"}

	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	if v := os.Getenv("ANTHROPIC_API_KEY"); v != "" {
		cfg.AnthropicAPIKey = v
	}
	if v := os.Getenv("GMAIL_CLIENT_ID"); v != "" {
		cfg.GmailClientID = v
	}
	if v := os.Getenv("GMAIL_CLIENT_SECRET"); v != "" {
		cfg.GmailClientSecret = v
	}
	if v := os.Getenv("GMAIL_ACCESS_TOKEN"); v != "" {
		cfg.GmailAccessToken = v
	}
	if v := os.Getenv("GMAIL_REFRESH_TOKEN"); v != "" {
		cfg.GmailRefreshToken = v
	}

	return cfg, nil
}

func Save(cfg *models.Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
