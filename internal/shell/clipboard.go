package shell

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

var copyTextToClipboard = copyToClipboard

func copyToClipboard(text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("empty text")
	}

	switch runtime.GOOS {
	case "windows":
		return runClipboardCommand("cmd", []string{"/c", "clip"}, text)
	case "darwin":
		return runClipboardCommand("pbcopy", nil, text)
	default:
		linuxCandidates := []struct {
			name string
			args []string
		}{
			{name: "wl-copy"},
			{name: "xclip", args: []string{"-selection", "clipboard"}},
			{name: "xsel", args: []string{"--clipboard", "--input"}},
		}

		for _, candidate := range linuxCandidates {
			if _, err := exec.LookPath(candidate.name); err != nil {
				continue
			}
			if err := runClipboardCommand(candidate.name, candidate.args, text); err == nil {
				return nil
			}
		}
		return fmt.Errorf("no clipboard utility available (tried wl-copy, xclip, xsel)")
	}
}

func runClipboardCommand(name string, args []string, text string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(text)
	if out, err := cmd.CombinedOutput(); err != nil {
		if len(out) == 0 {
			return err
		}
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
