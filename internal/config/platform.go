package config

import "runtime"

func defaultShell() string {
	if runtime.GOOS == "windows" {
		return "cmd.exe"
	}
	return "sh"
}
