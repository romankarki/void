package installer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type UpdateOptions struct {
	Repo string
}

func Update(opts UpdateOptions, stdout io.Writer) error {
	repo := strings.TrimSpace(opts.Repo)
	if repo == "" {
		repo = defaultRepo
	}

	targetPath, err := installBinaryPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(targetPath); err != nil {
		return fmt.Errorf("Void is not installed at %s; run `void install` first", targetPath)
	}

	asset, err := releaseAssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://github.com/%s/releases/latest/download/%s", repo, asset)

	tmpDir, err := os.MkdirTemp("", "void-update-*")
	if err != nil {
		return fmt.Errorf("create temp directory: %w", err)
	}
	tmpFile := filepath.Join(tmpDir, asset)

	fmt.Fprintf(stdout, "Downloading %s\n", url)
	if err := downloadFile(url, tmpFile); err != nil {
		return err
	}

	exePath, _ := os.Executable()
	if samePath(exePath, targetPath) && runtime.GOOS == "windows" {
		scriptPath := filepath.Join(tmpDir, "apply-update.cmd")
		if err := os.WriteFile(scriptPath, []byte(updateScript(tmpFile, targetPath)), 0o644); err != nil {
			return fmt.Errorf("prepare update script: %w", err)
		}
		cmd := exec.Command("cmd", "/C", "start", "", "/B", scriptPath)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("launch update script: %w", err)
		}
		fmt.Fprintln(stdout, "Update staged. Close this terminal and start Void again.")
		return nil
	}

	if err := copyFile(tmpFile, targetPath, 0o755); err != nil {
		return fmt.Errorf("replace executable: %w", err)
	}
	_ = os.RemoveAll(tmpDir)
	fmt.Fprintln(stdout, "Void updated successfully.")
	return nil
}

func releaseAssetName(goos, goarch string) (string, error) {
	switch goos {
	case "windows":
		switch goarch {
		case "amd64":
			return "void-windows-amd64.exe", nil
		case "arm64":
			return "void-windows-arm64.exe", nil
		default:
			return "", fmt.Errorf("unsupported Windows architecture %q", goarch)
		}
	case "linux":
		switch goarch {
		case "amd64":
			return "void-linux-amd64", nil
		case "arm64":
			return "void-linux-arm64", nil
		default:
			return "", fmt.Errorf("unsupported Linux architecture %q", goarch)
		}
	case "darwin":
		switch goarch {
		case "amd64":
			return "void-darwin-amd64", nil
		case "arm64":
			return "void-darwin-arm64", nil
		default:
			return "", fmt.Errorf("unsupported macOS architecture %q", goarch)
		}
	default:
		return "", fmt.Errorf("unsupported platform %s/%s", goos, goarch)
	}
}

func downloadFile(url, path string) error {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download update: unexpected status %s", resp.Status)
	}

	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return fmt.Errorf("open temp file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("save update: %w", err)
	}
	return nil
}

func updateScript(tmpFile, targetFile string) string {
	return "@echo off\r\n" +
		"setlocal\r\n" +
		"ping 127.0.0.1 -n 2 >nul\r\n" +
		"copy /Y \"" + tmpFile + "\" \"" + targetFile + "\" >nul\r\n" +
		"del /Q \"" + tmpFile + "\" >nul 2>nul\r\n" +
		"del \"%~f0\" >nul 2>nul\r\n" +
		"endlocal\r\n"
}
