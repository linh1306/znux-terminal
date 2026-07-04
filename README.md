# znux — Intelligent Terminal Autocomplete

> A lightweight terminal emulator written in Go that wraps your existing shell (bash/zsh) and layers an interactive, context-aware autocomplete popup on top — without replacing your shell or touching its configuration.

```
$ git che█
│ ● checkout     : Switch branches or restore working tree files
│ ○ cherry-pick
│ ○ cherry
◆ - 3/3 rules
```

---

## Features

- 🚀 **Zero configuration** — drops in on top of your existing shell
- 🎯 **Context-aware suggestions** — understands commands, subcommands, flags, and arguments
- ⌨️  **Full line editing** — cursor movement, word kill, history navigation (powered by `liner`)
- 🖥️  **Alt-screen passthrough** — vim, less, htop, and other full-screen apps work transparently
- 📋 **Bracketed paste** support — paste large blocks safely
- 🗂️  **Filesystem suggestions** — file/folder completion for supported commands
- 🔌 **Extensible via YAML** — add autocomplete specs for any CLI tool without recompiling

---

## Installation

### Prerequisites

- Go 1.21 or higher
- A POSIX-compatible shell (bash or zsh)

### Build from source

```bash
git clone https://github.com/nguyenlinh13602/goshell
cd goshell

# Build binary to dist/
make build

# Or install directly to ~/.local/bin/znux
make install
```

### Add to PATH (if using `make build`)

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Then simply run:

```bash
znux
```

---

## Usage

Launch `znux` as you would any shell. It wraps your current `$SHELL` (bash/zsh) and intercepts keystrokes to show real-time suggestions.

```bash
znux
```

### Keybindings

| Key | Action |
|-----|--------|
| `Tab` | Accept the highlighted suggestion |
| `↑` / `↓` | Navigate suggestions (or scroll history when popup is closed) |
| `←` / `→` | Move cursor within the line |
| `Enter` | Submit command to the shell |
| `Ctrl-C` | Cancel current input |
| `Ctrl-D` | Send EOF to shell |
| `Ctrl-Z` | Suspend current process (SIGTSTP) |
| `Ctrl-U` | Kill from cursor to start of line |
| `Ctrl-K` | Kill from cursor to end of line |
| `Ctrl-W` | Delete word before cursor |
| `ESC` (once) | Close suggestion popup |
| `ESC` (twice, < 200ms) | Exit znux |

---

## How It Works

znux intercepts keystrokes **before** they reach the shell:

```
stdin (raw mode)
     │
     ▼
Dispatcher ─── LineBuf (UTF-8 buffer with cursor)
     │       ├── Parser  → classifies context (command / subcommand / flag / arg)
     │       ├── Engine  → looks up Spec, filters suggestions by prefix
     │       └── Popup   → renders ANSI popup below cursor
     │
     └──► PTY write → shell process (bash/zsh)

PTY output
     │
     ▼
Terminal Emulator (vt10x) ─── alt-screen detection
     │
     ▼
stdout (raw PTY bytes, unchanged)
```

When you press **Enter**, znux erases the popup, forwards the complete command to the underlying shell via PTY, and resets the buffer.

---

## Supported Commands (built-in specs)

| Command | Notes |
|---------|-------|
| `git` | Full subcommand, flag, and common arg support |
| `docker` | Subcommands, options, image/container args |
| `docker compose` | Compose-specific subcommands and flags |
| `kubectl` | Common Kubernetes CLI commands |
| `go` | Go toolchain subcommands |
| `npm` | Package manager commands |
| `cd` | Directory completion |
| `mkdir` | Directory creation flags |
| `rm` | Removal flags |
| `cat` | File completion |
| `fuser` | Process/file inspection flags |
| `claude` | Claude CLI subcommands |
| `codex` | Codex CLI subcommands |

---

## Adding Custom Command Specs

### Option 1: YAML (recommended)

Create a file at `suggest/<cmd>.yaml` next to the binary:

```yaml
name: mycmd
subcommands:
  - name: run
    description: Run something
    options:
      - names: ["--verbose", "-v"]
        description: Enable verbose output
      - names: ["--output", "-o"]
        description: Output file path
    args:
      - name: target
        template: folder   # enables filesystem completion
  - name: build
    description: Build the project
args:
  - name: config
    template: file
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
            {Names: []string{"--verbose", "-v"}, Description: "Enable verbose output"},
        },
    })
}
```

The suggestion engine picks it up automatically — no other changes required.

---

## Architecture

znux is built on **six independent cores**:

| Core | Package | Responsibility |
|------|---------|----------------|
| **Input** | `internal/input` | Keystroke handling, input lifecycle, history |
| **Buffer + Parser** | `internal/buffer` | UTF-8 line buffer, context classification |
| **Suggestion Engine** | `internal/suggest` | Context → Suggestions via Spec registry |
| **Render/UI** | `internal/render` | ANSI popup rendering, output serialization |
| **PTY** | `internal/pty` | Shell process management, window resize |
| **Terminal Emulator** | `internal/terminal` | VT10x state tracking, alt-screen detection |

For a detailed breakdown of types, data flows, and concurrency model, see [ARCHITECTURE.md](ARCHITECTURE.md).

---

## Dependencies

| Package | Purpose |
|---------|---------|
| [`github.com/creack/pty`](https://github.com/creack/pty) | PTY creation and resizing |
| [`github.com/hinshun/vt10x`](https://github.com/hinshun/vt10x) | VT100/VT10x terminal emulation (alt-screen detection) |
| [`github.com/peterh/liner`](https://github.com/peterh/liner) | Line editing and history persistence |
| [`github.com/BurntSushi/toml`](https://github.com/BurntSushi/toml) | TOML config file parsing |
| [`golang.org/x/term`](https://pkg.go.dev/golang.org/x/term) | Raw mode on stdin |
| [`github.com/mattn/go-runewidth`](https://github.com/mattn/go-runewidth) | Unicode display-width calculations |
| [`gopkg.in/yaml.v3`](https://pkg.go.dev/gopkg.in/yaml.v3) | YAML spec file parsing |

---

## Development

```bash
# Run tests
make test

# Build binary
make build

# Install to ~/.local/bin
make install

# Clean build artifacts
make clean
```

### Project Structure

```
znux-terminal/
├── cmd/                    # Main entry point
├── internal/
│   ├── input/              # Keystroke dispatcher, input loop
│   ├── buffer/             # Line buffer, context parser
│   ├── suggest/            # Suggestion engine and Spec registry
│   │   └── specs/          # Per-tool spec definitions (Go)
│   ├── render/             # ANSI popup renderer, output serializer
│   ├── pty/                # PTY management (Unix/Windows stubs)
│   ├── terminal/           # VT10x emulator wrapper
│   └── config/             # TOML configuration loader
├── suggest/                # YAML spec files (loaded at runtime)
├── ARCHITECTURE.md         # Detailed architecture documentation
└── Makefile
```

---

## License

MIT License — see [LICENSE](LICENSE) for details.
