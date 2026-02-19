package integration

import "testing"

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
