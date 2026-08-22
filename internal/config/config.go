// Package config stores which Antenne instance the CLI talks to and the session
// it holds for it.
package config

import (
	"encoding/json"
	"errors"
	"io"
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

	f, err := os.Open(Path())
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	data, err := readSecure(f)
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

// readSecure reads an open credential file, first tightening it to 0600 when it
// is found with any group or other bit set. CLI-STANDARD §6.2 requires this:
// installs predate the permission rule, and a tool that only writes correctly
// leaves every one of those tokens exposed. The stat and the chmod go through
// the open handle rather than a separate os.Stat/os.Chmod pair, which would
// leave a window for the file to be swapped between the two calls.
func readSecure(f *os.File) ([]byte, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm()&0o077 != 0 {
		if err := f.Chmod(0o600); err != nil {
			return nil, err
		}
	}
	return io.ReadAll(f)
}

// Save writes the configuration with owner-only permissions. The session token
// is a bearer credential, so the mode is set at creation rather than chmod'd
// afterwards: writing first and fixing the mode second leaves a window in which
// the token is world-readable, and on a shared machine that window is the whole
// attack. The directory is created 0700 for the same reason.
//
// The mode is re-asserted on the open handle because the perm argument to
// OpenFile applies only when the file is created — an existing file keeps
// whatever mode it already had, so rewriting a config that predates this rule
// would otherwise leave it readable. This is the same belt-and-braces
// `facile`'s own credential writer uses.
func Save(cfg Config) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	f, err := os.OpenFile(Path(), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Clear removes the stored session but keeps the instance URL, so logging out
// does not also make the user retype where their Antenne is.
//
// A file that cannot be parsed still loses its token. Logout is what somebody
// reaches for on a borrowed machine, and refusing because the JSON is malformed
// would leave a working credential exactly where they tried to remove it. The
// URL is unrecoverable in that case, so it falls back to the default.
func Clear() error {
	cfg, err := Load()
	if err != nil {
		cfg = Config{URL: DefaultURL}
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
