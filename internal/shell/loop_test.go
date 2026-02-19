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
