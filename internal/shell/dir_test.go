package shell

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderDirectorySortsFoldersBeforeFilesAndShowsIcons(t *testing.T) {
	tmp := t.TempDir()
	if err := os.Mkdir(filepath.Join(tmp, "zeta"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmp, "alpha"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "script.py"), []byte("print('hi')"), 0o644); err != nil {
		t.Fatalf("write py: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte("# hi"), 0o644); err != nil {
		t.Fatalf("write md: %v", err)
	}

	var out bytes.Buffer
	if err := renderDirectory(&out, tmp); err != nil {
		t.Fatalf("renderDirectory: %v", err)
	}
	got := out.String()

	if !strings.Contains(got, "📂 ") {
		t.Fatalf("expected directory header, got %q", got)
	}
	if !strings.Contains(got, "📁") || !strings.Contains(got, "🐍") || !strings.Contains(got, "📝") {
		t.Fatalf("expected folder and file icons, got %q", got)
	}

	alphaPos := strings.Index(got, "alpha"+string(os.PathSeparator))
	zetaPos := strings.Index(got, "zeta"+string(os.PathSeparator))
	readmePos := strings.Index(got, "README.md")
	pyPos := strings.Index(got, "script.py")
	if alphaPos == -1 || zetaPos == -1 || readmePos == -1 || pyPos == -1 {
		t.Fatalf("missing expected entries in output: %q", got)
	}
	if !(alphaPos < zetaPos && zetaPos < readmePos && readmePos < pyPos) {
		t.Fatalf("expected folders first and sorted names, got %q", got)
	}
}

func TestRunBuiltinDirUsageValidation(t *testing.T) {
	app := &App{}
	handled, code := app.runBuiltin("dir one two")
	if !handled || code != 1 {
		t.Fatalf("expected usage failure, handled=%v code=%d", handled, code)
	}
}
