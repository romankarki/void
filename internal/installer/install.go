package installer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/void-shell/void/internal/integration"
)

const (
	profileMarkerStart = "# >>> void init >>>"
	profileMarkerEnd   = "# <<< void init <<<"
	defaultRepo        = "void-shell/void"
)

type InstallOptions struct {
	Yes       bool
	Shell     string
	NoProfile bool
}

func Install(opts InstallOptions, stdout io.Writer, stdin io.Reader) error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}

	targetPath, err := installBinaryPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create install directory: %w", err)
	}

	if !samePath(exePath, targetPath) {
		if err := copyFile(exePath, targetPath, 0o755); err != nil {
			return fmt.Errorf("install executable: %w", err)
		}
	}

	configPath, createdConfig, err := ensureDefaultConfig()
	if err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		addPath := opts.Yes || confirm(stdin, stdout, "Add Void to your user PATH? [Y/n]: ", true)
		if addPath {
			changed, err := ensureUserPathHas(filepath.Dir(targetPath))
			if err != nil {
				return err
			}
			if changed {
				fmt.Fprintf(stdout, "Updated user PATH with %s\n", filepath.Dir(targetPath))
			}
		}
	}

	if !opts.NoProfile {
		shellName := normalizeShell(strings.TrimSpace(opts.Shell))
		if shellName == "" {
			shellName = defaultProfileShell()
		}
		configureProfile := opts.Yes || confirm(stdin, stdout, fmt.Sprintf("Configure %s profile for Void prompt? [Y/n]: ", shellName), true)
		if configureProfile {
			if err := installProfileSnippet(shellName); err != nil {
				return err
			}
			fmt.Fprintf(stdout, "Updated %s profile with Void prompt integration\n", shellName)
		}
	}

	fmt.Fprintln(stdout, "Void install complete.")
	fmt.Fprintf(stdout, "Binary: %s\n", targetPath)
	fmt.Fprintf(stdout, "Config: %s", configPath)
	if createdConfig {
		fmt.Fprint(stdout, " (created)")
	}
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "To use void immediately, run:")
	fmt.Fprintf(stdout, "  . $PROFILE")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Or open a new terminal window.")
	return nil
}

func installProfileSnippet(shellName string) error {
	snippet, err := integration.InitScript(shellName)
	if err != nil {
		return err
	}

	profilePath, err := profilePathForShell(shellName)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(profilePath), 0o755); err != nil {
		return fmt.Errorf("create profile directory: %w", err)
	}

	block := profileMarkerStart + "\n" + snippet + "\n" + profileMarkerEnd + "\n"
	return appendBlockIfMissing(profilePath, block, profileMarkerStart)
}

func appendBlockIfMissing(path, block, marker string) error {
	existing, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read profile: %w", err)
		}
		existing = nil
	}
	content := string(existing)
	if start := strings.Index(content, marker); start >= 0 {
		endRel := strings.Index(content[start:], profileMarkerEnd)
		if endRel >= 0 {
			end := start + endRel + len(profileMarkerEnd)
			for end < len(content) && (content[end] == '\r' || content[end] == '\n') {
				end++
			}
			content = content[:start] + block + content[end:]

			mode := os.FileMode(0o644)
			if info, err := os.Stat(path); err == nil {
				mode = info.Mode().Perm()
			}
			if err := os.WriteFile(path, []byte(content), mode); err != nil {
				return fmt.Errorf("write profile: %w", err)
			}
			return nil
		}
	}

	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}

	combined := append([]byte(content), []byte(block)...)
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		combined = append(existing, '\n')
		combined = append(combined, []byte(block)...)
	}

	if err := os.WriteFile(path, combined, mode); err != nil {
		return fmt.Errorf("write profile: %w", err)
	}
	return nil
}

func ensureDefaultConfig() (string, bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", false, fmt.Errorf("resolve home directory: %w", err)
	}
	configDir := filepath.Join(home, ".void")
	configPath := filepath.Join(configDir, "config.toml")

	if _, err := os.Stat(configPath); err == nil {
		return configPath, false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", false, fmt.Errorf("check config path: %w", err)
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", false, fmt.Errorf("create config directory: %w", err)
	}
	if err := os.WriteFile(configPath, []byte(defaultConfigTOML), 0o644); err != nil {
		return "", false, fmt.Errorf("write default config: %w", err)
	}
	return configPath, true, nil
}

func installBinaryPath() (string, error) {
	root, err := installRootDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(root, "bin", "void.exe"), nil
	}
	return filepath.Join(root, "bin", "void"), nil
}

func installRootDir() (string, error) {
	if runtime.GOOS == "windows" {
		localAppData := strings.TrimSpace(os.Getenv("LOCALAPPDATA"))
		if localAppData == "" {
			return "", fmt.Errorf("LOCALAPPDATA is not set")
		}
		return filepath.Join(localAppData, "Void"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".void"), nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func samePath(a, b string) bool {
	absA, errA := filepath.Abs(a)
	absB, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(absA), filepath.Clean(absB))
	}
	return filepath.Clean(absA) == filepath.Clean(absB)
}

func defaultProfileShell() string {
	if runtime.GOOS == "windows" {
		return "powershell"
	}
	return "bash"
}

func normalizeShell(shell string) string {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "pwsh", "powershell":
		return "powershell"
	case "bash":
		return "bash"
	case "zsh":
		return "zsh"
	case "cmd", "cmd.exe":
		return "cmd"
	default:
		return ""
	}
}

func profilePathForShell(shellName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	switch normalizeShell(shellName) {
	case "powershell":
		return detectPowerShellProfilePath(home)
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shellName)
	}
}

func detectPowerShellProfilePath(home string) (string, error) {
	psExe := findPowerShellBinary()
	if psExe != "" {
		cmd := exec.Command(psExe, "-NoProfile", "-NonInteractive", "-Command", "$PROFILE.CurrentUserCurrentHost")
		out, err := cmd.Output()
		if err == nil {
			path := strings.TrimSpace(string(out))
			if path != "" {
				return path, nil
			}
		}
	}

	// Fallback for Windows PowerShell profile location.
	return filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"), nil
}

func findPowerShellBinary() string {
	for _, name := range []string{"pwsh", "powershell"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

func ensureUserPathHas(entry string) (bool, error) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return false, nil
	}

	current := strings.TrimSpace(os.Getenv("PATH"))
	if runtime.GOOS == "windows" {
		userPath, err := getUserPath()
		if err != nil {
			return false, err
		}
		current = strings.TrimSpace(userPath)
	}

	if pathContainsEntry(current, entry) {
		return false, nil
	}

	next := strings.Trim(current, ";")
	if next == "" {
		next = entry
	} else {
		next = next + ";" + entry
	}

	if err := setUserPath(next); err != nil {
		return false, err
	}

	processPath := os.Getenv("PATH")
	if !pathContainsEntry(processPath, entry) {
		processPath = appendPathEntry(processPath, entry)
	}
	if err := os.Setenv("PATH", processPath); err != nil {
		return false, err
	}
	return true, nil
}

func pathContainsEntry(pathValue, entry string) bool {
	needle := normalizePathEntry(entry)
	for _, part := range strings.Split(pathValue, string(os.PathListSeparator)) {
		if normalizePathEntry(part) == needle {
			return true
		}
	}
	return false
}

func normalizePathEntry(path string) string {
	p := strings.TrimSpace(path)
	p = strings.TrimRight(p, `\/`)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}

func setUserPath(value string) error {
	if runtime.GOOS != "windows" {
		return nil
	}
	psExe := findPowerShellBinary()
	if psExe == "" {
		return fmt.Errorf("could not find powershell/pwsh to update user PATH")
	}
	escaped := strings.ReplaceAll(value, `'`, `''`)
	cmd := exec.Command(psExe, "-NoProfile", "-NonInteractive", "-Command", "[Environment]::SetEnvironmentVariable('Path', '"+escaped+"', 'User')")
	if out, err := cmd.CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return fmt.Errorf("update user PATH: %w", err)
		}
		return fmt.Errorf("update user PATH: %w: %s", err, msg)
	}
	return nil
}

func getUserPath() (string, error) {
	if runtime.GOOS != "windows" {
		return os.Getenv("PATH"), nil
	}
	psExe := findPowerShellBinary()
	if psExe == "" {
		return "", fmt.Errorf("could not find powershell/pwsh to read user PATH")
	}
	cmd := exec.Command(psExe, "-NoProfile", "-NonInteractive", "-Command", "[Environment]::GetEnvironmentVariable('Path', 'User')")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return "", fmt.Errorf("read user PATH: %w", err)
		}
		return "", fmt.Errorf("read user PATH: %w: %s", err, msg)
	}
	return strings.TrimSpace(string(out)), nil
}

func appendPathEntry(current, entry string) string {
	trimmed := strings.TrimSpace(current)
	if trimmed == "" {
		return entry
	}
	return strings.TrimRight(trimmed, string(os.PathListSeparator)) + string(os.PathListSeparator) + entry
}

func confirm(stdin io.Reader, stdout io.Writer, question string, defaultYes bool) bool {
	fmt.Fprint(stdout, question)
	reader := bufio.NewReader(stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return defaultYes
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	if answer == "" {
		return defaultYes
	}
	return answer == "y" || answer == "yes"
}

const defaultConfigTOML = `# Void configuration
preset = "hacker"

[shell]
executable = "cmd.exe"
args = ["/C"]

[prompt]
symbol = ">"
segments = ["user", "path", "exit_code", "time"]

[palette]
user_fg = "#ffffff"
user_bg = "#ff6347"
path_fg = "#eceff1"
path_bg_1 = "#1565c0"
path_bg_2 = "#00695c"
path_bg_3 = "#827717"
path_bg_4 = "#0d47a1"
time_fg = "#b2dfdb"
time_bg = "#004d40"
exit_code_fg = "#ffcdd2"
exit_code_bg = "#b71c1c"
symbol_fg = "#80cbc4"

[history]
path = ".void/history"
max_size = 5000

[alias]
ll = "ls -la"
gst = "git status"
`
