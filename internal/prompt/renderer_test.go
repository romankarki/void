package prompt

import (
	"os/user"
	"strings"
	"testing"
)

func TestRenderAppliesPaletteBadges(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("VOID_PROMPT_UNICODE", "1")

	palette := map[string]string{
		"path_fg":   "#ffffff",
		"path_bg_1": "#123456",
		"path_bg_2": "#345678",
		"symbol_fg": "#abcdef",
	}

	out := Render([]string{"path"}, ">", palette, Context{WorkDir: "/tmp/project"})

	if !strings.Contains(out, "\x1b[38;2;255;255;255m") || !strings.Contains(out, "\x1b[48;2;18;52;86m") {
		t.Fatalf("expected path ANSI colors, got %q", out)
	}
	if !strings.Contains(out, "\x1b[48;2;52;86;120m") {
		t.Fatalf("expected second path gradient color, got %q", out)
	}
	if !strings.Contains(out, folderIcon) || !strings.Contains(out, "project") {
		t.Fatalf("expected folder breadcrumb output, got %q", out)
	}
	if strings.Count(out, segmentSeparator) < 3 {
		t.Fatalf("expected each path part to be rendered as its own badge, got %q", out)
	}
	if !strings.Contains(out, segmentSeparator) {
		t.Fatalf("expected arrow separator, got %q", out)
	}
	if !strings.Contains(out, "\x1b[38;2;171;205;239m>") {
		t.Fatalf("expected symbol color, got %q", out)
	}
}

func TestRenderBreaksPromptAfterBadges(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("VOID_PROMPT_UNICODE", "1")

	out := Render([]string{"path"}, ">", map[string]string{"path_bg": "#123456"}, Context{WorkDir: "/tmp/project"})
	if !strings.Contains(out, "\n"+promptLinePrefix+">") {
		t.Fatalf("expected badge line break with prompt prefix, got %q", out)
	}
}

func TestRenderPathParts(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	got := renderPathParts("/Users/john/Desktop")
	want := []string{
		folderIcon,
		labelWithOptionalIcon(folderIcon, "Users"),
		labelWithOptionalIcon(folderIcon, "john"),
		labelWithOptionalIcon(folderIcon, "Desktop"),
	}

	if len(got) != len(want) {
		t.Fatalf("unexpected part count\nwant: %d\n got: %d", len(want), len(got))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected path part at index %d\nwant: %q\n got: %q", i, want[i], got[i])
		}
	}
}

func TestPathGradientFallbacks(t *testing.T) {
	if got := pathGradient(map[string]string{"path_bg": "#101010"}); len(got) != defaultGradientSteps {
		t.Fatalf("expected %d colors from path_bg pool, got %d", defaultGradientSteps, len(got))
	} else {
		assertUniqueColors(t, got)
		if !containsColor(got, "#101010") {
			t.Fatalf("expected configured path_bg to be included in palette, got %#v", got)
		}
	}

	palette := map[string]string{"path_bg_1": "#111111", "path_bg_2": "#222222"}
	got := pathGradient(palette)
	if len(got) != 2 || got[0] != "#111111" || got[1] != "#222222" {
		t.Fatalf("expected gradient palette, got %#v", got)
	}

	got = pathGradient(map[string]string{})
	if len(got) != defaultGradientSteps {
		t.Fatalf("expected %d default gradient colors, got %d", defaultGradientSteps, len(got))
	}
	assertUniqueColors(t, got)
}

func TestRenderPathSegmentsUsesSameForegroundColor(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	palette := map[string]string{
		"path_fg": "#ffd166",
		"path_bg": "#1f2937",
	}
	segments := renderPathSegments("/Users/Asus/Desktop", palette)
	if len(segments) < 3 {
		t.Fatalf("expected multiple path segments, got %d", len(segments))
	}
	if segments[0].fg != "#ffd166" {
		t.Fatalf("expected first segment to keep base path_fg, got %q", segments[0].fg)
	}
	if segments[1].fg != segments[0].fg {
		t.Fatalf("expected path segments to use the same foreground color, got %#v", segments)
	}
}

func TestRenderPathPartsCapsBreadcrumbs(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	got := renderPathParts("/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t/u/v")

	if len(got) != maxPathBreadcrumbs {
		t.Fatalf("expected %d breadcrumbs, got %d", maxPathBreadcrumbs, len(got))
	}

	if got[0] != folderIcon {
		t.Fatalf("expected root breadcrumb first, got %q", got[0])
	}

	if got[len(got)-1] != labelWithOptionalIcon(folderIcon, "s") {
		t.Fatalf("expected capped breadcrumb to end at s, got %q", got[len(got)-1])
	}
}

func TestPathGradientUsesVariedColorsFromPool(t *testing.T) {
	got := pathGradient(map[string]string{"path_bg": "#ff00aa"})
	if len(got) != defaultGradientSteps {
		t.Fatalf("expected %d colors, got %d", defaultGradientSteps, len(got))
	}
	assertUniqueColors(t, got)
	if !containsColor(got, "#ff00aa") {
		t.Fatalf("expected configured path_bg to be included, got %#v", got)
	}
}

func TestRenderExitCodeUsesErrorLabel(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("VOID_PROMPT_UNICODE", "1")

	out := Render([]string{"exit_code"}, ">", map[string]string{"exit_code_fg": "#ffffff", "exit_code_bg": "#ff0000"}, Context{LastExitCode: 1})
	if !strings.Contains(out, "1 error") {
		t.Fatalf("expected singular error label, got %q", out)
	}

	out = Render([]string{"exit_code"}, ">", map[string]string{"exit_code_fg": "#ffffff", "exit_code_bg": "#ff0000"}, Context{LastExitCode: 2})
	if !strings.Contains(out, "2 errors") {
		t.Fatalf("expected plural error label, got %q", out)
	}
}

func TestRenderUserSegmentUsesActivationLabelOverride(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("VOID_PROMPT_UNICODE", "1")
	t.Setenv("VOID_ACTIVE_LABEL", "my-app")
	t.Setenv("CONDA_DEFAULT_ENV", "ignored-env")

	out := Render([]string{"user"}, ">", map[string]string{}, Context{})
	if !strings.Contains(out, "MY-APP") {
		t.Fatalf("expected activation label to replace username, got %q", out)
	}
	if strings.Contains(out, "IGNORED-ENV") {
		t.Fatalf("expected VOID_ACTIVE_LABEL precedence, got %q", out)
	}
}

func TestRenderUserSegmentUsesVirtualEnvPrompt(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("VOID_PROMPT_UNICODE", "1")
	t.Setenv("VOID_ACTIVE_LABEL", "")
	t.Setenv("VIRTUAL_ENV_PROMPT", "(api-env) ")

	out := Render([]string{"user"}, ">", map[string]string{}, Context{})
	if !strings.Contains(out, "API-ENV") {
		t.Fatalf("expected virtual env prompt label, got %q", out)
	}
}

func TestRenderUserSegmentUsesVirtualEnvPath(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "")
	t.Setenv("VOID_PROMPT_UNICODE", "1")
	t.Setenv("VOID_ACTIVE_LABEL", "")
	t.Setenv("VIRTUAL_ENV_PROMPT", "")
	t.Setenv("CONDA_DEFAULT_ENV", "")
	t.Setenv("VIRTUAL_ENV", `/tmp/repo/.venv`)

	out := Render([]string{"user"}, ">", map[string]string{}, Context{})
	if !strings.Contains(out, ".VENV") {
		t.Fatalf("expected virtual env directory name, got %q", out)
	}
}

func TestResolveUserSegmentLabelUsesGitBranchWhenAvailable(t *testing.T) {
	t.Setenv("VOID_ACTIVE_LABEL", "")
	t.Setenv("VIRTUAL_ENV_PROMPT", "")
	t.Setenv("CONDA_DEFAULT_ENV", "")
	t.Setenv("VIRTUAL_ENV", "")

	origGit := resolveGitBranchForDir
	origUser := resolveCurrentUser
	origHost := resolveHostname
	t.Cleanup(func() {
		resolveGitBranchForDir = origGit
		resolveCurrentUser = origUser
		resolveHostname = origHost
	})

	resolveGitBranchForDir = func(string) (string, error) { return "main", nil }
	resolveCurrentUser = func() (*user.User, error) { return &user.User{Username: "asus"}, nil }
	resolveHostname = func() (string, error) { return "laptop", nil }

	got := resolveUserSegmentLabel("/tmp/repo")
	if got != "main" {
		t.Fatalf("expected git branch label, got %q", got)
	}
}

func TestResolveUserSegmentLabelCombinesVenvAndGitBranch(t *testing.T) {
	t.Setenv("VOID_ACTIVE_LABEL", "")
	t.Setenv("VIRTUAL_ENV_PROMPT", "(api-env)")
	t.Setenv("CONDA_DEFAULT_ENV", "")
	t.Setenv("VIRTUAL_ENV", "")

	origGit := resolveGitBranchForDir
	t.Cleanup(func() {
		resolveGitBranchForDir = origGit
	})
	resolveGitBranchForDir = func(string) (string, error) { return "main", nil }

	got := resolveUserSegmentLabel("/tmp/repo")
	if got != "API-ENV | main" {
		t.Fatalf("expected venv + git branch label, got %q", got)
	}
}

func TestResolveUserSegmentLabelFallsBackToSystemIdentity(t *testing.T) {
	t.Setenv("VOID_ACTIVE_LABEL", "")
	t.Setenv("VIRTUAL_ENV_PROMPT", "")
	t.Setenv("CONDA_DEFAULT_ENV", "")
	t.Setenv("VIRTUAL_ENV", "")

	origGit := resolveGitBranchForDir
	origUser := resolveCurrentUser
	origHost := resolveHostname
	t.Cleanup(func() {
		resolveGitBranchForDir = origGit
		resolveCurrentUser = origUser
		resolveHostname = origHost
	})

	resolveGitBranchForDir = func(string) (string, error) { return "", nil }
	resolveCurrentUser = func() (*user.User, error) { return &user.User{Username: "asus"}, nil }
	resolveHostname = func() (string, error) { return "laptop", nil }

	got := resolveUserSegmentLabel("")
	if got != "LAPTOP\\ASUS" {
		t.Fatalf("expected system identity fallback, got %q", got)
	}
}

func TestAnsiRGBRejectsInvalidColor(t *testing.T) {
	if got := ansiRGB("38", "blue"); got != "" {
		t.Fatalf("expected empty for invalid color, got %q", got)
	}
}

func TestRenderFallsBackToASCIIWhenDisabled(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "vscode")
	t.Setenv("VOID_PROMPT_UNICODE", "0")

	palette := map[string]string{
		"path_fg":   "#ffffff",
		"path_bg_1": "#123456",
		"path_bg_2": "#345678",
	}
	out := Render([]string{"path"}, "\u276f", palette, Context{WorkDir: "/tmp/project"})

	if segmentSeparator != "" && strings.Contains(out, segmentSeparator) {
		t.Fatalf("expected ASCII separator fallback in vscode, got %q", out)
	}
	if strings.Contains(out, "\u276f") {
		t.Fatalf("expected ASCII symbol fallback in vscode, got %q", out)
	}
	if !strings.Contains(out, ">") {
		t.Fatalf("expected ASCII glyphs in fallback output, got %q", out)
	}
	if strings.Contains(out, folderIcon) {
		t.Fatalf("expected empty icon fallback in vscode, got %q", out)
	}
}

func TestRenderUsesUnicodeByDefaultInVSCode(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "vscode")
	t.Setenv("VOID_PROMPT_UNICODE", "")

	palette := map[string]string{
		"path_fg":   "#ffffff",
		"path_bg_1": "#123456",
		"path_bg_2": "#345678",
	}
	out := Render([]string{"path"}, "\u276f", palette, Context{WorkDir: "/tmp/project"})

	if segmentSeparator != "" && !strings.Contains(out, segmentSeparator) {
		t.Fatalf("expected unicode separator by default in vscode, got %q", out)
	}
	if !strings.Contains(out, "\u276f") {
		t.Fatalf("expected unicode symbol by default in vscode, got %q", out)
	}
	if !strings.Contains(out, folderIcon) {
		t.Fatalf("expected icons to be shown by default in vscode unicode mode, got %q", out)
	}
}

func TestRenderKeepsIconsInVSCodeWhenUnicodeEnabled(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "vscode")
	t.Setenv("VOID_PROMPT_UNICODE", "1")

	out := Render([]string{"path"}, ">", map[string]string{"path_bg": "#123456"}, Context{WorkDir: "/tmp/project"})
	if isLikelyMojibakeIcon(folderIcon) {
		if strings.Contains(out, folderIcon) {
			t.Fatalf("expected mojibake icon fallback in vscode, got %q", out)
		}
		return
	}
	if !strings.Contains(out, folderIcon) {
		t.Fatalf("expected glyph icons in vscode when unicode is enabled, got %q", out)
	}
}

func TestPromptIconFallsBackForMojibakeInVSCode(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "vscode")
	t.Setenv("VOID_PROMPT_UNICODE", "1")

	if got := promptIcon("\u00f0\u0178\u2018\u00a4"); got != "" {
		t.Fatalf("expected mojibake icon fallback, got %q", got)
	}
	if got := promptIcon("\u00c3\u00b0\u00c5\u00b8\u00e2\u20ac\u02dc\u00c2\u00a4"); got != "" {
		t.Fatalf("expected mojibake icon fallback, got %q", got)
	}
}

func TestPromptIconKeepsValidUnicodeInVSCode(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "vscode")
	t.Setenv("VOID_PROMPT_UNICODE", "1")
	if got := promptIcon("\U0001F4C2"); got == "" {
		t.Fatalf("expected valid unicode icon to be kept in vscode")
	}
}

func assertUniqueColors(t *testing.T, colors []string) {
	t.Helper()
	seen := map[string]struct{}{}
	for _, color := range colors {
		normalized := strings.ToLower(color)
		if _, ok := seen[normalized]; ok {
			t.Fatalf("expected unique colors, got duplicate %q in %#v", color, colors)
		}
		seen[normalized] = struct{}{}
	}
}

func containsColor(colors []string, target string) bool {
	for _, color := range colors {
		if strings.EqualFold(color, target) {
			return true
		}
	}
	return false
}
