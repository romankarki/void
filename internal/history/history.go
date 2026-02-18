package history

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	path    string
	maxSize int
	entries []string
	seen    map[string]struct{}
}

func New(path string, maxSize int) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	s := &Store{path: path, maxSize: maxSize, seen: map[string]struct{}{}}
	if err := s.Load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Load() error {
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	s.entries = nil
	s.seen = map[string]struct{}{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		s.Add(line)
	}
	return scanner.Err()
}

func (s *Store) Add(cmd string) {
	if cmd == "" {
		return
	}
	if _, ok := s.seen[cmd]; ok {
		return
	}
	s.entries = append(s.entries, cmd)
	s.seen[cmd] = struct{}{}
	if len(s.entries) > s.maxSize {
		old := s.entries[0]
		delete(s.seen, old)
		s.entries = s.entries[1:]
	}
}

func (s *Store) Entries() []string {
	out := make([]string, len(s.entries))
	copy(out, s.entries)
	return out
}

func (s *Store) Save() error {
	f, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, line := range s.entries {
		if _, err := w.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return w.Flush()
}
