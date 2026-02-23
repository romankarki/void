package ronb

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var getStdHandle = kernel32.NewProc("GetStdHandle")
var setConsoleMode = kernel32.NewProc("SetConsoleMode")
var getConsoleMode = kernel32.NewProc("GetConsoleMode")
var readConsoleInput = kernel32.NewProc("ReadConsoleInputW")

const (
	STD_INPUT_HANDLE  = ^uint32(0) - 10
	ENABLE_ECHO_INPUT = uint32(0x0004)
	ENABLE_LINE_INPUT = uint32(0x0002)
	ENABLE_PROCESSED  = uint32(0x0001)
)

type inputRecord struct {
	eventType uint16
	padding   [2]byte
	keyEvent  keyEventRecord
}

type keyEventRecord struct {
	bKeyDown          int32
	wRepeatCount      uint16
	wVirtualKeyCode   uint16
	wVirtualScanCode  uint16
	unicodeChar       uint16
	dwControlKeyState uint32
}

var stdinHandle syscall.Handle
var oldMode uint32

func initConsole() error {
	h, _, _ := getStdHandle.Call(uintptr(STD_INPUT_HANDLE))
	stdinHandle = syscall.Handle(h)

	var mode uint32
	_, _, _ = getConsoleMode.Call(uintptr(stdinHandle), uintptr(unsafe.Pointer(&mode)))
	oldMode = mode

	newMode := mode & ^(ENABLE_ECHO_INPUT | ENABLE_LINE_INPUT)
	_, _, _ = setConsoleMode.Call(uintptr(stdinHandle), uintptr(newMode))

	return nil
}

func restoreConsole() {
	_, _, _ = setConsoleMode.Call(uintptr(stdinHandle), uintptr(oldMode))
}

func readKey() (string, bool) {
	var ir inputRecord
	var numRead uint32

	for {
		_, _, _ = readConsoleInput.Call(
			uintptr(stdinHandle),
			uintptr(unsafe.Pointer(&ir)),
			uintptr(1),
			uintptr(unsafe.Pointer(&numRead)),
		)

		if ir.eventType == 1 && ir.keyEvent.bKeyDown != 0 {
			vk := ir.keyEvent.wVirtualKeyCode
			ch := ir.keyEvent.unicodeChar

			switch vk {
			case 0x26:
				return "up", true
			case 0x28:
				return "down", true
			case 0x0D:
				return "enter", true
			case 0x1B:
				return "esc", true
			}

			if ch >= 32 && ch < 127 {
				return string(rune(ch)), true
			}
		}
	}
}

func RunTUI(articles []Article) int {
	if len(articles) == 0 {
		fmt.Println("\n No articles found")
		return 1
	}

	initConsole()
	defer restoreConsole()

	selected := 0
	view := "list"
	var content string
	var err error
	escCount := 0

	for {
		if view == "list" {
			renderList(articles, selected)
		} else {
			renderDetail(articles[selected].Title, content, selected, len(articles))
		}

		key, ok := readKey()
		if !ok {
			continue
		}

		if view == "list" {
			switch key {
			case "up":
				if selected > 0 {
					selected--
				}
				escCount = 0
			case "down":
				if selected < len(articles)-1 {
					selected++
				}
				escCount = 0
			case "enter":
				fmt.Print("\x1b[?25l")
				fmt.Print("\x1b[2J\x1b[H")
				fmt.Println(" Loading article...")
				content, err = FetchArticleContent(articles[selected].URL)
				if err != nil {
					content = "Failed to load: " + err.Error()
				}
				view = "detail"
				escCount = 0
			case "esc":
				escCount++
				if escCount >= 2 {
					clearScreen()
					fmt.Print("\x1b[?25h")
					return 0
				}
			default:
				escCount = 0
			}
		} else {
			switch key {
			case "esc":
				escCount++
				if escCount >= 2 {
					clearScreen()
					fmt.Print("\x1b[?25h")
					return 0
				}
				view = "list"
			case "up", "down":
				escCount = 0
			case "enter":
				view = "list"
				escCount = 0
			default:
				escCount = 0
			}
		}
	}
}

func renderList(articles []Article, selected int) {
	clearScreen()
	fmt.Print("\x1b[?25l")

	fmt.Print("\n \x1b[1;36m RONB News - Latest Updates\x1b[0m\n")
	fmt.Print(" \x1b[36m─────────────────────────────────────────────────────────\x1b[0m\n")
	fmt.Print(" \x1b[90m↑/↓ Navigate  ·  Enter View  ·  Esc×2 Exit\x1b[0m\n")
	fmt.Print(" \x1b[36m─────────────────────────────────────────────────────────\x1b[0m\n\n")

	maxShow := 15
	start := 0
	if selected >= maxShow {
		start = selected - maxShow + 1
	}
	end := start + maxShow
	if end > len(articles) {
		end = len(articles)
	}

	for i := start; i < end; i++ {
		prefix := "   "
		if i == selected {
			prefix = " \x1b[1;32m▶\x1b[0m "
			fmt.Printf("%s\x1b[1;32m%s\x1b[0m\n", prefix, articles[i].Title)
		} else {
			fmt.Printf("%s\x1b[90m%s\x1b[0m\n", prefix, articles[i].Title)
		}
	}

	if len(articles) > maxShow {
		fmt.Printf("\n \x1b[90mShowing %d-%d of %d articles\x1b[0m\n", start+1, end, len(articles))
	}
}

func renderDetail(title, content string, idx, total int) {
	clearScreen()

	fmt.Print("\n \x1b[1;36m RONB News - Article\x1b[0m\n")
	fmt.Print(" \x1b[36m─────────────────────────────────────────────────────────\x1b[0m\n")
	fmt.Printf(" \x1b[1;33m%s\x1b[0m\n", title)
	fmt.Print(" \x1b[36m─────────────────────────────────────────────────────────\x1b[0m\n\n")

	words := strings.Fields(content)
	lines := []string{}
	currentLine := ""
	for _, word := range words {
		if len(currentLine)+len(word)+1 > 70 {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			if currentLine != "" {
				currentLine += " "
			}
			currentLine += word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	for _, line := range lines {
		fmt.Printf(" %s\n", line)
	}

	fmt.Printf("\n\n \x1b[90mArticle %d of %d  ·  Enter/Back  ·  Esc×2 Exit\x1b[0m\n", idx+1, total)
}

func clearScreen() {
	fmt.Print("\x1b[2J\x1b[H")
}
