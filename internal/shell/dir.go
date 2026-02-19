package shell

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type directoryEntry struct {
	name    string
	isDir   bool
	modTime string
	size    int64
	icon    string
}

func (a *App) runBuiltin(line string) (bool, int) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return false, 0
	}
	if strings.ToLower(fields[0]) != "dir" {
		return false, 0
	}
	if len(fields) > 2 {
		fmt.Fprintln(os.Stderr, "usage: dir [path]")
		return true, 1
	}
	if len(fields) == 2 && strings.HasPrefix(fields[1], "/") {
		// Let shell-native switches (/w, /p, /s...) continue to work.
		return false, 0
	}

	target := "."
	if len(fields) == 2 {
		target = fields[1]
	}
	if err := renderDirectory(os.Stdout, target); err != nil {
		fmt.Fprintf(os.Stderr, "dir: %v\n", err)
		return true, 1
	}
	return true, 0
}

func renderDirectory(w io.Writer, target string) error {
	absPath, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return err
	}

	rows := make([]directoryEntry, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rows = append(rows, directoryEntry{
			name:    entry.Name(),
			isDir:   entry.IsDir(),
			modTime: info.ModTime().Format("2006-01-02 15:04"),
			size:    info.Size(),
			icon:    fileIcon(entry.Name(), entry.IsDir()),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].isDir != rows[j].isDir {
			return rows[i].isDir
		}
		return strings.ToLower(rows[i].name) < strings.ToLower(rows[j].name)
	})

	fmt.Fprintf(w, "📂 %s\n\n", absPath)
	var dirCount, fileCount int
	var totalBytes int64
	for _, row := range rows {
		displayName := row.name
		sizeText := humanBytes(row.size)
		if row.isDir {
			displayName += string(os.PathSeparator)
			sizeText = "<DIR>"
			dirCount++
		} else {
			fileCount++
			totalBytes += row.size
		}
		fmt.Fprintf(w, "%s  %s  %8s  %s\n", row.icon, row.modTime, sizeText, displayName)
	}
	fmt.Fprintf(w, "\n%d folder(s), %d file(s), %s total\n", dirCount, fileCount, humanBytes(totalBytes))
	return nil
}

func fileIcon(name string, isDir bool) string {
	if isDir {
		return "📁"
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".py":
		return "🐍"
	case ".go":
		return "🐹"
	case ".js", ".ts":
		return "🟨"
	case ".md":
		return "📝"
	case ".toml", ".ini", ".yaml", ".yml":
		return "⚙️"
	case ".json":
		return "🧩"
	case ".exe", ".bat", ".cmd", ".sh":
		return "⚡"
	default:
		return "📄"
	}
}

func humanBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}
