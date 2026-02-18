package history

import (
	"path/filepath"
	"testing"
)

func TestStoreDedupAndMax(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	s, err := New(path, 2)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	s.Add("echo one")
	s.Add("echo one")
	s.Add("echo two")
	s.Add("echo three")

	entries := s.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0] != "echo two" || entries[1] != "echo three" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}
