package beautify

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func Run(command string, args []string) int {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		fullCmd := command
		if len(args) > 0 {
			fullCmd = command + " " + strings.Join(args, " ")
		}
		cmd = exec.Command("cmd", "/C", fullCmd)
	} else {
		cmd = exec.Command(command, args...)
	}

	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	output := stdout.String()
	errOutput := stderr.String()
	exitCode := 0

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
			errOutput = fmt.Sprintf("error: %v", err)
		}
	}

	printBeautified(command, args, output, errOutput, exitCode, duration)
	return exitCode
}

func printBeautified(cmd string, args []string, output, errOutput string, exitCode int, duration time.Duration) {
	fullCmd := cmd
	if len(args) > 0 {
		fullCmd = cmd + " " + strings.Join(args, " ")
	}

	fmt.Println()
	printSeparator()

	fmt.Print(dim("  [ "))
	fmt.Print(green(fullCmd))
	fmt.Println(dim(" ]"))

	printSeparator()

	if output != "" {
		printLines(output, false)
	}

	if errOutput != "" {
		printLines(errOutput, true)
	}

	printSeparator()

	durationStr := formatDuration(duration)

	if exitCode == 0 {
		fmt.Print("  ")
		fmt.Print(green("[ OK ]"))
		fmt.Print(dim(" ~ "))
		fmt.Println(cyan(durationStr))
	} else {
		fmt.Print("  ")
		fmt.Print(red(fmt.Sprintf("[ FAILED %d ]", exitCode)))
		fmt.Print(dim(" ~ "))
		fmt.Println(cyan(durationStr))
	}

	printSeparator()
	fmt.Println()
}

func printLines(content string, isError bool) {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fmt.Print("  ")
		if isError {
			fmt.Println(red(line))
		} else {
			fmt.Println(white(line))
		}
	}
}

func printSeparator() {
	fmt.Println(dim("  " + strings.Repeat("-", 50)))
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func green(s string) string { return "\x1b[32m" + s + "\x1b[0m" }
func red(s string) string   { return "\x1b[31m" + s + "\x1b[0m" }
func cyan(s string) string  { return "\x1b[36m" + s + "\x1b[0m" }
func white(s string) string { return "\x1b[37m" + s + "\x1b[0m" }
func dim(s string) string   { return "\x1b[90m" + s + "\x1b[0m" }
