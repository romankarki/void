package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeShell(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "pwsh", want: "powershell"},
		{in: "powershell", want: "powershell"},
		{in: "bash", want: "bash"},
		{in: "zsh", want: "zsh"},
		{in: "cmd", want: "cmd"},
		{in: "unknown", want: ""},
	}

	for _, tc := range cases {
		if got := normalizeShell(tc.in); got != tc.want {
			t.Fatalf("normalizeShell(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestAppendBlockIfMissingIsIdempotent(t *testing.T) {
	tmp := t.TempDir()
	profile := filepath.Join(tmp, "profile.txt")
	block := profileMarkerStart + "\nhello\n" + profileMarkerEnd + "\n"

	if err := appendBlockIfMissing(profile, block, profileMarkerStart); err != nil {
		t.Fatalf("first append failed: %v", err)
	}
	if err := appendBlockIfMissing(profile, block, profileMarkerStart); err != nil {
		t.Fatalf("second append failed: %v", err)
	}

	contents, err := os.ReadFile(profile)
	if err != nil {
		t.Fatalf("read profile: %v", err)
	}
	if count := strings.Count(string(contents), profileMarkerStart); count != 1 {
		t.Fatalf("expected one marker, got %d in %q", count, string(contents))
	}
}

func TestAppendBlockIfMissingReplacesExistingMarkedBlock(t *testing.T) {
	tmp := t.TempDir()
	profile := filepath.Join(tmp, "profile.txt")

	original := profileMarkerStart + "\nold content\n" + profileMarkerEnd + "\n"
	if err := os.WriteFile(profile, []byte(original), 0o644); err != nil {
		t.Fatalf("write profile: %v", err)
	}

	replacement := profileMarkerStart + "\nnew content\n" + profileMarkerEnd + "\n"
	if err := appendBlockIfMissing(profile, replacement, profileMarkerStart); err != nil {
		t.Fatalf("replace block failed: %v", err)
	}

	contents, err := os.ReadFile(profile)
	if err != nil {
		t.Fatalf("read profile: %v", err)
	}
	got := string(contents)
	if !strings.Contains(got, "new content") || strings.Contains(got, "old content") {
		t.Fatalf("expected block replacement, got %q", got)
	}
}

func TestPathContainsEntry(t *testing.T) {
	pathValue := strings.Join([]string{`C:\Tools`, `C:\Users\ASUS\AppData\Local\Void\bin`}, string(os.PathListSeparator))
	if !pathContainsEntry(pathValue, `C:\Users\ASUS\AppData\Local\Void\bin`) {
		t.Fatalf("expected path entry to be detected")
	}
	if pathContainsEntry(pathValue, `C:\missing`) {
		t.Fatalf("did not expect missing entry to be detected")
	}
}

func TestAppendPathEntry(t *testing.T) {
	got := appendPathEntry("", "C:\\Void\\bin")
	if got != "C:\\Void\\bin" {
		t.Fatalf("expected only entry, got %q", got)
	}
	got = appendPathEntry("C:\\Tools", "C:\\Void\\bin")
	want := "C:\\Tools" + string(os.PathListSeparator) + "C:\\Void\\bin"
	if got != want {
		t.Fatalf("appendPathEntry mismatch: got %q want %q", got, want)
	}
}
