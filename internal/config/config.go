// Package config stores which Antenne instance the CLI talks to and the session
// it holds for it.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// DefaultURL is where a locally running Antenne listens. There is deliberately no
// facile.studio default: this is a self-hosted tool, and a CLI that silently
// points at somebody else's instance is a surprise nobody wants.
const DefaultURL = "http://localhost:9090"

// Config is the whole stored state. It is small on purpose — everything else
// lives in the instance.
type Config struct {
	URL   string `json:"url"`
	Token string `json:"token,omitempty"`
}

// Dir is the configuration directory, honouring XDG_CONFIG_HOME.
func Dir() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "antenne")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".antenne"
	}
	return filepath.Join(home, ".config", "antenne")
}

// Path is the configuration file.
func Path() string { return filepath.Join(Dir(), "config.json") }

// Load reads the configuration, returning defaults when none exists yet.
func Load() (Config, error) {
	cfg := Config{URL: DefaultURL}

	data, err := os.ReadFile(Path())
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	if cfg.URL == "" {
		cfg.URL = DefaultURL
	}
	return cfg, nil
}

// Save writes the configuration with owner-only permissions. The session token
// is a bearer credential, so the file must not be group- or world-readable, and
// the directory is created 0700 for the same reason.
func Save(cfg Config) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), append(data, '\n'), 0o600)
}

// Clear removes the stored session but keeps the instance URL, so logging out
// does not also make the user retype where their Antenne is.
func Clear() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.Token = ""
	return Save(cfg)
}

// NormalizeURL trims a trailing slash and supplies a scheme, so `antenne login
// antenne.facile.studio` works as typed.
func NormalizeURL(raw string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		return "https://" + trimmed
	}
	return trimmed
}
