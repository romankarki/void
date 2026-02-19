package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/void-shell/void/internal/config"
	"github.com/void-shell/void/internal/integration"
	"github.com/void-shell/void/internal/prompt"
	"github.com/void-shell/void/internal/shell"
	"github.com/void-shell/void/internal/theme"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "prompt":
			os.Exit(runPrompt(os.Args[2:]))
		case "init":
			os.Exit(runInit(os.Args[2:]))
		}
	}

	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	cfg, configFile, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "void: failed to load config: %v\n", err)
		os.Exit(1)
	}

	app, err := shell.New(cfg, configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "void: failed to initialize shell: %v\n", err)
		os.Exit(1)
	}

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "void: %v\n", err)
		os.Exit(1)
	}
}

func runPrompt(args []string) int {
	fs := flag.NewFlagSet("prompt", flag.ContinueOnError)
	configPath := fs.String("config", "", "Path to config file")
	lastExitCode := fs.Int("last-exit-code", 0, "Previous command exit code")
	workdir := fs.String("workdir", "", "Working directory")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	cfg, _, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "void: failed to load config: %v\n", err)
		return 1
	}
	merged, err := theme.ApplyPreset(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "void: failed to apply theme preset: %v\n", err)
		return 1
	}

	out := prompt.Render(merged.Prompt.Segments, merged.Prompt.Symbol, merged.Palette, prompt.Context{
		LastExitCode: *lastExitCode,
		WorkDir:      *workdir,
	})
	fmt.Print(out)
	return 0
}

func runInit(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: void init <powershell|bash|zsh|cmd>")
		return 1
	}
	snippet, err := integration.InitScript(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "void: %v\n", err)
		return 1
	}
	fmt.Println(snippet)
	return 0
}
