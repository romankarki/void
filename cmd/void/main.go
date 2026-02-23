package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/void-shell/void/internal/beautify"
	"github.com/void-shell/void/internal/config"
	"github.com/void-shell/void/internal/console"
	"github.com/void-shell/void/internal/installer"
	"github.com/void-shell/void/internal/integration"
	"github.com/void-shell/void/internal/prompt"
	"github.com/void-shell/void/internal/shell"
	"github.com/void-shell/void/internal/theme"
)

var copyTextToClipboard = shell.CopyTextToClipboard

func main() {
	console.EnableUTF8()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "prompt":
			os.Exit(runPrompt(os.Args[2:]))
		case "init":
			os.Exit(runInit(os.Args[2:]))
		case "install":
			os.Exit(runInstall(os.Args[2:]))
		case "update":
			os.Exit(runUpdate(os.Args[2:]))
		case "cp":
			os.Exit(runCopy(os.Args[2:]))
		case "copy-error":
			os.Exit(runCopy([]string{"error"}))
		case "bench", "b":
			os.Exit(runBench(os.Args[2:]))
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

func runInstall(args []string) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "Apply recommended install actions without prompts")
	shellName := fs.String("shell", "", "Shell profile to configure (powershell|bash|zsh|cmd)")
	noProfile := fs.Bool("no-profile", false, "Skip shell profile changes")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	opts := installer.InstallOptions{
		Yes:       *yes,
		Shell:     *shellName,
		NoProfile: *noProfile,
	}
	if err := installer.Install(opts, os.Stdout, os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "void: install failed: %v\n", err)
		return 1
	}
	return 0
}

func runUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	repo := fs.String("repo", "", "GitHub repo for releases (owner/name), default void-shell/void")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return 1
	}

	opts := installer.UpdateOptions{
		Repo: *repo,
	}
	if err := installer.Update(opts, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "void: update failed: %v\n", err)
		return 1
	}
	return 0
}

func runCopy(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: void cp <err|error>")
		return 1
	}

	target := strings.ToLower(strings.TrimSpace(args[0]))
	switch target {
	case "err", "error":
		message := strings.TrimSpace(os.Getenv("VOID_LAST_ERROR"))
		if message == "" {
			if code := strings.TrimSpace(os.Getenv("VOID_LAST_EXIT_CODE")); code != "" && code != "0" {
				message = fmt.Sprintf("last command exited with code %s", code)
			}
		}
		if message == "" {
			fmt.Fprintln(os.Stderr, "void: no captured error found in this shell session")
			fmt.Fprintln(os.Stderr, "hint: in PowerShell, reload your profile after updating: . $PROFILE")
			return 1
		}
		if err := copyTextToClipboard(message); err != nil {
			fmt.Fprintf(os.Stderr, "void: cp error failed: %v\n", err)
			return 1
		}
		fmt.Println("copied last error to clipboard")
		return 0
	default:
		fmt.Fprintln(os.Stderr, "usage: void cp <err|error>")
		return 1
	}
}

func runBench(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: void bench <command> [args...]")
		return 1
	}
	return beautify.Run(args[0], args[1:])
}
