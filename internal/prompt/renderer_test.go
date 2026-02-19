package prompt

import (
	"strings"
	"testing"
)

func TestRenderAppliesPaletteBadges(t *testing.T) {
	palette := map[string]string{
		"path_fg":   "#ffffff",
		"path_bg":   "#123456",
		"symbol_fg": "#abcdef",
	}

	out := Render([]string{"path"}, ">", palette, Context{WorkDir: "/tmp/project"})

	if !strings.Contains(out, "\x1b[38;2;255;255;255m") || !strings.Contains(out, "\x1b[48;2;18;52;86m") {
		t.Fatalf("expected path ANSI colors, got %q", out)
	}
	if !strings.Contains(out, " project ") {
		t.Fatalf("expected path badge style, got %q", out)
	}
	if !strings.Contains(out, "") {
		t.Fatalf("expected arrow separator, got %q", out)
	}
	if !strings.Contains(out, "\x1b[38;2;171;205;239m>") {
		t.Fatalf("expected symbol color, got %q", out)
	}
}

func TestAnsiRGBRejectsInvalidColor(t *testing.T) {
	if got := ansiRGB("38", "blue"); got != "" {
		t.Fatalf("expected empty for invalid color, got %q", got)
	}
}
