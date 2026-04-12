package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server string `yaml:"server"`
	Token  string `yaml:"token,omitempty"`
	UserID string `yaml:"user-id,omitempty"`
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "klokku", "config.yaml")
}

// Load reads the config file. Returns zero Config if file doesn't exist.
func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config: %w", err)
	}
	return cfg, nil
}

// Save writes the config to the given path, creating parent directories.
func Save(path string, cfg Config) error {
	if path == "" {
		path = DefaultPath()
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// Resolve merges config from file, env vars, and flags.
// Priority: flags > env > file.
func Resolve(fileCfg Config, flagServer, flagUserID, flagToken string) Config {
	resolved := fileCfg

	// Env overrides file
	if v := os.Getenv("KLOKKU_SERVER"); v != "" {
		resolved.Server = v
	}
	if v := os.Getenv("KLOKKU_USER_ID"); v != "" {
		resolved.UserID = v
	}
	if v := os.Getenv("KLOKKU_TOKEN"); v != "" {
		resolved.Token = v
	}

	// Flags override env
	if flagServer != "" {
		resolved.Server = flagServer
	}
	if flagUserID != "" {
		resolved.UserID = flagUserID
	}
	if flagToken != "" {
		resolved.Token = flagToken
	}

	return resolved
}

// Validate checks that the config has required fields for API calls.
func Validate(cfg Config) error {
	if cfg.Server == "" {
		return fmt.Errorf("server URL is required (set via --server, KLOKKU_SERVER, or config file)")
	}
	if cfg.Token != "" && cfg.UserID != "" {
		return fmt.Errorf("cannot set both token and user-id; use token for managed (app.klokku.com) or user-id for self-hosted")
	}
	if cfg.Token == "" && cfg.UserID == "" {
		return fmt.Errorf("authentication required: set --token (managed) or --user-id (self-hosted)")
	}
	return nil
}
