package shell

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/void-shell/void/internal/autocomplete"
	"github.com/void-shell/void/internal/config"
	"github.com/void-shell/void/internal/history"
	"github.com/void-shell/void/internal/prompt"
	"github.com/void-shell/void/internal/theme"
)

type App struct {
	cfg       config.Config
	configSrc string
	lastCode  int
	history   *history.Store
	complete  *autocomplete.Engine
}

func New(cfg config.Config, configSrc string) (*App, error) {
	merged, err := theme.ApplyPreset(cfg)
	if err != nil {
		return nil, err
	}
	historyStore, err := history.New(merged.History.Path, merged.History.MaxSize)
	if err != nil {
		return nil, err
	}
	return &App{cfg: merged, configSrc: configSrc, history: historyStore, complete: autocomplete.New()}, nil
}

func (a *App) Run() error {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		wd, _ := os.Getwd()
		fmt.Print(prompt.Render(a.cfg.Prompt.Segments, a.cfg.Prompt.Symbol, a.cfg.Palette, prompt.Context{LastExitCode: a.lastCode, WorkDir: wd}))
		if !scanner.Scan() {
			_ = a.history.Save()
			return nil
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "exit" {
			return a.history.Save()
		}
		if strings.HasPrefix(line, "void ") {
			a.lastCode = a.runMeta(line)
			continue
		}
		if strings.HasPrefix(line, "cd ") {
			if err := os.Chdir(strings.TrimSpace(strings.TrimPrefix(line, "cd "))); err != nil {
				fmt.Fprintf(os.Stderr, "cd: %v\n", err)
				a.lastCode = 1
			} else {
				a.lastCode = 0
			}
			continue
		}
		expanded := a.expandAlias(line)
		a.history.Add(expanded)
		a.lastCode = a.runCommand(expanded)
	}
}

func (a *App) expandAlias(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return line
	}
	repl, ok := a.cfg.Alias[fields[0]]
	if !ok {
		return line
	}
	return strings.TrimSpace(strings.Replace(line, fields[0], repl, 1))
}

func (a *App) runMeta(line string) int {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		fmt.Println("void commands: complete, history, reload")
		return 1
	}
	switch fields[1] {
	case "history":
		for _, e := range a.history.Entries() {
			fmt.Println(e)
		}
		return 0
	case "complete":
		if len(fields) < 3 {
			fmt.Println("usage: void complete <prefix>")
			return 1
		}
		matches := a.complete.Complete(fields[2], a.history.Entries())
		for _, m := range matches {
			fmt.Println(m)
		}
		return 0
	case "reload":
		cfg, _, err := config.Load(a.configSrc)
		if err != nil {
			fmt.Fprintf(os.Stderr, "reload failed: %v\n", err)
			return 1
		}
		merged, err := theme.ApplyPreset(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "reload failed: %v\n", err)
			return 1
		}
		a.cfg = merged
		fmt.Println("configuration reloaded")
		return 0
	default:
		fmt.Println("unknown void command")
		return 1
	}
}

func (a *App) runCommand(line string) int {
	cmd := exec.Command(a.cfg.Shell.Executable, append(a.cfg.Shell.Args, line)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "void: run command: %v\n", err)
		return 1
	}
	return 0
}
