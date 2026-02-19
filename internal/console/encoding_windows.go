//go:build windows

package console

import "syscall"

const utf8CodePage = 65001

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleOutputCP  = kernel32.NewProc("SetConsoleOutputCP")
	procSetConsoleInputCP   = kernel32.NewProc("SetConsoleCP")
)

// EnableUTF8 switches the active console code pages to UTF-8 so Unicode
// prompt glyphs render correctly in terminals backed by Windows ConPTY.
func EnableUTF8() {
	_, _, _ = procSetConsoleOutputCP.Call(uintptr(utf8CodePage))
	_, _, _ = procSetConsoleInputCP.Call(uintptr(utf8CodePage))
}

