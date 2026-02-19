package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSimpleConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := `preset = "minimal"
[shell]
executable = "sh"
args = ["-c"]
[prompt]
symbol = ">"
segments = ["path", "time"]
[history]
path = "/history"
max_size = 42
[palette]
path_fg = "#ffffff"
path_bg = "#112233"
[alias]
ls = "ls -la"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, src, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if src != path {
		t.Fatalf("expected source path %s, got %s", path, src)
	}
	if cfg.History.MaxSize != 42 {
		t.Fatalf("expected history max 42, got %d", cfg.History.MaxSize)
	}
	if cfg.Alias["ls"] != "ls -la" {
		t.Fatalf("alias not parsed")
	}
	if cfg.Palette["path_bg"] != "#112233" {
		t.Fatalf("palette not parsed")
	}
}
