package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestRunCopyUsesCapturedPowerShellError(t *testing.T) {
	t.Setenv("VOID_LAST_ERROR", "The term 'void' is not recognized")
	t.Setenv("VOID_LAST_EXIT_CODE", "1")

	orig := copyTextToClipboard
	t.Cleanup(func() { copyTextToClipboard = orig })

	var copied string
	copyTextToClipboard = func(text string) error {
		copied = text
		return nil
	}

	if code := runCopy([]string{"error"}); code != 0 {
		t.Fatalf("expected runCopy to succeed, got %d", code)
	}
	if copied != "The term 'void' is not recognized" {
		t.Fatalf("unexpected copied value: %q", copied)
	}
}

func TestRunCopyFallsBackToExitCode(t *testing.T) {
	t.Setenv("VOID_LAST_ERROR", "")
	t.Setenv("VOID_LAST_EXIT_CODE", "127")

	orig := copyTextToClipboard
	t.Cleanup(func() { copyTextToClipboard = orig })

	var copied string
	copyTextToClipboard = func(text string) error {
		copied = text
		return nil
	}

	if code := runCopy([]string{"err"}); code != 0 {
		t.Fatalf("expected runCopy to succeed, got %d", code)
	}
	if copied != "last command exited with code 127" {
		t.Fatalf("unexpected copied value: %q", copied)
	}
}

func TestRunCopyReturnsErrorWhenClipboardFails(t *testing.T) {
	t.Setenv("VOID_LAST_ERROR", "boom")
	t.Setenv("VOID_LAST_EXIT_CODE", "1")

	orig := copyTextToClipboard
	t.Cleanup(func() { copyTextToClipboard = orig })

	copyTextToClipboard = func(string) error {
		return fmt.Errorf("clipboard unavailable")
	}

	if code := runCopy([]string{"error"}); code != 1 {
		t.Fatalf("expected runCopy failure when clipboard fails, got %d", code)
	}
}

func TestRunCopyUsageValidation(t *testing.T) {
	if code := runCopy(nil); code != 1 {
		t.Fatalf("expected usage error for empty args, got %d", code)
	}
	if code := runCopy([]string{"history"}); code != 1 {
		t.Fatalf("expected usage error for unsupported target, got %d", code)
	}
}

func TestRunCopyRequiresCapturedError(t *testing.T) {
	t.Setenv("VOID_LAST_ERROR", " ")
	t.Setenv("VOID_LAST_EXIT_CODE", "0")

	orig := copyTextToClipboard
	t.Cleanup(func() { copyTextToClipboard = orig })

	called := false
	copyTextToClipboard = func(text string) error {
		called = true
		if strings.TrimSpace(text) == "" {
			t.Fatalf("clipboard should not be called with empty text")
		}
		return nil
	}

	if code := runCopy([]string{"error"}); code != 1 {
		t.Fatalf("expected failure when no captured error exists, got %d", code)
	}
	if called {
		t.Fatalf("clipboard should not be called when no captured error exists")
	}
}
