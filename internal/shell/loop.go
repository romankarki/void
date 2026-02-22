package shell

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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
	lastError string
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
				a.reportError(fmt.Sprintf("cd: %v", err))
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
		a.reportError("void commands: complete, history, reload, copy-error, cp err")
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
			a.reportError("usage: void complete <prefix>")
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
			a.reportError(fmt.Sprintf("reload failed: %v", err))
			return 1
		}
		merged, err := theme.ApplyPreset(cfg)
		if err != nil {
			a.reportError(fmt.Sprintf("reload failed: %v", err))
			return 1
		}
		a.cfg = merged
		fmt.Println("configuration reloaded")
		return 0
	case "copy-error":
		return a.copyLastError("copy-error")
	case "cp":
		if len(fields) >= 3 && strings.EqualFold(fields[2], "err") {
			return a.copyLastError("cp err")
		}
		a.reportError("usage: void cp err")
		return 1
	default:
		a.reportError("unknown void command")
		return 1
	}
}

func (a *App) copyLastError(commandName string) int {
	if strings.TrimSpace(a.lastError) == "" {
		a.reportError("no error message captured yet")
		return 1
	}
	if err := copyTextToClipboard(a.lastError); err != nil {
		a.reportError(fmt.Sprintf("%s failed: %v", commandName, err))
		return 1
	}
	fmt.Println("copied last error to clipboard")
	return 0
}

func (a *App) runCommand(line string) int {
	if handled, code := a.runBuiltin(line); handled {
		return code
	}
	if handled, code := a.runCommandWithEnvSync(line); handled {
		return code
	}

	cmd := exec.Command(a.cfg.Shell.Executable, append(a.cfg.Shell.Args, line)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			a.recordError(fmt.Sprintf("command %q exited with code %d", line, exitErr.ExitCode()))
			a.printCopyErrorHint()
			return exitErr.ExitCode()
		}
		a.reportError(fmt.Sprintf("void: run command: %v", err))
		return 1
	}
	return 0
}

func (a *App) runCommandWithEnvSync(line string) (bool, int) {
	if !isCmdShellExecutable(a.cfg.Shell.Executable) || !isActivationCommand(line) {
		return false, 0
	}

	const marker = "__VOID_ENV_SYNC_BEGIN__"
	wrapped := line + ` & set "__VOID_EXIT_CODE=!ERRORLEVEL!" & echo ` + marker + ` & set`
	cmd := exec.Command(a.cfg.Shell.Executable, "/V:ON", "/C", wrapped)
	cmd.Stdin = os.Stdin

	outBytes, err := cmd.CombinedOutput()
	output := string(outBytes)
	preOutput, envBlock, found := splitEnvSyncOutput(output, marker)
	if preOutput != "" {
		fmt.Print(preOutput)
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			a.recordError(fmt.Sprintf("command %q exited with code %d", line, exitErr.ExitCode()))
			a.printCopyErrorHint()
			return true, exitErr.ExitCode()
		}
		a.reportError(fmt.Sprintf("void: run command: %v", err))
		return true, 1
	}

	if !found {
		return true, 0
	}

	envSnapshot := parseCmdSetOutput(envBlock)
	exitCode := parseExitCode(envSnapshot)
	deleteEnvKeyCaseInsensitive(envSnapshot, "__VOID_EXIT_CODE")
	applyEnvironmentSnapshot(envSnapshot)

	if exitCode != 0 {
		a.recordError(fmt.Sprintf("command %q exited with code %d", line, exitCode))
		a.printCopyErrorHint()
	}
	return true, exitCode
}

func isCmdShellExecutable(executable string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(executable)))
	return base == "cmd" || base == "cmd.exe"
}

func isActivationCommand(line string) bool {
	lower := strings.ToLower(strings.TrimSpace(line))
	if lower == "" {
		return false
	}
	if strings.Contains(lower, "activate.bat") {
		return true
	}
	if strings.Contains(lower, `\scripts\activate`) || strings.Contains(lower, `/scripts/activate`) {
		return true
	}
	if strings.HasPrefix(lower, "conda activate ") || strings.HasPrefix(lower, "conda deactivate") {
		return true
	}
	if lower == "deactivate" || strings.HasPrefix(lower, "deactivate ") {
		return true
	}
	if strings.HasPrefix(lower, "call ") {
		target := strings.TrimSpace(strings.TrimPrefix(lower, "call "))
		if strings.Contains(target, "activate") || strings.HasPrefix(target, "deactivate") {
			return true
		}
	}
	return false
}

func splitEnvSyncOutput(output, marker string) (string, string, bool) {
	idx := strings.Index(output, marker)
	if idx == -1 {
		return output, "", false
	}
	pre := output[:idx]
	post := output[idx+len(marker):]
	post = strings.TrimLeft(post, "\r\n")
	return pre, post, true
}

func parseCmdSetOutput(block string) map[string]string {
	env := map[string]string{}
	lines := strings.Split(block, "\n")
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		if key == "" || strings.HasPrefix(key, "=") {
			continue
		}
		env[key] = line[idx+1:]
	}
	return env
}

func parseExitCode(env map[string]string) int {
	for key, value := range env {
		if strings.EqualFold(key, "__VOID_EXIT_CODE") {
			code, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return 0
			}
			return code
		}
	}
	return 0
}

func deleteEnvKeyCaseInsensitive(env map[string]string, key string) {
	for existing := range env {
		if strings.EqualFold(existing, key) {
			delete(env, existing)
		}
	}
}

func applyEnvironmentSnapshot(snapshot map[string]string) {
	setVars, unsetVars := diffEnvironment(os.Environ(), snapshot)
	for key, value := range setVars {
		_ = os.Setenv(key, value)
	}
	for _, key := range unsetVars {
		_ = os.Unsetenv(key)
	}
}

func diffEnvironment(current []string, snapshot map[string]string) (map[string]string, []string) {
	currentEnv := normalizeEnvironment(current)
	nextEnv := normalizeEnvironmentFromMap(snapshot)

	setVars := map[string]string{}
	unsetVars := make([]string, 0)

	for normKey, next := range nextEnv {
		cur, exists := currentEnv[normKey]
		if !exists || cur.value != next.value || cur.key != next.key {
			setVars[next.key] = next.value
		}
	}
	for normKey, cur := range currentEnv {
		if _, exists := nextEnv[normKey]; !exists {
			unsetVars = append(unsetVars, cur.key)
		}
	}
	sort.Strings(unsetVars)

	return setVars, unsetVars
}

type envEntry struct {
	key   string
	value string
}

func normalizeEnvironment(entries []string) map[string]envEntry {
	env := map[string]envEntry{}
	for _, entry := range entries {
		idx := strings.Index(entry, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(entry[:idx])
		if key == "" || strings.HasPrefix(key, "=") {
			continue
		}
		norm := normalizeEnvKey(key)
		env[norm] = envEntry{key: key, value: entry[idx+1:]}
	}
	return env
}

func normalizeEnvironmentFromMap(entries map[string]string) map[string]envEntry {
	env := map[string]envEntry{}
	for key, value := range entries {
		key = strings.TrimSpace(key)
		if key == "" || strings.HasPrefix(key, "=") {
			continue
		}
		norm := normalizeEnvKey(key)
		env[norm] = envEntry{key: key, value: value}
	}
	return env
}

func normalizeEnvKey(key string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(key)
	}
	return key
}

func (a *App) recordError(message string) {
	a.lastError = strings.TrimSpace(message)
}

func (a *App) printCopyErrorHint() {
	if strings.TrimSpace(a.lastError) == "" {
		return
	}
	fmt.Fprintln(os.Stderr, "hint: run `void cp err` (or `void copy-error`) to copy the last error")
}

func (a *App) reportError(message string) {
	a.recordError(message)
	if strings.TrimSpace(a.lastError) == "" {
		return
	}
	fmt.Fprintln(os.Stderr, a.lastError)
	a.printCopyErrorHint()
}
