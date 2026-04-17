# goshell

A terminal emulator written in Go that wraps the user's existing shell (bash/zsh) and layers on top an interactive autocomplete popup for common CLI tools.

## Purpose

goshell intercepts keystrokes before they reach the shell, echoes them locally, and renders a suggestion popup in real time — without replacing the shell or modifying its configuration. When the user submits a command, goshell forwards it to the underlying shell via a PTY.

## Architecture

```
stdin (raw mode)
     │
     ▼
Dispatcher (internal/input)
     │  keystroke handling, line editing, suggestion lifecycle
     │
     ├──► LineBuf (internal/buffer)      — UTF-8 aware line buffer with cursor
     ├──► Parser  (internal/buffer)      — tokenise input, derive Context
     ├──► Engine  (internal/suggest)     — match Context against Specs
     ├──► Popup   (internal/render)      — ANSI escape rendering below cursor
     └──► PTY write (internal/pty)       — forward completed commands to shell
                                           and handle terminal resize (SIGWINCH)

PTY output
     │
     ▼
Emulator (internal/terminal)            — vt10x wrapper, tracks alt-screen mode
     │
     ▼
stdout                                  — raw PTY bytes passed through unchanged
```

## Key Components

### `cmd/`
- `main.go` — entry point: spawns the shell in a PTY, wires goroutines for PTY output, output serialisation, and the input dispatcher.
- `init.go` — (future) initialisation helpers.

### `internal/input/`
- `dispatcher.go` — holds all shared state (linebuf, suggestions, history, popup). Exposes `Stop()` to signal the input loop from outside (used when the shell process exits).
- `liner_input.go` — `RunWithLiner()`: the main input loop in raw mode. Reads bytes from a goroutine via a channel so it can `select` on the `done` channel and exit cleanly when the shell exits. Handles Ctrl-C, Ctrl-D, Ctrl-Z, Tab, Enter, arrow keys, bracketed paste, history navigation, and ESC (exits the app).

### `internal/buffer/`
- `linebuf.go` — cursor-aware UTF-8 line buffer. Operations: Append, Delete, DeleteWord, MoveCursor*, TruncateFrom, ReplaceLastWord, DisplayWidth.
- `parser.go` — tokenises the current line and classifies it into a `Context` (ContextCommandPartial, ContextSubcommand, ContextFlag, etc.) to drive suggestion lookup.

### `internal/suggest/`
- `engine.go` — maps a `Context` to a `[]specs.Suggestion` by looking up the registered `Spec` for the current command and filtering by partial prefix.
- `cache.go` — (reserved for caching dynamic generators).

### `internal/render/`
- `popup.go` — renders the suggestion list below the cursor using ANSI save/restore cursor sequences. Shows bullet `●`/`○`, name, and description. Erase and AcceptAndRedraw keep the shell prompt untouched.
- `types.go` — `OutputOp` and `OutputChan` interface used to serialise all stdout writes through a single goroutine.

### `internal/terminal/`
- `emulator.go` — wraps `vt10x` to track terminal state. Used to detect alt-screen mode (vim, less, etc.) so keystrokes are passed through transparently instead of being interpreted by goshell.

### `internal/pty/`
- `pty_unix.go` — `DisableEcho`, `Resizepty` for Unix.
- `pty_windows.go` — stub for Windows.

### `internal/config/`
- `config.go` — TOML config (theme colours, keybindings). Loaded from a file; falls back to `DefaultConfig()`.

### `specs/`
- `spec.go` — core types: `Spec`, `Subcommand`, `Option`, `ArgSpec`, `Suggestion`.
- `registry.go` — global `map[string]*Spec` with `Register` / `Get`.
- `git.go`, `docker.go`, `kubectl.go`, `npm.go`, `go.go` — per-tool specs registered at init time.

## Data Flow: Suggestion Lifecycle

1. User types a character → `handleRuneInteractive` appends to `LineBuf` and calls `refreshSuggestions`.
2. `refreshSuggestions` → `Parser.GetCurrentContext` → `Engine.GetSuggestions` → `Popup.Render`.
3. Arrow Up/Down navigates the popup; Tab or Enter accepts and forwards to `ReplaceLastWord`.
4. On Enter the popup is erased, the command is forwarded to the PTY, and `LineBuf` is reset.

## Keybindings

| Key | Action |
|-----|--------|
| `Tab` | Accept selected suggestion |
| `↑` / `↓` | Navigate suggestions (or history when popup hidden) |
| `←` / `→` | Move cursor in line |
| `Enter` | Submit command |
| `Ctrl-C` | Cancel current input |
| `Ctrl-D` | Send EOF to shell |
| `Ctrl-Z` | Suspend (SIGTSTP) |
| `Ctrl-U` | Kill line from cursor to start |
| `Ctrl-K` | Kill line from cursor to end |
| `Ctrl-W` | Kill word before cursor |
| `ESC` | Close popup (first press) or exit goshell (second press) |

## Adding a New Command Spec

Create `specs/<cmd>.go` and call `specs.Register` in an `init()` function:

```go
package specs

func init() {
    Register("mycmd", &Spec{
        Name: "mycmd",
        Subcommands: []Subcommand{
            {Name: "run", Description: "Run something"},
        },
        Options: []Option{
            {Names: []string{"--verbose", "-v"}, Description: "Verbose output"},
        },
    })
}
```

The engine will pick it up automatically — no other changes needed.

## Build

```bash
go build -o goshell ./cmd/
```

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/creack/pty` | PTY creation and resizing |
| `github.com/hinshun/vt10x` | VT100/VT10x terminal emulation (alt-screen detection) |
| `github.com/peterh/liner` | History persistence |
| `github.com/BurntSushi/toml` | Config file parsing |
| `golang.org/x/term` | Raw mode on stdin |
| `github.com/mattn/go-runewidth` | Display-width calculation for Unicode |
