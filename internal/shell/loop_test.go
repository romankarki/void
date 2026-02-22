package shell

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/void-shell/void/internal/config"
)

func TestRunMetaCopyErrorSuccess(t *testing.T) {
	app := &App{lastError: "cd: path not found"}

	original := copyTextToClipboard
	defer func() { copyTextToClipboard = original }()

	var copied string
	copyTextToClipboard = func(text string) error {
		copied = text
		return nil
	}

	if code := app.runMeta("void copy-error"); code != 0 {
		t.Fatalf("expected copy-error to succeed, got %d", code)
	}
	if copied != app.lastError {
		t.Fatalf("expected copied text %q, got %q", app.lastError, copied)
	}
}

func TestRunMetaCpErrAliasSuccess(t *testing.T) {
	app := &App{lastError: "reload failed: boom"}

	original := copyTextToClipboard
	defer func() { copyTextToClipboard = original }()

	var copied string
	copyTextToClipboard = func(text string) error {
		copied = text
		return nil
	}

	if code := app.runMeta("void cp err"); code != 0 {
		t.Fatalf("expected cp err alias to succeed, got %d", code)
	}
	if copied != app.lastError {
		t.Fatalf("expected copied text %q, got %q", app.lastError, copied)
	}
}

func TestRunMetaCpErrorAliasSuccess(t *testing.T) {
	app := &App{lastError: "command failed"}

	original := copyTextToClipboard
	defer func() { copyTextToClipboard = original }()

	var copied string
	copyTextToClipboard = func(text string) error {
		copied = text
		return nil
	}

	if code := app.runMeta("void cp error"); code != 0 {
		t.Fatalf("expected cp error alias to succeed, got %d", code)
	}
	if copied != app.lastError {
		t.Fatalf("expected copied text %q, got %q", app.lastError, copied)
	}
}

func TestRunMetaCopyErrorFailure(t *testing.T) {
	app := &App{lastError: "reload failed: boom"}

	original := copyTextToClipboard
	defer func() { copyTextToClipboard = original }()

	copyTextToClipboard = func(string) error {
		return errors.New("clipboard offline")
	}

	if code := app.runMeta("void copy-error"); code != 1 {
		t.Fatalf("expected copy-error to fail, got %d", code)
	}
	if !strings.Contains(app.lastError, "copy-error failed: clipboard offline") {
		t.Fatalf("expected stored copy-error message, got %q", app.lastError)
	}
}

func TestRunMetaCopyErrorWithoutCapturedMessage(t *testing.T) {
	app := &App{}
	if code := app.runMeta("void copy-error"); code != 1 {
		t.Fatalf("expected copy-error without stored error to fail, got %d", code)
	}
	if app.lastError != "no error message captured yet" {
		t.Fatalf("expected user-friendly error, got %q", app.lastError)
	}
}

func TestRunCommandRecordsExitCodeMessage(t *testing.T) {
	shellCfg := config.ShellConfig{Executable: "sh", Args: []string{"-c"}}
	command := "exit 7"
	if runtime.GOOS == "windows" {
		shellCfg = config.ShellConfig{Executable: "cmd.exe", Args: []string{"/C"}}
		command = "exit 7"
	}

	app := &App{
		cfg: config.Config{
			Shell: shellCfg,
		},
	}

	if code := app.runCommand(command); code != 7 {
		t.Fatalf("expected exit code 7, got %d", code)
	}
	if !strings.Contains(app.lastError, "exited with code 7") {
		t.Fatalf("expected stored exit code summary, got %q", app.lastError)
	}
}

func TestIsActivationCommand(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{line: `call .venv\Scripts\activate.bat`, want: true},
		{line: `.venv\Scripts\activate`, want: true},
		{line: `deactivate`, want: true},
		{line: `conda activate base`, want: true},
		{line: `echo hello`, want: false},
	}
	for _, tc := range cases {
		if got := isActivationCommand(tc.line); got != tc.want {
			t.Fatalf("isActivationCommand(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func TestParseCmdSetOutputAndExitCode(t *testing.T) {
	block := "PATH=C:\\\\Windows\r\nVIRTUAL_ENV=C:\\\\repo\\\\.venv\r\n__VOID_EXIT_CODE=9\r\n"
	env := parseCmdSetOutput(block)
	if env["PATH"] != "C:\\\\Windows" {
		t.Fatalf("expected PATH to be parsed, got %#v", env)
	}
	if env["VIRTUAL_ENV"] != "C:\\\\repo\\\\.venv" {
		t.Fatalf("expected VIRTUAL_ENV to be parsed, got %#v", env)
	}
	if code := parseExitCode(env); code != 9 {
		t.Fatalf("expected exit code 9, got %d", code)
	}
	deleteEnvKeyCaseInsensitive(env, "__void_exit_code")
	if _, ok := env["__VOID_EXIT_CODE"]; ok {
		t.Fatalf("expected sync-only exit variable to be deleted")
	}
}

func TestDiffEnvironmentCaseInsensitiveOnWindows(t *testing.T) {
	current := []string{
		"Path=C:\\Windows",
		"VIRTUAL_ENV=C:\\repo\\.venv",
		"KEEP=1",
	}
	snapshot := map[string]string{
		"PATH":              "C:\\repo\\.venv\\Scripts;C:\\Windows",
		"CONDA_DEFAULT_ENV": "base",
		"KEEP":              "1",
	}
	setVars, unsetVars := diffEnvironment(current, snapshot)

	if got := setVars["PATH"]; got != "C:\\repo\\.venv\\Scripts;C:\\Windows" {
		t.Fatalf("expected PATH update, got %#v", setVars)
	}
	if got := setVars["CONDA_DEFAULT_ENV"]; got != "base" {
		t.Fatalf("expected CONDA_DEFAULT_ENV to be added, got %#v", setVars)
	}

	hasVirtualEnvUnset := false
	for _, key := range unsetVars {
		if strings.EqualFold(key, "VIRTUAL_ENV") {
			hasVirtualEnvUnset = true
			break
		}
	}
	if !hasVirtualEnvUnset {
		t.Fatalf("expected VIRTUAL_ENV to be unset, got %#v", unsetVars)
	}
}
