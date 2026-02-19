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
	if !strings.Contains(out, folderIcon) || !strings.Contains(out, "project") {
		t.Fatalf("expected folder breadcrumb output, got %q", out)
	}
	if strings.Count(out, "\x1b[48;2;18;52;86m") < 3 {
		t.Fatalf("expected each path part to be rendered as its own badge, got %q", out)
	}
	if !strings.Contains(out, "") {
		t.Fatalf("expected arrow separator, got %q", out)
	}
	if !strings.Contains(out, "\x1b[38;2;171;205;239m>") {
		t.Fatalf("expected symbol color, got %q", out)
	}
}

func TestRenderPathParts(t *testing.T) {
	got := renderPathParts("/Users/john/Desktop")
	want := []string{" /", " Users", " john", " Desktop"}

	if len(got) != len(want) {
		t.Fatalf("unexpected part count\nwant: %d\n got: %d", len(want), len(got))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected path part at index %d\nwant: %q\n got: %q", i, want[i], got[i])
		}
	}
}

func TestAnsiRGBRejectsInvalidColor(t *testing.T) {
	if got := ansiRGB("38", "blue"); got != "" {
		t.Fatalf("expected empty for invalid color, got %q", got)
	}
}
