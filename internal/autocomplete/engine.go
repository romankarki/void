package autocomplete

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Engine struct {
	builtins []string
}

func New() *Engine {
	return &Engine{builtins: []string{"cd", "dir", "copy", "del", "exit", "git", "go", "npm", "docker", "python"}}
}

func (e *Engine) Complete(prefix string, history []string) []string {
	all := map[string]struct{}{}
	for _, b := range e.builtins {
		all[b] = struct{}{}
	}
	for _, h := range history {
		all[strings.Fields(h)[0]] = struct{}{}
	}
	for _, c := range fromPath() {
		all[c] = struct{}{}
	}

	matches := make([]string, 0)
	for c := range all {
		if strings.HasPrefix(strings.ToLower(c), strings.ToLower(prefix)) {
			matches = append(matches, c)
		}
	}
	sort.Strings(matches)
	if len(matches) > 20 {
		matches = matches[:20]
	}
	return matches
}

func fromPath() []string {
	path := os.Getenv("PATH")
	if path == "" {
		return nil
	}
	seen := map[string]struct{}{}
	for _, dir := range filepath.SplitList(path) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			name := ent.Name()
			base := strings.TrimSuffix(name, filepath.Ext(name))
			if _, ok := seen[base]; ok {
				continue
			}
			full := filepath.Join(dir, name)
			if fi, err := os.Stat(full); err == nil && fi.Mode().IsRegular() && isExecutable(full) {
				seen[base] = struct{}{}
			}
		}
	}
	res := make([]string, 0, len(seen))
	for k := range seen {
		res = append(res, k)
	}
	return res
}

func isExecutable(path string) bool {
	_, err := exec.LookPath(path)
	return err == nil
}
