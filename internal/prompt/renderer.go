package prompt

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)

type Context struct {
	LastExitCode int
	WorkDir      string
}

func Render(segments []string, symbol string, ctx Context) string {
	parts := make([]string, 0, len(segments)+1)
	for _, segment := range segments {
		switch segment {
		case "user":
			if u, err := user.Current(); err == nil {
				parts = append(parts, u.Username)
			}
		case "path":
			wd := ctx.WorkDir
			if wd == "" {
				wd, _ = os.Getwd()
			}
			parts = append(parts, filepath.Base(wd))
		case "time":
			parts = append(parts, time.Now().Format("15:04:05"))
		case "exit_code":
			if ctx.LastExitCode != 0 {
				parts = append(parts, fmt.Sprintf("✗ %d", ctx.LastExitCode))
			}
		}
	}
	if symbol == "" {
		symbol = ">"
	}
	parts = append(parts, symbol)
	return strings.Join(parts, " ") + " "
}
