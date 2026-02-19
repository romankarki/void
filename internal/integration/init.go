package integration

import (
	"fmt"
	"strings"
)

// InitScript returns a shell-specific setup snippet that wires Void's prompt
// renderer into external terminals.
func InitScript(shell string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "powershell", "pwsh":
		return powershellScript(), nil
	case "bash":
		return bashScript(), nil
	case "zsh":
		return zshScript(), nil
	case "cmd", "cmd.exe":
		return cmdScript(), nil
	default:
		return "", fmt.Errorf("unsupported shell %q (supported: powershell, bash, zsh, cmd)", shell)
	}
}

func powershellScript() string {
	return `$global:__void_last_exit = 0
function prompt {
    $code = $global:LASTEXITCODE
    if ($null -eq $code) { $code = 0 }
    $global:__void_last_exit = $code
    void prompt --last-exit-code $code --workdir "$PWD"
}`
}

func bashScript() string {
	return `__void_prompt() {
  local code="$?"
  PS1="$(void prompt --last-exit-code "$code" --workdir "$PWD")"
}
PROMPT_COMMAND=__void_prompt`
}

func zshScript() string {
	return `function precmd() {
  local code="$?"
  PROMPT="$(void prompt --last-exit-code "$code" --workdir "$PWD")"
}`
}

func cmdScript() string {
	return `:: CMD does not expose a native pre-prompt hook for running external programs.
:: This fallback keeps path + time visible in plain CMD.
PROMPT $P $T $G `
}
