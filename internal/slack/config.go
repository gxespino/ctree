package slack

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds Slack integration settings.
type Config struct {
	BotToken  string `json:"bot_token"`
	ChannelID string `json:"channel_id"`
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ctree", "slack.json")
}

// LoadConfig reads Slack config from disk. Returns (nil, nil) if not configured.
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.BotToken == "" || cfg.ChannelID == "" {
		return nil, nil
	}
	return &cfg, nil
}

// IsConfigured returns true if Slack config exists and is valid.
func IsConfigured() bool {
	cfg, err := LoadConfig()
	return cfg != nil && err == nil
}

// SaveConfig writes Slack config to disk.
func SaveConfig(cfg Config) error {
	dir := filepath.Dir(configPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), append(data, '\n'), 0o644)
}
