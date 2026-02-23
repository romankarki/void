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
	return `$utf8NoBom = [System.Text.UTF8Encoding]::new($false)
[Console]::InputEncoding = $utf8NoBom
[Console]::OutputEncoding = $utf8NoBom
$OutputEncoding = $utf8NoBom

$__voidBinPath = Join-Path $env:LOCALAPPDATA "Void\bin"
if (($env:PATH -split ';' | Where-Object { $_ -ieq $__voidBinPath }).Count -eq 0) {
    $env:PATH = $__voidBinPath + ';' + $env:PATH
}

function __void_render_prompt([int]$code, [string]$workdir) {
    try {
        $psi = New-Object System.Diagnostics.ProcessStartInfo
        $psi.FileName = "void"
        $escapedWorkdir = $workdir -replace '"', '\"'
        $psi.Arguments = ('prompt --last-exit-code {0} --workdir "{1}"' -f $code, $escapedWorkdir)
        $psi.UseShellExecute = $false
        $psi.RedirectStandardOutput = $true

        $process = New-Object System.Diagnostics.Process
        $process.StartInfo = $psi
        [void]$process.Start()

        $stdout = New-Object System.IO.MemoryStream
        $process.StandardOutput.BaseStream.CopyTo($stdout)
        $process.WaitForExit()

        return [System.Text.Encoding]::UTF8.GetString($stdout.ToArray())
    } catch {
        return "> "
    }
}

$global:__void_last_exit = 0
function prompt {
    $lastCommandSucceeded = $?
    $code = $global:LASTEXITCODE
    if ($null -eq $code) { $code = 0 }
    if (-not $lastCommandSucceeded -and $code -eq 0) { $code = 1 }
    $env:VOID_LAST_EXIT_CODE = "$code"
    if (-not $lastCommandSucceeded -or $code -ne 0) {
        $lastMessage = ""
        if ($Error.Count -gt 0 -and $null -ne $Error[0]) {
            $lastMessage = ($Error[0] | Out-String).Trim()
        }
        if ([string]::IsNullOrWhiteSpace($lastMessage)) {
            $lastMessage = ("command exited with code {0}" -f $code)
        }
        $env:VOID_LAST_ERROR = $lastMessage
    } else {
        $env:VOID_LAST_ERROR = ""
    }
    $global:__void_last_exit = $code
    __void_render_prompt -code $code -workdir $PWD.Path
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
