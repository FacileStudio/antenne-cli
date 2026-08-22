package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isolate points the config at a throwaway directory, so no test can read or
// write the developer's real session.
func isolate(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	return filepath.Join(base, "antenne")
}

func modeOf(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return info.Mode().Perm()
}

// writeCorrupt puts an unparseable config holding a recognisable token where
// Load expects one, bypassing Save so the bytes really are malformed.
func writeCorrupt(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	corrupt := `{"url": "https://antenne.example.com", "token": "tok-still-live"` + "\n"
	if err := os.WriteFile(Path(), []byte(corrupt), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// CLI-STANDARD §6.6: logout removes the credential. A file Load cannot parse
// must still lose its token, or the one command somebody runs on a borrowed
// machine leaves the credential exactly where they tried to remove it.
func TestClearSurvivesUnparseableConfig(t *testing.T) {
	isolate(t)
	writeCorrupt(t)

	if _, err := Load(); err == nil {
		t.Fatal("load parsed the corrupt config, so this test proves nothing")
	}
	if err := Clear(); err != nil {
		t.Fatalf("clear on an unparseable config must not fail, got %v", err)
	}

	raw, err := os.ReadFile(Path())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(raw), "tok-still-live") {
		t.Fatalf("token survived clear, file is now %q", raw)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("config is still unreadable after clear: %v", err)
	}
	if got.Token != "" {
		t.Fatalf("token is %q after clear, want empty", got.Token)
	}
	if got.URL != DefaultURL {
		t.Fatalf("url is %q after clearing an unreadable file, want the default", got.URL)
	}
}

func TestClearKeepsURL(t *testing.T) {
	isolate(t)

	if err := Save(Config{URL: "https://antenne.example.com", Token: "tok"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := Clear(); err != nil {
		t.Fatalf("clear: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Token != "" {
		t.Fatalf("token is %q after clear, want empty", got.Token)
	}
	if got.URL != "https://antenne.example.com" {
		t.Fatalf("url is %q after clear, want it kept", got.URL)
	}
}

func TestSavePermissions(t *testing.T) {
	dir := isolate(t)

	if err := Save(Config{URL: DefaultURL, Token: "tok"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got := modeOf(t, Path()); got != 0o600 {
		t.Fatalf("config mode is %#o, want 0600", got)
	}
	if got := modeOf(t, dir); got != 0o700 {
		t.Fatalf("config dir mode is %#o, want 0700", got)
	}
}

// CLI-STANDARD §6.2: OpenFile applies its perm argument only when it creates
// the file, so a config that predates the rule keeps its loose mode unless Save
// re-asserts it on the open handle.
func TestSaveReassertsModeOnExistingFile(t *testing.T) {
	isolate(t)

	if err := Save(Config{URL: DefaultURL, Token: "tok"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := os.Chmod(Path(), 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	if err := Save(Config{URL: DefaultURL, Token: "tok2"}); err != nil {
		t.Fatalf("save over an existing file: %v", err)
	}

	if got := modeOf(t, Path()); got != 0o600 {
		t.Fatalf("mode is %#o after rewriting an existing config, want 0600", got)
	}
}

// CLI-STANDARD §6.2: a credential file found group- or world-readable is
// tightened on read, because installs predate the rule.
func TestLoadTightensLoosePermissions(t *testing.T) {
	isolate(t)

	if err := Save(Config{URL: DefaultURL, Token: "tok"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := os.Chmod(Path(), 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Token != "tok" {
		t.Fatalf("token is %q, want it read back intact", got.Token)
	}
	if got := modeOf(t, Path()); got != 0o600 {
		t.Fatalf("mode is %#o after load, want it tightened to 0600", got)
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	isolate(t)

	got, err := Load()
	if err != nil {
		t.Fatalf("load with no config must not error, got %v", err)
	}
	if got.URL != DefaultURL || got.Token != "" {
		t.Fatalf("got %+v, want the default URL and no token", got)
	}
}
