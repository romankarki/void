package ronb

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

var procGetStdHandle = kernel32.NewProc("GetStdHandle")
var procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
var procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
var procReadConsoleInput = kernel32.NewProc("ReadConsoleInputW")
var procFlushConsoleInputBuffer = kernel32.NewProc("FlushConsoleInputBuffer")

const (
	STD_INPUT_HANDLE       = 0xFFFFFFF6
	ENABLE_ECHO_INPUT      = 0x0004
	ENABLE_LINE_INPUT      = 0x0002
	KEY_EVENT              = 0x0001
	ENABLE_EXTENDED_FLAGS  = 0x0080
	ENABLE_QUICK_EDIT_MODE = 0x0040
)

var stdinHandle syscall.Handle
var oldMode uint32

func initConsole() error {
	r, _, _ := procGetStdHandle.Call(uintptr(STD_INPUT_HANDLE))
	stdinHandle = syscall.Handle(r)

	var mode uint32
	procGetConsoleMode.Call(uintptr(stdinHandle), uintptr(unsafe.Pointer(&mode)))
	oldMode = mode

	newMode := uint32(ENABLE_EXTENDED_FLAGS)
	newMode &^= ENABLE_QUICK_EDIT_MODE
	newMode &^= ENABLE_ECHO_INPUT
	newMode &^= ENABLE_LINE_INPUT

	procSetConsoleMode.Call(uintptr(stdinHandle), uintptr(newMode))
	procFlushConsoleInputBuffer.Call(uintptr(stdinHandle))

	return nil
}

func restoreConsole() {
	procSetConsoleMode.Call(uintptr(stdinHandle), uintptr(oldMode))
}

const (
	KEY_UP    = 0x26
	KEY_DOWN  = 0x28
	KEY_ENTER = 0x0D
	KEY_ESC   = 0x1B
	KEY_LEFT  = 0x25
	KEY_RIGHT = 0x27
)

func readKey() (string, bool) {
	var buf [20]byte
	var numRead uint32

	for {
		ret, _, _ := procReadConsoleInput.Call(
			uintptr(stdinHandle),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(1),
			uintptr(unsafe.Pointer(&numRead)),
		)

		if ret == 0 || numRead == 0 {
			continue
		}

		eventType := uint16(buf[0]) | (uint16(buf[1]) << 8)
		if eventType != KEY_EVENT {
			continue
		}

		keyDown := uint32(buf[4]) | (uint32(buf[5]) << 8) | (uint32(buf[6]) << 16) | (uint32(buf[7]) << 24)
		if keyDown == 0 {
			continue
		}

		vk := uint16(buf[10]) | (uint16(buf[11]) << 8)
		ch := uint16(buf[14]) | (uint16(buf[15]) << 8)

		switch vk {
		case KEY_UP:
			return "up", true
		case KEY_DOWN:
			return "down", true
		case KEY_ENTER:
			return "enter", true
		case KEY_ESC:
			return "esc", true
		case KEY_LEFT:
			return "left", true
		case KEY_RIGHT:
			return "right", true
		}

		if ch >= 32 && ch < 127 {
			return string(rune(ch)), true
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
	page := 1
	view := "list"
	var content string
	var err error
	escCount := 0

	for {
		if view == "list" {
			renderList(articles, selected, page)
		} else {
			renderDetail(articles[selected].Title, content, selected, len(articles), page)
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
			case "left", "n":
				clearScreen()
				fmt.Print("\x1b[?25h")
				fmt.Printf("\n Fetching page %d...\n", page+1)
				page++
				newArticles, ferr := FetchNews(page)
				if ferr != nil || len(newArticles) == 0 {
					page--
					fmt.Fprintf(os.Stderr, " No more pages or failed to fetch\n")
					articles, _ = FetchNews(page)
				} else {
					articles = newArticles
					selected = 0
				}
				escCount = 0
			case "right", "p":
				if page > 1 {
					clearScreen()
					fmt.Print("\x1b[?25h")
					page--
					fmt.Printf("\n Fetching page %d...\n", page)
					articles, err = FetchNews(page)
					if err != nil {
						fmt.Fprintf(os.Stderr, " Failed to fetch: %v\n", err)
					}
					selected = 0
				}
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

func renderList(articles []Article, selected int, page int) {
	clearScreen()
	fmt.Print("\x1b[?25l")

	fmt.Printf("\n \x1b[1;36m RONB News - Page %d\x1b[0m\n", page)
	fmt.Print(" \x1b[36m─────────────────────────────────────────────────────────\x1b[0m\n")
	fmt.Print(" \x1b[90m↑/↓ Navigate  ·  Enter View  ·  ←/N Next  ·  →/P Prev  ·  Esc×2 Exit\x1b[0m\n")
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

func renderDetail(title, content string, idx, total int, page int) {
	clearScreen()

	fmt.Printf("\n \x1b[1;36m RONB News - Article (Page %d)\x1b[0m\n", page)
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
