package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Preset  string
	Palette map[string]string
	Shell   ShellConfig
	Prompt  PromptConfig
	History HistoryConfig
	Alias   map[string]string
}

type ShellConfig struct {
	Executable string
	Args       []string
}

type PromptConfig struct {
	Symbol   string
	Segments []string
}

type HistoryConfig struct {
	Path    string
	MaxSize int
}

func Default() Config {
	return Config{
		Shell:   ShellConfig{Executable: defaultShell(), Args: []string{}},
		Prompt:  PromptConfig{Symbol: ">", Segments: []string{"user", "path", "time"}},
		History: HistoryConfig{Path: "~/.void/history", MaxSize: 5000},
		Alias:   map[string]string{},
		Palette: map[string]string{},
	}
}

func Load(fromFlag string) (Config, string, error) {
	cfg := Default()
	path := resolveConfigPath(fromFlag)
	if path == "" {
		return cfg, "", nil
	}

	if err := decodeSimpleTOML(path, &cfg); err != nil {
		return cfg, path, fmt.Errorf("decode config: %w", err)
	}
	cfg.History.Path = expandHome(cfg.History.Path)

	if err := validate(cfg); err != nil {
		return cfg, path, err
	}

	return cfg, path, nil
}

func resolveConfigPath(fromFlag string) string {
	candidates := []string{}
	if fromFlag != "" {
		candidates = append(candidates, fromFlag)
	}
	if env := os.Getenv("TERMFORGE_CONFIG"); env != "" {
		candidates = append(candidates, env)
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".void", "config.toml"))
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		candidates = append(candidates, filepath.Join(appData, "Void", "config.toml"))
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

func validate(cfg Config) error {
	if strings.TrimSpace(cfg.Shell.Executable) == "" {
		return errors.New("shell.executable cannot be empty")
	}
	if cfg.History.MaxSize <= 0 {
		return errors.New("history.max_size must be greater than zero")
	}
	if cfg.History.Path == "" {
		return errors.New("history.path cannot be empty")
	}
	return nil
}

func decodeSimpleTOML(path string, cfg *Config) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	section := ""
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.Trim(line, "[]")
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"")

		switch section {
		case "":
			if key == "preset" {
				cfg.Preset = value
			}
		case "shell":
			switch key {
			case "executable":
				cfg.Shell.Executable = value
			case "args":
				cfg.Shell.Args = parseArray(value)
			}
		case "prompt":
			switch key {
			case "symbol":
				cfg.Prompt.Symbol = value
			case "segments":
				cfg.Prompt.Segments = parseArray(value)
			}
		case "history":
			switch key {
			case "path":
				cfg.History.Path = value
			case "max_size":
				max, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("invalid history.max_size: %w", err)
				}
				cfg.History.MaxSize = max
			}
		case "alias":
			cfg.Alias[key] = strings.Trim(value, "\"")
		}
	}
	return s.Err()
}

func parseArray(value string) []string {
	value = strings.TrimSpace(strings.Trim(value, "[]"))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, strings.Trim(strings.TrimSpace(p), "\""))
	}
	return out
}
