//go:build !windows

package console

// EnableUTF8 is a no-op outside Windows.
func EnableUTF8() {}

