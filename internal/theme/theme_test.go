package theme

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/void-shell/void/internal/config"
)

func TestApplyPresetFallsBackToExecutableDirectory(t *testing.T) {
	tmp := t.TempDir()
	exeDir := filepath.Join(tmp, "bin")
	presetDir := filepath.Join(exeDir, "presets")
	if err := os.MkdirAll(presetDir, 0o755); err != nil {
		t.Fatalf("mkdir preset dir: %v", err)
	}

	presetContent := `[prompt]
symbol = ">>"
segments = ["path"]

[palette]
path_fg = "#abcdef"
`
	if err := os.WriteFile(filepath.Join(presetDir, "cyberpunk.toml"), []byte(presetContent), 0o644); err != nil {
		t.Fatalf("write preset file: %v", err)
	}

	workingDir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("mkdir work dir: %v", err)
	}

	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	origExecutablePath := executablePath
	executablePath = func() (string, error) {
		return filepath.Join(exeDir, "void.exe"), nil
	}
	t.Cleanup(func() {
		executablePath = origExecutablePath
	})

	cfg := config.Default()
	cfg.Preset = "cyberpunk"
	cfg.Palette = map[string]string{}

	got, err := ApplyPreset(cfg)
	if err != nil {
		t.Fatalf("ApplyPreset returned error: %v", err)
	}
	if got.Prompt.Symbol != ">>" {
		t.Fatalf("expected preset prompt symbol to be applied, got %q", got.Prompt.Symbol)
	}
	if got.Palette["path_fg"] != "#abcdef" {
		t.Fatalf("expected preset palette to be applied, got %q", got.Palette["path_fg"])
	}
}
