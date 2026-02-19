package prompt

import (
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
	want := []string{folderIcon, folderIcon + " Users", folderIcon + " john", folderIcon + " Desktop"}

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
		t.Fatalf("expected %d derived colors from path_bg, got %d", defaultGradientSteps, len(got))
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

	if got[len(got)-1] != folderIcon+" s" {
		t.Fatalf("expected capped breadcrumb to end at s, got %q", got[len(got)-1])
	}
}

func TestPathGradientDerivesShadesFromPathBG(t *testing.T) {
	got := pathGradient(map[string]string{"path_bg": "#ff00aa"})
	if len(got) != defaultGradientSteps {
		t.Fatalf("expected %d derived colors, got %d", defaultGradientSteps, len(got))
	}
	if got[0] == got[len(got)-1] {
		t.Fatalf("expected gradient variation, got %#v", got)
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
	out := Render([]string{"path"}, "❯", palette, Context{WorkDir: "/tmp/project"})

	if strings.Contains(out, segmentSeparator) {
		t.Fatalf("expected ASCII separator fallback in vscode, got %q", out)
	}
	if strings.Contains(out, "❯") {
		t.Fatalf("expected ASCII symbol fallback in vscode, got %q", out)
	}
	if !strings.Contains(out, ">") {
		t.Fatalf("expected ASCII glyphs in fallback output, got %q", out)
	}
	if strings.Contains(out, folderIcon) {
		t.Fatalf("expected empty icon fallback in vscode, got %q", out)
	}
}
