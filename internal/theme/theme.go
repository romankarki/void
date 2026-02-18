package theme

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/void-shell/void/internal/config"
)

var presetMap = map[string]string{
	"minimal":   "minimal.toml",
	"cyberpunk": "cyberpunk.toml",
}

func ApplyPreset(cfg config.Config) (config.Config, error) {
	if cfg.Preset == "" {
		return cfg, nil
	}
	file, ok := presetMap[cfg.Preset]
	if !ok {
		return cfg, fmt.Errorf("unknown preset: %s", cfg.Preset)
	}
	presetPath := filepath.Join("presets", file)
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
	if _, err := os.Stat(presetPath); err != nil {
		return cfg, err
	}
	return cfg, nil
}
