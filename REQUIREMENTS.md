# void — Requirements & Build Plan
> A futuristic, extensible terminal beautifier for Windows CMD and VS Code, built in Go.

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Architecture Overview](#architecture-overview)
3. [Feature Specification](#feature-specification)
4. [Configuration System](#configuration-system)
5. [Preset Themes](#preset-themes)
6. [Autocomplete Engine](#autocomplete-engine)
7. [VS Code Integration](#vs-code-integration)
8. [Build Plan — Step by Step](#build-plan--step-by-step)
9. [Project Structure](#project-structure)
10. [Packaging & Distribution](#packaging--distribution)

---

## Project Overview

**Void** is a terminal shell wrapper written in Go that sits on top of Windows CMD (and optionally PowerShell). It intercepts input/output and renders a fully customizable, visually rich prompt experience — colors, glyphs, path segments, autocomplete, and preset themes — without replacing the underlying shell.

### Goals

- Drop-in replacement shell wrapper (no system modification required)
- Fully driven by a single TOML config file
- Preset theme system (choose a look in one line)
- Smart autocomplete for common CLIs and built-in commands
- Render properly inside VS Code's integrated terminal
- Distributable via a Windows `.exe` installer (NSIS or WiX)
- Open-source ready from day one

---

## Architecture Overview

```
User Input
    │
    ▼
┌─────────────────────────────────┐
│        Void Shell          │
│  ┌──────────┐  ┌─────────────┐  │
│  │  Input   │  │   Prompt    │  │
│  │  Handler │  │   Renderer  │  │
│  └────┬─────┘  └──────┬──────┘  │
│       │               │         │
│  ┌────▼──────────────▼──────┐   │
│  │    Autocomplete Engine   │   │
│  └────────────┬─────────────┘   │
│               │                 │
│  ┌────────────▼─────────────┐   │
│  │     Config Engine        │   │
│  │  (TOML + Theme Presets)  │   │
│  └──────────────────────────┘   │
└─────────────────────────────────┘
    │
    ▼
Windows CMD / PowerShell (subprocess)
```

**Key Design Decisions:**
- Go's `os/exec` spawns CMD as a subprocess; Void proxies stdin/stdout
- ANSI escape codes handle all color/cursor rendering (Windows 10+ supports this natively via `ENABLE_VIRTUAL_TERMINAL_PROCESSING`)
- Config loaded once at startup; hot-reload triggered by `SIGHUP` or a dedicated command
- All rendering is segment-based (Powerline-style), each segment is a pluggable struct

---

## Feature Specification

### 1. Prompt Rendering

The prompt is composed of **segments**. Each segment has:
- Content (text or a dynamic value)
- Foreground color
- Background color
- Separator glyph (e.g., ``, ``, `|`, `>`)
- Optional icon (Nerd Font glyph)

**Default segments available:**

| Segment ID | Description | Example Output |
|---|---|---|
| `os` | OS icon | ` ` |
| `drive` | Drive letter | ` C:` |
| `path` | Full path breadcrumbs | ` Users >  Projects` |
| `git` | Git branch + status | ` main ✓` |
| `node` | Node.js version | ` v20.11` |
| `python` | Python venv indicator | ` venv` |
| `time` | Current time | `12:34` |
| `exit_code` | Last command exit code | `✗ 1` |
| `duration` | Last command duration | `⏱ 2.3s` |
| `user` | Current username | ` john` |

Each segment is opt-in and reorderable in config.

### 2. Colors

- Full 24-bit RGB color support for Windows 10+ (`#RRGGBB` in config)
- Fallback 256-color palette for older terminals
- Per-segment foreground + background colors
- Global color palette defined once and referenced by name

### 3. Autocomplete

- Triggered on `Tab` keypress
- Menu-style popup rendered inline (not a separate window)
- Completion sources: built-in commands, CLI tools, file paths, history
- Full list of supported CLI tools (see Autocomplete Engine section)

### 4. History

- Persistent command history across sessions (stored in `~/.void/history`)
- History search with `Ctrl+R` (fuzzy)
- Dedup and configurable max size

### 5. Keybindings

- Configurable keybindings in config
- Defaults: standard readline-like (Ctrl+A, Ctrl+E, Ctrl+R, Ctrl+L, etc.)

### 6. Aliases

- Define aliases in config; Void expands them before passing to CMD

---

## Configuration System

### File Location

Void resolves config in this order (first found wins):
1. `--config <path>` flag at launch
2. `$TERMFORGE_CONFIG` environment variable
3. `%USERPROFILE%\.void\config.toml`
4. `%APPDATA%\Void\config.toml`

### Full Annotated `config.toml`

```toml
# ─────────────────────────────────────────────────
# Void Configuration File
# ─────────────────────────────────────────────────

# Preset theme to use. Overrides all visual settings below.
# Set to "" to use manual config instead.
# Available: "cyberpunk", "nord", "dracula", "gruvbox", "minimal", "ocean"
preset = ""

# ─── Shell ─────────────────────────────────────────
[shell]
executable = "cmd.exe"      # or "powershell.exe"
args = ["/K"]               # args passed to the shell
init_script = ""            # optional: path to a cmd script to source on start

# ─── Prompt ────────────────────────────────────────
[prompt]
newline_before = true       # print blank line before prompt
newline_after = false       # print newline after prompt (multiline prompt)
style = "powerline"         # "powerline" | "arrow" | "plain" | "minimal"

# Separator glyphs between segments
[prompt.separators]
left = ""                  # Powerline right-arrow glyph (requires Nerd Font)
left_thin = ""             # thin separator
right = ""                 # right-side prompt separator
# For plain ASCII fallback use: left = ">", left_thin = "|"

# Define which segments appear and in what order
[[prompt.segments]]
id = "os"
enabled = true
fg = "#ffffff"
bg = "#1e1e2e"
icon = " "

[[prompt.segments]]
id = "drive"
enabled = true
fg = "#89b4fa"
bg = "#313244"

[[prompt.segments]]
id = "path"
enabled = true
fg = "#fab387"
bg = "#45475a"
max_depth = 4               # only show last N path components
shorten_home = true         # replace home dir with ~

[[prompt.segments]]
id = "git"
enabled = true
fg = "#a6e3a1"
bg = "#313244"
show_branch = true
show_status = true          # show dirty/clean/ahead/behind symbols
dirty_symbol = "✗"
clean_symbol = "✓"
ahead_symbol = "↑"
behind_symbol = "↓"

[[prompt.segments]]
id = "exit_code"
enabled = true
fg_success = "#a6e3a1"
fg_fail = "#f38ba8"
bg = "#1e1e2e"
show_success = false        # hide segment when exit code is 0

[[prompt.segments]]
id = "time"
enabled = false
fg = "#6c7086"
bg = "#1e1e2e"
format = "15:04"            # Go time format string

[[prompt.segments]]
id = "duration"
enabled = true
fg = "#f9e2af"
bg = "#1e1e2e"
min_duration_ms = 2000      # only show if command took longer than this

# ─── Colors (Named Palette) ────────────────────────
# Reference these by name in segment fg/bg fields
[colors]
primary   = "#89b4fa"
secondary = "#cba6f7"
success   = "#a6e3a1"
warning   = "#f9e2af"
danger    = "#f38ba8"
muted     = "#6c7086"
surface0  = "#313244"
surface1  = "#45475a"
base      = "#1e1e2e"

# ─── Autocomplete ──────────────────────────────────
[autocomplete]
enabled = true
max_suggestions = 8
fuzzy = true                        # fuzzy match (not just prefix)
history_weight = 0.4                # how much to boost history matches (0.0–1.0)
show_descriptions = true            # show description next to each suggestion

# Which completion sources are active
[autocomplete.sources]
history    = true
filesystem = true
commands   = true
git        = true
npm        = true
pnpm       = true
bun        = true
pip        = true
cargo      = true
go_cli     = true
docker     = true

# ─── History ──────────────────────────────────────
[history]
enabled = true
max_size = 5000
dedup = true
path = ""                           # leave blank for default (~/.void/history)

# ─── Keybindings ─────────────────────────────────
[keybindings]
accept          = "Enter"
complete        = "Tab"
history_prev    = "Up"
history_next    = "Down"
history_search  = "Ctrl+R"
clear_screen    = "Ctrl+L"
cancel          = "Ctrl+C"
move_bol        = "Ctrl+A"
move_eol        = "Ctrl+E"
delete_word     = "Ctrl+W"
kill_line       = "Ctrl+K"

# ─── Aliases ─────────────────────────────────────
[aliases]
ll  = "dir /a"
la  = "dir /a /b"
cls = "clear"
gs  = "git status"
gp  = "git push"
gl  = "git pull"

# ─── Display ─────────────────────────────────────
[display]
nerd_fonts = true               # set false to disable Nerd Font glyphs
unicode = true                  # set false to fall back to ASCII-only
tab_size = 4

# ─── Updates ─────────────────────────────────────
[updates]
check_on_start = true
channel = "stable"              # "stable" | "nightly"
```

---

## Preset Themes

Presets are bundled TOML files embedded in the binary (`go:embed`). They override the visual settings but not behavior settings.

### Available Presets

| Preset | Vibe | Colors |
|---|---|---|
| `cyberpunk` | Neon, futuristic, high contrast | Magenta, cyan, electric yellow on dark |
| `nord` | Cool, minimal, Scandinavian | Polar blues and snow whites |
| `dracula` | Classic dark, purple-forward | Purple, pink, green on near-black |
| `gruvbox` | Warm retro | Amber, orange, green on brown-grey |
| `minimal` | Clean, monochrome | White/grey only, no background colors |
| `ocean` | Deep sea blues | Blues and teals on dark navy |
| `matrix` | Green on black | Green shades only |
| `catppuccin` | Pastel mocha | Soft pastels on dark base |

### Using a Preset

```toml
preset = "cyberpunk"
```

### Overriding Part of a Preset

Presets are applied first, then your manual config layered on top. So you can use a preset and then tweak specific segments:

```toml
preset = "nord"

# Override just the path segment color
[[prompt.segments]]
id = "path"
fg = "#ff0000"   # this overrides nord's path color
```

---

## Autocomplete Engine

### Architecture

The autocomplete engine maintains a **completion tree** per CLI tool. At launch, it loads static completion definitions from embedded YAML files. At runtime, it merges:
1. Static completions (subcommands, flags, descriptions)
2. Dynamic completions (file paths, git branches, npm scripts, etc.)
3. History-based suggestions (ranked by recency + frequency)

### Supported Tools

**Package managers / runtimes:**

| Tool | Completions |
|---|---|
| `npm` | install, run, scripts from package.json, publish, audit |
| `pnpm` | Same as npm + workspace commands |
| `bun` | Same as npm + bun-specific (bun add, bun x) |
| `pip` | install, uninstall, freeze, list, show |
| `cargo` | build, run, test, add, publish, fmt, clippy |
| `go` | build, run, test, get, mod, generate, fmt, vet |

**Version control:**

| Tool | Completions |
|---|---|
| `git` | All common subcommands, dynamic branch/tag/remote names, stash list |

**Navigation & filesystem:**

| Tool | Completions |
|---|---|
| `cd` | Directory names relative to current path |
| `ls` / `dir` | Flags |
| `mkdir`, `rm`, `cp`, `mv` | Paths |

**Dev tools:**

| Tool | Completions |
|---|---|
| `docker` | build, run, ps, exec, images, compose |
| `npx` / `bunx` | Common CLI packages |
| `node` | .js files |
| `python` / `python3` | .py files |

### Adding Custom Completions

Users can add completions in `~/.void/completions/` as YAML:

```yaml
# ~/.void/completions/mytool.yaml
tool: mytool
subcommands:
  - name: deploy
    description: Deploy the app
    flags:
      - name: --env
        description: Target environment
        values: [staging, production]
  - name: rollback
    description: Rollback last deploy
```

---

## VS Code Integration

Void works inside VS Code's integrated terminal with **zero extra setup** if:
1. The default terminal profile points to `void.exe`
2. The VS Code terminal font is set to a Nerd Font

### Setting Default Terminal in VS Code

In `settings.json`:
```json
{
  "terminal.integrated.defaultProfile.windows": "Void",
  "terminal.integrated.profiles.windows": {
    "Void": {
      "path": "C:\\Program Files\\Void\\void.exe",
      "icon": "terminal"
    }
  },
  "terminal.integrated.fontFamily": "JetBrainsMono Nerd Font"
}
```

Void auto-detects the `TERM_PROGRAM=vscode` environment variable and applies VS Code-compatible rendering optimizations (disables some ANSI sequences that VS Code doesn't handle).

---

## Build Plan — Step by Step

### Phase 0 — Project Bootstrap (Week 1)

**Step 1: Repository Setup**
- Init Go module: `go mod init github.com/yourname/void`
- Set up folder structure (see Project Structure section)
- Add `Makefile` with targets: `build`, `test`, `lint`, `install`, `release`
- Set up `golangci-lint` config
- Init `git` with `.gitignore` for Go

**Step 2: Windows ANSI Support**
- Enable `ENABLE_VIRTUAL_TERMINAL_PROCESSING` on Windows console handle
- Write a thin `ansi` package: color codes, cursor movement, clear line
- Test colors and cursor control in a plain CMD window

**Step 3: Config Loading**
- Add `github.com/BurntSushi/toml` dependency
- Create `Config` structs matching the full config schema above
- Implement config file discovery (flag → env → user home → appdata)
- Write config validation with helpful error messages

---

### Phase 1 — Core Shell Loop (Week 2)

**Step 4: Input Reading**
- Implement raw terminal input reading on Windows (`ReadConsoleInput`)
- Handle: printable chars, backspace, arrow keys, Ctrl sequences, Enter
- Build a `LineEditor` struct that manages the current input buffer

**Step 5: Shell Subprocess**
- Spawn `cmd.exe` as a long-running subprocess
- Pipe stdin/stdout/stderr
- Forward Void input to CMD, intercept CMD output for rendering

**Step 6: Basic Prompt Rendering**
- Implement the segment pipeline: load segments → render each → join with separators
- Render prompt to stdout before each input read
- Implement: `os`, `drive`, `path`, `exit_code` segments first

---

### Phase 2 — Full Prompt Engine (Week 3)

**Step 7: Remaining Segments**
- Implement: `git`, `time`, `duration`, `user`, `node`, `python`
- For `git`: shell out to `git rev-parse`, `git status --porcelain`
- For `duration`: record start time before forwarding command, compute diff on return

**Step 8: Color System**
- Parse `#RRGGBB` → ANSI 24-bit escape sequences
- Named palette resolution (config `[colors]` table)
- 256-color fallback for older terminals

**Step 9: Separator & Style Engine**
- Implement `powerline`, `arrow`, `plain`, and `minimal` prompt styles
- Handle Nerd Font glyphs with ASCII fallback (`display.nerd_fonts = false`)

---

### Phase 3 — Autocomplete (Week 4–5)

**Step 10: Tab Completion Infrastructure**
- On Tab press in LineEditor, extract the current word/token
- Call the autocomplete engine; render a popup menu inline using ANSI
- Arrow keys navigate the menu; Enter accepts; Escape dismisses

**Step 11: Static Completion Definitions**
- Create YAML completion files for: git, npm, pnpm, bun, pip, cargo, go, docker, cd
- Embed all files into the binary with `//go:embed completions/*.yaml`
- Build a trie/map structure for fast lookup

**Step 12: Dynamic Completions**
- File/directory completion (relative to CWD)
- Git dynamic: branch names, remotes, stash entries
- npm/pnpm: read `package.json` scripts from CWD
- History-based completions: rank by recency + frequency

**Step 13: Fuzzy Matching**
- Implement simple fuzzy match scoring (subsequence match with bonus for prefix/consecutive)
- Sort and display ranked suggestions

---

### Phase 4 — History & Keybindings (Week 6)

**Step 14: History Engine**
- Write/read history file (`~/.void/history`) in JSON Lines format
- Implement dedup, max-size trimming
- Load history on startup; append on command execution

**Step 15: History Search**
- `Ctrl+R` triggers incremental search mode
- Renders search query + matching history entry inline
- Enter accepts; Ctrl+C cancels

**Step 16: Full Keybinding System**
- Map config keybinding strings to actions
- Implement all default actions: BOL, EOL, kill word, kill line, etc.

---

### Phase 5 — Themes & Presets (Week 7)

**Step 17: Preset System**
- Write all 8 preset TOML files in `internal/presets/`
- Embed with `//go:embed presets/*.toml`
- Implement preset loading: parse preset → merge with user config (user config wins)

**Step 18: `void theme` Command**
- Built-in command: `void theme list` — prints all presets with a color preview
- `void theme set <name>` — writes `preset = "<name>"` to config file
- `void theme preview <name>` — renders a sample prompt in that theme

---

### Phase 6 — VS Code & Polish (Week 8)

**Step 19: VS Code Detection & Optimization**
- Detect `TERM_PROGRAM=vscode` at startup
- Adjust rendering: disable some sequences, ensure cursor positioning is correct

**Step 20: `void` Meta Commands**
- `void config edit` — opens config in `$EDITOR`
- `void config validate` — validates config and prints issues
- `void reload` — hot-reloads config without restarting
- `void doctor` — checks font, Windows version, terminal compatibility

**Step 21: Error Handling & Logging**
- Structured log file at `~/.void/void.log`
- `--debug` flag for verbose output
- Graceful error display (never crash the shell, fall back to plain prompt)

---

### Phase 7 — Packaging (Week 9)

**Step 22: Cross-Compilation & Release Build**
- `GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o void.exe`
- Also build ARM64 (`GOARCH=arm64`) for Arm-based Windows
- Strip debug symbols, embed version string via `-ldflags`

**Step 23: Windows Installer**
- Use **NSIS** (Nullsoft Scriptable Install System) or **WiX Toolset** for the installer
- Installer actions:
  - Copy `void.exe` to `C:\Program Files\Void\`
  - Add to system `PATH`
  - Create default config at `%USERPROFILE%\.void\config.toml`
  - Optional: register as VS Code terminal profile
  - Create Start Menu shortcut
  - Create uninstaller

**Step 24: CI/CD Pipeline (GitHub Actions)**
- On push to `main`: run `go test ./...` + `golangci-lint`
- On tag `v*.*.*`: build Windows binaries (amd64 + arm64), create NSIS installer, publish GitHub Release with artifacts

---

## Project Structure

```
void/
├── cmd/
│   └── void/
│       └── main.go                  # Entry point
│
├── internal/
│   ├── config/
│   │   ├── config.go                # Config structs + loader
│   │   └── validate.go              # Validation logic
│   │
│   ├── shell/
│   │   ├── shell.go                 # CMD subprocess manager
│   │   └── loop.go                  # Main REPL loop
│   │
│   ├── input/
│   │   ├── reader.go                # Raw Windows console input
│   │   └── editor.go                # Line editor / buffer
│   │
│   ├── prompt/
│   │   ├── renderer.go              # Segment pipeline + render
│   │   ├── segments.go              # All segment implementations
│   │   └── style.go                 # Style engines (powerline, arrow, etc.)
│   │
│   ├── ansi/
│   │   ├── ansi.go                  # ANSI escape code helpers
│   │   └── windows.go               # Windows VT enable
│   │
│   ├── autocomplete/
│   │   ├── engine.go                # Completion orchestrator
│   │   ├── history.go               # History-based completions
│   │   ├── filesystem.go            # File/dir completions
│   │   ├── dynamic.go               # Dynamic (git, npm scripts)
│   │   ├── loader.go                # Load YAML completion defs
│   │   └── menu.go                  # Inline popup menu renderer
│   │
│   ├── history/
│   │   └── history.go               # Persistent history read/write
│   │
│   └── theme/
│       └── theme.go                 # Preset loader + merger
│
├── completions/                     # Embedded YAML completion defs
│   ├── git.yaml
│   ├── npm.yaml
│   ├── pnpm.yaml
│   ├── bun.yaml
│   ├── pip.yaml
│   ├── cargo.yaml
│   ├── go.yaml
│   └── docker.yaml
│
├── presets/                         # Embedded TOML theme presets
│   ├── cyberpunk.toml
│   ├── nord.toml
│   ├── dracula.toml
│   ├── gruvbox.toml
│   ├── minimal.toml
│   ├── ocean.toml
│   ├── matrix.toml
│   └── catppuccin.toml
│
├── installer/
│   └── void.nsi                # NSIS installer script
│
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                   # Test + lint on push
│   │   └── release.yml              # Build + publish on tag
│   └── ISSUE_TEMPLATE/
│
├── docs/
│   ├── config.md
│   ├── themes.md
│   ├── custom-completions.md
│   └── vscode.md
│
├── Makefile
├── go.mod
├── go.sum
├── config.example.toml              # Full annotated example config
└── README.md
```

---

## Packaging & Distribution

### Installer (NSIS)

```nsis
; void.nsi (abbreviated)
Name "Void"
OutFile "Void-Setup-1.0.0.exe"
InstallDir "$PROGRAMFILES64\Void"

Section "Install"
  SetOutPath "$INSTDIR"
  File "void.exe"
  WriteUninstaller "$INSTDIR\uninstall.exe"
  ; Add to PATH
  EnVar::AddValue "PATH" "$INSTDIR"
  ; Create default config
  CreateDirectory "$PROFILE\.void"
  File /oname="$PROFILE\.void\config.toml" "config.example.toml"
SectionEnd
```

### Build & Release Artifacts

Each MVP release produces:
- `Void-Setup-x.x.x.exe` — Windows installer (amd64)
- `void-windows-amd64.exe` — Standalone binary for manual install
- `void-windows-arm64.exe` — Standalone binary (ARM Windows)

### CI/CD (GitHub Actions)

- On push to `main`: run `go test ./...` + `golangci-lint`
- On tag `v*.*.*`: build both binaries + NSIS installer, attach to GitHub Release

---

## Go Dependencies

| Package | Purpose |
|---|---|
| `github.com/BurntSushi/toml` | Config file parsing |
| `golang.org/x/sys/windows` | Windows console API access |
| `github.com/lithammer/fuzzysearch` | Fuzzy autocomplete matching |
| `gopkg.in/yaml.v3` | Completion definition YAML loading |

All other functionality uses the Go standard library.

---

*Document version: 1.0 — Created for Void project planning.*
