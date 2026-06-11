# goshell

A terminal emulator written in Go that wraps the user's existing shell (bash/zsh) and layers on top an interactive autocomplete popup for common CLI tools.

## Purpose

goshell intercepts keystrokes before they reach the shell, echoes them locally, and renders a suggestion popup in real time — without replacing the shell or modifying its configuration. When the user submits a command, goshell forwards it to the underlying shell via a PTY.

## Six Core Architecture

goshell is organized into 6 cores, each with distinct responsibilities:

```
┌─────────────────────────────────────────────────────────────┐
│  CORE 1: INPUT (internal/input)                             │
│  Keystroke handling, input lifecycle, history navigation    │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  CORE 2: BUFFER + PARSER (internal/buffer)                  │
│  UTF-8 line buffer with cursor, context classification      │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  CORE 3: SUGGESTION ENGINE (internal/suggest)              │
│  Context → Suggestions mapping, Spec registry                │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  CORE 4: RENDER/UI (internal/render)                        │
│  ANSI popup rendering, output serialization                  │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  CORE 5: PTY (internal/pty)                                 │
│  Shell process communication, PTY creation and resize      │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  CORE 6: TERMINAL EMULATOR (internal/terminal)             │
│  VT10x wrapper, alt-screen mode detection                   │
└─────────────────────────────────────────────────────────────┘
```

## Data Flow

```
stdin (raw mode)
     │
     ▼
Dispatcher (CORE 1: Input)
     │  keystroke handling, line editing, suggestion lifecycle
     │
     ├──► LineBuf (CORE 2: Buffer)      — UTF-8 aware line buffer with cursor
     ├──► Parser  (CORE 2: Buffer)     — tokenise input, derive Context
     ├──► Engine  (CORE 3: Suggest)    — match Context against Specs
     ├──► Popup   (CORE 4: Render)     — ANSI escape rendering below cursor
     └──► PTY write (CORE 5: PTY)     — forward completed commands to shell
                                          and handle terminal resize (SIGWINCH)

PTY output
     │
     ▼
Emulator (CORE 6: Terminal)         — vt10x wrapper, tracks alt-screen mode
     │
     ▼
stdout                                  — raw PTY bytes passed through unchanged
```

## Core Details

### CORE 1: Input (`internal/input`)

**Responsibility:** Keystroke handling, input lifecycle, history navigation

**Files:**
- `dispatcher.go` — holds all shared state (linebuf, suggestions, history, popup). Exposes `Stop()` to signal the input loop from outside.
- `liner_input.go` — `RunWithLiner()`: the main input loop in raw mode. Reads bytes from a goroutine via a channel so it can `select` on the `done` channel and exit cleanly when the shell exits.

**Handles:** Ctrl-C, Ctrl-D, Ctrl-Z, Tab, Enter, arrow keys, bracketed paste, history navigation, ESC (exits app)

### CORE 2: Buffer + Parser (`internal/buffer`)

**Responsibility:** Line editing and context classification

**Files:**
- `linebuf.go` — cursor-aware UTF-8 line buffer. Operations: Append, Delete, DeleteWord, MoveCursor*, TruncateFrom, ReplaceLastWord, DisplayWidth.
- `parser.go` — tokenises the current line and classifies it into a `Context` (ContextCommandPartial, ContextSubcommand, ContextFlag, etc.) to drive suggestion lookup.

### CORE 3: Suggestion Engine (`internal/suggest`)

**Responsibility:** Context → Suggestions mapping via Spec registry

**Files:**
- `engine.go` — maps a `Context` to a `[]Suggestion` by looking up the registered `Spec` for the current command and filtering by partial prefix.
- `cache.go` — (reserved for caching dynamic generators).
- `specs/` — command specifications (types, registry, YAML loader)

**Spec Files:**
- `specs/spec.go` — core types: `Spec`, `Subcommand`, `Option`, `ArgSpec`, `Suggestion`.
- `specs/registry.go` — global `map[string]*Spec` with `Register` / `Get`.
- `specs/loader.go` — loads specs from YAML files in `specs/data/`.
- `specs/generators.go` — registry for dynamic suggestion generators.
- `specs/git.go`, `specs/docker.go`, `specs/kubectl.go`, `specs/npm.go`, `specs/go.go` — per-tool specs.

### CORE 4: Render/UI (`internal/render`)

**Responsibility:** Suggestion popup rendering and output serialization

**Files:**
- `popup.go` — renders the suggestion list below the cursor using ANSI save/restore cursor sequences. Shows bullet `●`/`○`, name, and description.
- `types.go` — `OutputOp` and `OutputChan` interface used to serialise all stdout writes through a single goroutine.

### CORE 5: PTY (`internal/pty`)

**Responsibility:** Shell process communication, PTY management

**Files:**
- `pty_unix.go` — `DisableEcho`, `Resizepty` for Unix.
- `pty_windows.go` — stub for Windows.

### CORE 6: Terminal Emulator (`internal/terminal`)

**Responsibility:** VT10x state tracking, alt-screen mode detection

**Files:**
- `emulator.go` — wraps `vt10x` to track terminal state. Used to detect alt-screen mode (vim, less, etc.) so keystrokes are passed through transparently instead of being interpreted by goshell.

### Supporting Cores

### `internal/config/`
- `config.go` — TOML config (theme colours, keybindings). Loaded from a file; falls back to `DefaultConfig()`.

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

## Suggestion Lifecycle

1. User types a character → `handleRuneInteractive` appends to `LineBuf` and calls `refreshSuggestions`.
2. `refreshSuggestions` → `Parser.GetCurrentContext` → `Engine.GetSuggestions` → `Popup.Render`.
3. Arrow Up/Down navigates the popup; Tab or Enter accepts and forwards to `ReplaceLastWord`.
4. On Enter the popup is erased, the command is forwarded to the PTY, and `LineBuf` is reset.

## Adding a New Command Spec

### Option 1: YAML (recommended for complex specs)

Create `suggest/<cmd>.yaml`:

```yaml
name: mycmd
subcommands:
  - name: run
    description: Run something
    options:
      - names: ["--verbose", "-v"]
        description: Verbose output
args:
  - name: target
    template: folder
```

### Option 2: Go code (for dynamic suggestions)

Create `internal/suggest/specs/<cmd>.go`:

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

## Architecture Reference

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed documentation on each core, including:
- Detailed type definitions and interfaces
- Complete data flow diagrams
- Concurrency model
- Error handling strategies
- ANSI escape sequence reference

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
| `gopkg.in/yaml.v3` | YAML spec file parsing |
