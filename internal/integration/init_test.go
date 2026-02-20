package integration

import (
	"strings"
	"testing"
)

func TestInitScriptSupportedShells(t *testing.T) {
	tests := []string{"powershell", "pwsh", "bash", "zsh", "cmd", "cmd.exe"}
	for _, shell := range tests {
		shell := shell
		t.Run(shell, func(t *testing.T) {
			snippet, err := InitScript(shell)
			if err != nil {
				t.Fatalf("InitScript returned error: %v", err)
			}
			if snippet == "" {
				t.Fatal("InitScript returned empty snippet")
			}
		})
	}
}

func TestInitScriptUnsupportedShell(t *testing.T) {
	if _, err := InitScript("fish"); err == nil {
		t.Fatal("expected an error for unsupported shell")
	}
}

func TestPowershellInitScriptSetsUTF8Encoding(t *testing.T) {
	snippet, err := InitScript("powershell")
	if err != nil {
		t.Fatalf("InitScript returned error: %v", err)
	}

	checks := []string{
		"[Console]::InputEncoding = $utf8NoBom",
		"[Console]::OutputEncoding = $utf8NoBom",
		"$OutputEncoding = $utf8NoBom",
		"function __void_render_prompt",
		"$process.StandardOutput.BaseStream.CopyTo($stdout)",
		"[System.Text.Encoding]::UTF8.GetString($stdout.ToArray())",
		"$lastCommandSucceeded = $?",
		"if (-not $lastCommandSucceeded -and $code -eq 0) { $code = 1 }",
		"__void_render_prompt -code $code -workdir $PWD.Path",
	}

	for _, check := range checks {
		if !strings.Contains(snippet, check) {
			t.Fatalf("expected powershell snippet to contain %q", check)
		}
	}
}
