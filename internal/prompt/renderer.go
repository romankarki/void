package prompt

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Context struct {
	LastExitCode int
	WorkDir      string
}

func Render(segments []string, symbol string, palette map[string]string, ctx Context) string {
	parts := make([]string, 0, len(segments)+1)
	for _, segment := range segments {
		switch segment {
		case "user":
			if u, err := user.Current(); err == nil {
				parts = append(parts, styleSegment("user", u.Username, palette))
			}
		case "path":
			wd := ctx.WorkDir
			if wd == "" {
				wd, _ = os.Getwd()
			}
			parts = append(parts, styleSegment("path", filepath.Base(wd), palette))
		case "time":
			parts = append(parts, styleSegment("time", time.Now().Format("15:04:05"), palette))
		case "exit_code":
			if ctx.LastExitCode != 0 {
				parts = append(parts, styleSegment("exit_code", fmt.Sprintf("✗ %d", ctx.LastExitCode), palette))
			}
		}
	}
	if symbol == "" {
		symbol = ">"
	}
	parts = append(parts, styleSegment("symbol", symbol, palette))
	return strings.Join(parts, " ") + " "
}

func styleSegment(name, text string, palette map[string]string) string {
	fg := palette[name+"_fg"]
	bg := palette[name+"_bg"]

	styled := text
	if bg != "" {
		styled = " " + styled + " "
	}

	start := strings.Builder{}
	if fgCode := ansiRGB("38", fg); fgCode != "" {
		start.WriteString(fgCode)
	}
	if bgCode := ansiRGB("48", bg); bgCode != "" {
		start.WriteString(bgCode)
	}
	if start.Len() == 0 {
		return styled
	}
	return start.String() + styled + "\x1b[0m"
}

func ansiRGB(prefix, color string) string {
	if !strings.HasPrefix(color, "#") || len(color) != 7 {
		return ""
	}
	r, err := strconv.ParseInt(color[1:3], 16, 64)
	if err != nil {
		return ""
	}
	g, err := strconv.ParseInt(color[3:5], 16, 64)
	if err != nil {
		return ""
	}
	b, err := strconv.ParseInt(color[5:7], 16, 64)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("\x1b[%s;2;%d;%d;%dm", prefix, r, g, b)
}
