package theme

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/void-shell/void/internal/config"
)

//go:embed presets/*.toml
var presetFS embed.FS

var presetMap = map[string]string{
	"minimal":   "minimal.toml",
	"cyberpunk": "cyberpunk.toml",
	"hacker":    "hacker.toml",
}

var executablePath = os.Executable

func ApplyPreset(cfg config.Config) (config.Config, error) {
	if cfg.Preset == "" {
		return cfg, nil
	}
	file, ok := presetMap[cfg.Preset]
	if !ok {
		return cfg, fmt.Errorf("unknown preset: %s", cfg.Preset)
	}
	presetPath, err := resolvePresetPath(file)
	if err != nil {
		return cfg, err
	}
	themeCfg, _, err := config.Load(presetPath)
	if err != nil {
		return cfg, err
	}
	if themeCfg.Prompt.Symbol != "" {
		cfg.Prompt.Symbol = themeCfg.Prompt.Symbol
	}
	if len(themeCfg.Prompt.Segments) > 0 {
		cfg.Prompt.Segments = themeCfg.Prompt.Segments
	}
	for k, v := range themeCfg.Palette {
		cfg.Palette[k] = v
	}
	return cfg, nil
}

func resolvePresetPath(file string) (string, error) {
	candidates := []string{filepath.Join("presets", file)}
	if exe, err := executablePath(); err == nil && strings.TrimSpace(exe) != "" {
		exeCandidate := filepath.Join(filepath.Dir(exe), "presets", file)
		if exeCandidate != candidates[0] {
			candidates = append(candidates, exeCandidate)
		}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	embeddedPath := "presets/" + file
	data, err := presetFS.ReadFile(embeddedPath)
	if err == nil {
		tmpFile, err := os.CreateTemp("", "void-preset-*.toml")
		if err != nil {
			return "", fmt.Errorf("create temp file for embedded preset: %w", err)
		}
		if _, err := tmpFile.Write(data); err != nil {
			tmpFile.Close()
			return "", fmt.Errorf("write embedded preset to temp: %w", err)
		}
		tmpFile.Close()
		return tmpFile.Name(), nil
	}

	return "", fmt.Errorf("preset file %q not found (looked in: %s)", file, strings.Join(candidates, ", "))
}
