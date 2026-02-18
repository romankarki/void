# Void

Void is a configurable shell wrapper inspired by the requirements in `REQUIREMENTS.md`. It adds a customizable prompt, alias expansion, persistent history, theme presets, and basic command completion helpers while delegating command execution to your existing shell.

## Current MVP Features

- Config-driven shell executable and prompt (`config.toml`).
- Prompt segments: `user`, `path`, `time`, `exit_code`.
- Presets: `minimal`, `cyberpunk`.
- Alias expansion before command execution.
- Persistent history with dedup + max size cap.
- Meta commands:
  - `void history`
  - `void complete <prefix>`
  - `void reload`

## Project Layout

```text
cmd/void/main.go             # app entrypoint
internal/config/             # config model + loader
internal/shell/              # interactive loop and command dispatch
internal/prompt/             # prompt segment renderer
internal/history/            # history persistence
internal/autocomplete/       # completion suggestions
internal/theme/              # preset application
presets/                     # built-in preset files
```

## Setup Guide

### 1) Prerequisites

- Go 1.22+
- A shell executable in your PATH (`cmd.exe`, `powershell.exe`, `sh`, etc.)

### 2) Clone and build

```bash
git clone <your-fork-or-repo-url>
cd void
go build -o void ./cmd/void
```

### 3) Create config

Copy the example:

```bash
mkdir -p ~/.void
cp config.example.toml ~/.void/config.toml
```

Or pass a config explicitly:

```bash
./void --config ./config.example.toml
```

### 4) Run

```bash
./void
```

Type commands as usual. Use `exit` to leave Void.

### 5) Configure for Windows CMD behavior

In `config.toml`:

```toml
[shell]
executable = "cmd.exe"
args = ["/C"]
```

### 6) Preset themes

Use a preset in `config.toml`:

```toml
preset = "cyberpunk"
```

Available now: `cyberpunk`, `minimal`.

## Development

Run checks:

```bash
go test ./...
go vet ./...
```

## Notes

This implementation is intentionally MVP-focused and currently uses a lightweight TOML parser that supports the subset used by `config.example.toml` and presets.
