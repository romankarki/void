package prompt

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const folderIcon = "󰉋"

type Context struct {
	LastExitCode int
	WorkDir      string
}

type renderSegment struct {
	text string
	fg   string
	bg   string
}

func Render(segments []string, symbol string, palette map[string]string, ctx Context) string {
	rendered := make([]renderSegment, 0, len(segments)+1)
	for _, segment := range segments {
		switch segment {
		case "user":
			if u, err := user.Current(); err == nil {
				rendered = append(rendered, newSegment("user", u.Username, palette))
			}
		case "path":
			wd := ctx.WorkDir
			if wd == "" {
				wd, _ = os.Getwd()
			}
			rendered = append(rendered, newSegment("path", renderPathBreadcrumbs(wd), palette))
		case "time":
			rendered = append(rendered, newSegment("time", time.Now().Format("15:04:05"), palette))
		case "exit_code":
			if ctx.LastExitCode != 0 {
				rendered = append(rendered, newSegment("exit_code", fmt.Sprintf("✗ %d", ctx.LastExitCode), palette))
			}
		}
	}
	if symbol == "" {
		symbol = ">"
	}
	rendered = append(rendered, newSegment("symbol", symbol, palette))

	return renderWithArrows(rendered)
}

func renderPathBreadcrumbs(wd string) string {
	if wd == "" {
		return folderIcon
	}

	clean := filepath.Clean(wd)
	vol := filepath.VolumeName(clean)
	remainder := strings.TrimPrefix(clean, vol)

	sep := string(filepath.Separator)
	if sep == "\\" {
		remainder = strings.ReplaceAll(remainder, "/", "\\")
	}
	parts := strings.FieldsFunc(remainder, func(r rune) bool {
		return r == '/' || r == '\\'
	})

	crumbs := make([]string, 0, len(parts)+1)
	if vol != "" {
		crumbs = append(crumbs, fmt.Sprintf("%s %s", folderIcon, vol))
	} else if strings.HasPrefix(clean, sep) || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "\\") {
		if runtime.GOOS == "windows" {
			crumbs = append(crumbs, folderIcon)
		} else {
			crumbs = append(crumbs, fmt.Sprintf("%s /", folderIcon))
		}
	}

	for _, part := range parts {
		crumbs = append(crumbs, fmt.Sprintf("%s %s", folderIcon, part))
	}

	if len(crumbs) == 0 {
		return folderIcon
	}

	return strings.Join(crumbs, " › ")
}

func newSegment(name, text string, palette map[string]string) renderSegment {
	return renderSegment{
		text: text,
		fg:   palette[name+"_fg"],
		bg:   palette[name+"_bg"],
	}
}

func renderWithArrows(segments []renderSegment) string {
	var out strings.Builder
	for i, segment := range segments {
		nextBG := ""
		if i+1 < len(segments) {
			nextBG = segments[i+1].bg
		}

		text := segment.text
		if segment.bg != "" {
			text = " " + text + " "
		}

		if start := ansiSeq(segment.fg, segment.bg); start != "" {
			out.WriteString(start)
			out.WriteString(text)
			out.WriteString("\x1b[0m")
		} else {
			out.WriteString(text)
		}

		if segment.bg != "" {
			arrowStyle := ansiSeq(segment.bg, nextBG)
			if arrowStyle != "" {
				out.WriteString(arrowStyle)
				out.WriteString("")
				out.WriteString("\x1b[0m")
			} else {
				out.WriteString("")
			}
		} else if i+1 < len(segments) {
			out.WriteByte(' ')
		}
	}
	out.WriteByte(' ')
	return out.String()
}

func ansiSeq(fg, bg string) string {
	var seq strings.Builder
	if fgCode := ansiRGB("38", fg); fgCode != "" {
		seq.WriteString(fgCode)
	}
	if bgCode := ansiRGB("48", bg); bgCode != "" {
		seq.WriteString(bgCode)
	}
	return seq.String()
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
