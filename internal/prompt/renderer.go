package prompt

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	// userIcon         = "💻" //"\U0001F464"
	// driveIcon        = "\U0001F4BE"
	// folderIcon       = "\U0001F4C2"
	// timeIcon         = "🕜" //"\u23F0"
	// errorIcon        = "\u26A0\uFE0F"
	segmentSeparator = "\ue0b0"

	userIcon   = "(-<)"
	driveIcon  = ""
	folderIcon = ""
	timeIcon   = "⌛"
	errorIcon  = ""
	// segmentSeparator      = ""
	segmentSeparatorASCII = ""
	promptLinePrefix      = "| "
	iconLabelGap          = "  "

	maxPathBreadcrumbs   = 20
	defaultGradientSteps = 20
)

type Context struct {
	LastExitCode int
	WorkDir      string
}

type renderSegment struct {
	text string
	fg   string
	bg   string
}

var (
	resolveGitBranchForDir = detectGitBranchForDir
	resolveGitDirtyForDir  = detectGitDirtyForDir
	resolveCurrentUser     = user.Current
	resolveHostname        = os.Hostname
)

func Render(segments []string, symbol string, palette map[string]string, ctx Context) string {
	unicodeOK := supportsUnicodePrompt()
	userPromptIcon := promptIcon(userIcon)
	timePromptIcon := promptIcon(timeIcon)
	errorPromptIcon := promptIcon(errorIcon)

	rendered := make([]renderSegment, 0, len(segments))
	for _, segment := range segments {
		switch segment {
		case "user":
			if userLabel := resolveUserSegmentLabel(ctx.WorkDir); userLabel != "" {
				rendered = append(rendered, newSegment("user", labelWithOptionalIcon(userPromptIcon, userLabel), palette))
			}
		case "path":
			wd := ctx.WorkDir
			if wd == "" {
				wd, _ = os.Getwd()
			}
			rendered = append(rendered, renderPathSegments(wd, palette)...)
		case "time":
			rendered = append(rendered, newSegment("time", labelWithOptionalIcon(timePromptIcon, time.Now().Format("3:04 PM")), palette))
		case "exit_code":
			if ctx.LastExitCode != 0 {
				suffix := "errors"
				if ctx.LastExitCode == 1 {
					suffix = "error"
				}
				rendered = append(rendered, newSegment("exit_code", labelWithOptionalIcon(errorPromptIcon, fmt.Sprintf("%d %s", ctx.LastExitCode, suffix)), palette))
			}
		}
	}
	if symbol == "" {
		symbol = ">"
	}
	if !unicodeOK && !isASCII(symbol) {
		symbol = ">"
	}
	symbolSegment := newSegment("symbol", symbol, palette)

	if len(rendered) == 0 {
		return renderWithArrows([]renderSegment{symbolSegment}, unicodeOK)
	}

	badges := strings.TrimRight(renderWithArrows(rendered, unicodeOK), " ")
	promptSymbol := strings.TrimLeft(renderWithArrows([]renderSegment{symbolSegment}, unicodeOK), " ")

	return badges + "\n" + promptLinePrefix + promptSymbol
}

func renderPathParts(wd string) []string {
	drivePromptIcon := promptIcon(driveIcon)
	folderPromptIcon := promptIcon(folderIcon)

	if wd == "" {
		root := folderPromptIcon
		if root == "" {
			return []string{string(filepath.Separator)}
		}
		return []string{root}
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
		crumbs = append(crumbs, labelWithOptionalIcon(drivePromptIcon, vol))
	} else if strings.HasPrefix(clean, sep) || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "\\") {
		root := folderPromptIcon
		if root == "" {
			if runtime.GOOS == "windows" {
				root = `\`
			} else {
				root = "/"
			}
		}
		crumbs = append(crumbs, root)
	}

	for _, part := range parts {
		crumbs = append(crumbs, labelWithOptionalIcon(folderPromptIcon, part))
		if len(crumbs) >= maxPathBreadcrumbs {
			break
		}
	}

	if len(crumbs) == 0 {
		root := folderPromptIcon
		if root == "" {
			return []string{string(filepath.Separator)}
		}
		return []string{root}
	}

	return crumbs
}

func renderPathSegments(wd string, palette map[string]string) []renderSegment {
	parts := renderPathParts(wd)
	segments := make([]renderSegment, 0, len(parts))
	pathColors := pathGradient(palette)
	for i, part := range parts {
		segment := newSegment("path", part, palette)
		segment.bg = pathColors[i%len(pathColors)]
		segments = append(segments, segment)
	}

	return segments
}

func resolveUserSegmentLabel(workDir string) string {
	envLabel := resolveActiveEnvLabel()
	branchLabel := resolveGitBranchLabel(workDir)
	dirtyLabel := ""
	if branchLabel != "" {
		dirtyLabel = resolveGitDirtyLabel(workDir)
	}

	if branchLabel != "" {
		if dirtyLabel != "" {
			branchLabel = branchLabel + " " + dirtyLabel
		}
		if envLabel != "" {
			return strings.ToUpper(envLabel) + " | " + branchLabel
		}
		return branchLabel
	}
	if envLabel != "" {
		return strings.ToUpper(envLabel)
	}
	return resolveSystemIdentityLabel()
}

func resolveActiveEnvLabel() string {
	if label := strings.TrimSpace(os.Getenv("VOID_ACTIVE_LABEL")); label != "" {
		return label
	}

	if label := parseVirtualEnvPromptLabel(os.Getenv("VIRTUAL_ENV_PROMPT")); label != "" {
		return label
	}

	if label := strings.TrimSpace(os.Getenv("CONDA_DEFAULT_ENV")); label != "" {
		return label
	}

	if venvPath := strings.TrimSpace(os.Getenv("VIRTUAL_ENV")); venvPath != "" {
		base := strings.TrimSpace(filepath.Base(filepath.Clean(venvPath)))
		if base != "" && base != "." && base != string(filepath.Separator) {
			return base
		}
	}

	return ""
}

func resolveGitBranchLabel(workDir string) string {
	dir := strings.TrimSpace(workDir)
	if dir == "" {
		if wd, err := os.Getwd(); err == nil {
			dir = wd
		}
	}
	if dir == "" {
		return ""
	}

	branch, err := resolveGitBranchForDir(dir)
	if err != nil {
		return ""
	}
	branch = strings.TrimSpace(branch)
	if branch == "" || strings.EqualFold(branch, "HEAD") {
		return ""
	}
	return branch
}

func resolveSystemIdentityLabel() string {
	username := ""
	if u, err := resolveCurrentUser(); err == nil && u != nil {
		username = strings.TrimSpace(u.Username)
	}
	if username == "" {
		username = strings.TrimSpace(os.Getenv("USERNAME"))
	}
	if username == "" {
		username = strings.TrimSpace(os.Getenv("USER"))
	}

	host := ""
	if h, err := resolveHostname(); err == nil {
		host = strings.TrimSpace(h)
	}

	switch {
	case username != "":
		if strings.Contains(username, `\`) || strings.Contains(username, "@") {
			return strings.ToUpper(username)
		}
		if host != "" {
			return strings.ToUpper(host + `\` + username)
		}
		return strings.ToUpper(username)
	case host != "":
		return strings.ToUpper(host)
	default:
		return ""
	}
}

func detectGitBranchForDir(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func resolveGitDirtyLabel(workDir string) string {
	dir := strings.TrimSpace(workDir)
	if dir == "" {
		if wd, err := os.Getwd(); err == nil {
			dir = wd
		}
	}
	if dir == "" {
		return ""
	}

	dirty, err := resolveGitDirtyForDir(dir)
	if err != nil || !dirty {
		return ""
	}

	return "[.]"
}

func detectGitDirtyForDir(dir string) (bool, error) {
	cmd := exec.Command("git", "-C", dir, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(output)) != "", nil
}

func parseVirtualEnvPromptLabel(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	if strings.HasPrefix(trimmed, "(") {
		if end := strings.Index(trimmed, ")"); end > 1 {
			if label := strings.TrimSpace(trimmed[1:end]); label != "" {
				return label
			}
		}
	}

	return trimmed
}

func pathGradient(palette map[string]string) []string {
	gradient := make([]string, 0, defaultGradientSteps)
	for i := 1; i <= defaultGradientSteps; i++ {
		if color := palette[fmt.Sprintf("path_bg_%d", i)]; color != "" {
			gradient = append(gradient, color)
		}
	}
	if len(gradient) > 0 {
		return gradient
	}

	defaultColors := []string{
		"#3b82f6", "#22c55e", "#a855f7", "#f59e0b", "#06b6d4",
		"#ef4444", "#84cc16", "#ec4899", "#6366f1", "#14b8a6",
		"#f97316", "#8b5cf6", "#10b981", "#eab308", "#0ea5e9",
		"#d946ef", "#65a30d", "#fb7185", "#2563eb", "#16a34a",
	}

	base := strings.TrimSpace(palette["path_bg"])
	if ansiRGB("48", base) == "" {
		base = ""
	}
	if base == "" {
		return shuffledColors(defaultColors)
	}

	colors := make([]string, 0, defaultGradientSteps)
	colors = append(colors, strings.ToLower(base))
	for _, color := range defaultColors {
		if strings.EqualFold(color, base) {
			continue
		}
		colors = append(colors, color)
		if len(colors) == defaultGradientSteps {
			break
		}
	}

	return shuffledColors(colors)
}

func shuffledColors(colors []string) []string {
	shuffled := append([]string(nil), colors...)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled
}

func newSegment(name, text string, palette map[string]string) renderSegment {
	fg := palette[name+"_fg"]
	if name == "user" {
		fg = "#ffffff"
	}

	return renderSegment{
		text: text,
		fg:   fg,
		bg:   palette[name+"_bg"],
	}
}

func renderWithArrows(segments []renderSegment, unicodeOK bool) string {
	var out strings.Builder
	separator := segmentSeparator
	if !unicodeOK {
		separator = segmentSeparatorASCII
	}
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
				out.WriteString(separator)
				out.WriteString("\x1b[0m")
			} else {
				out.WriteString(separator)
			}
		} else if i+1 < len(segments) {
			out.WriteByte(' ')
		}
	}
	out.WriteByte(' ')
	return out.String()
}

func supportsUnicodePrompt() bool {
	override := strings.TrimSpace(strings.ToLower(os.Getenv("VOID_PROMPT_UNICODE")))
	switch override {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return true
}

func promptIcon(icon string) string {
	if !supportsUnicodePrompt() {
		return ""
	}
	if isVSCodeTerminal() && envBool("VOID_VSCODE_EMPTY_ICONS") {
		return ""
	}
	if isVSCodeTerminal() && isLikelyMojibakeIcon(icon) {
		return ""
	}
	return icon
}

func isVSCodeTerminal() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("TERM_PROGRAM")), "vscode")
}

func envBool(name string) bool {
	switch strings.TrimSpace(strings.ToLower(os.Getenv(name))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func isLikelyMojibakeIcon(icon string) bool {
	s := strings.TrimSpace(icon)
	if s == "" {
		return false
	}

	// Common mojibake runes when UTF-8 glyph bytes are decoded as Windows-1252.
	// Values are code points to keep this file encoding-safe.
	for _, r := range s {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return true
		}
		switch r {
		case 0x00C3, // Ã
			0x00C2, // Â
			0x00E2, // â
			0x00F0, // ð
			0x0178, // Ÿ
			0x00EF, // ï
			0x00B8, // ¸
			0x2018, // ‘
			0x00A4, // ¤
			0x20AC, // €
			0x2122, // ™
			0x0153, // œ
			0x0161, // š
			0x017E: // ž
			return true
		}
	}
	return false
}

func labelWithOptionalIcon(icon, label string) string {
	if icon == "" {
		return label
	}
	if label == "" {
		return icon
	}
	return icon + iconLabelGap + label
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
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
