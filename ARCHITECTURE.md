# goshell Architecture

## Overview

goshell là một terminal emulator viết bằng Go, lớp phủ trên shell có sẵn (bash/zsh) với tính năng autocomplete popup tương tác. Nó hoạt động bằng cách:

1. Spawn shell trong PTY (pseudo-terminal)
2. Intercept keystrokes ở raw mode trước khi chúng đến shell
3. Echo keystrokes local và hiển thị suggestions real-time
4. Khi user submit, forward command đến shell qua PTY

## Six Core Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  INPUT CORE (internal/input)                               │
│  Responsibility: Keystroke handling, input lifecycle        │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  BUFFER + PARSER CORE (internal/buffer)                     │
│  Responsibility: Line editing, context classification       │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  SUGGESTION ENGINE CORE (internal/suggest)                  │
│  Responsibility: Context → Suggestions mapping             │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  RENDER/UI CORE (internal/render)                          │
│  Responsibility: ANSI popup rendering                      │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  PTY CORE (internal/pty)                                    │
│  Responsibility: Shell process communication               │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│  TERMINAL EMULATOR CORE (internal/terminal)                 │
│  Responsibility: VT10x state, alt-screen detection        │
└─────────────────────────────────────────────────────────────┘
```

---

## CORE 1: Input (`internal/input`)

### Responsibility
Quản lý toàn bộ input lifecycle: đọc keystrokes từ stdin (raw mode), xử lý từng loại input event, coordinate giữa tất cả các core khác.

### Key Types

```go
type Dispatcher struct {
    ptyOut      *os.File           // PTY master fd
    ptyMu       *sync.Mutex        // Protect PTY writes from concurrent resize
    emulator    *terminal.Emulator  // Terminal state tracker
    outputChan  render.OutputChan   // Serialized stdout writes
    config      *config.Config

    // Line editing
    linebuf     *buffer.LineBuf     // UTF-8 line buffer with cursor
    parser      *buffer.Parser       // Context classifier
    runeBuf     []byte               // UTF-8 byte accumulator

    // Suggestion state
    suggestEngine *suggest.Engine
    popup          *render.Popup
    suggestions    []specs.Suggestion
    selected       int
    showing        bool

    // History navigation
    history      []string
    historyPos   int
    historySaved string

    // Screen tracking
    screenCol    int   // Cursor column relative to input start
    currentCWD   string // From OSC 6973
}
```

### Input Flow

```
Keystroke (byte)
    │
    ▼
feedByte() ─── UTF-8 decode ──── invalid → ignore
    │
    ▼ valid rune
handleRuneInteractive()
    │
    ├── Alt-screen mode? ──yes──► ptyWrite() direct pass-through
    │
    ├── Control char (<32)? ──yes──► ignore
    │
    └── Printable:
        linebuf.Append(r)
        outputChan.WriteOp(echo)
        refreshSuggestions(text)
```

### Key Handler Rules

| Input | Handler | Behavior |
|-------|---------|----------|
| Printable | `handleRuneInteractive` | Append to linebuf, echo, refresh suggestions |
| Tab (0x09) | Tab handler | Accept suggestion if showing, else trigger showSuggestions |
| Enter (0x0A) | `handleSubmitInteractive` | Erase popup, send cmd+LF to PTY, reset linebuf |
| Backspace (0x7F) | Backspace handler | Delete from linebuf, redraw, refresh |
| Ctrl+C (0x03) | Ctrl+C handler | Send ^C to PTY, reset linebuf, hide popup |
| Ctrl+D (0x04) | Ctrl+D handler | Send EOF (byte 4) to PTY |
| Ctrl+Z (0x1A) | Ctrl+Z handler | Send SIGTSTP to PTY |
| Ctrl+U (0x15) | Ctrl+U handler | Kill from cursor to start |
| Ctrl+K (0x0B) | Ctrl+K handler | Kill from cursor to end |
| Ctrl+W (0x17) | Ctrl+W handler | Kill word before cursor |
| Arrow Up (ESC[A) | Arrow handler | Navigate suggestions (if showing) or history |
| Arrow Down (ESC[B) | Arrow handler | Navigate suggestions (if showing) or history |
| Arrow Left (ESC[D) | Arrow handler | Move cursor left |
| Arrow Right (ESC[C) | Arrow handler | Move cursor right |
| ESC (first) | Escape handler | Hide suggestions |
| ESC (second, <200ms) | Escape handler | Stop dispatcher, exit |
| Bracketed paste start (ESC[200~) | paste handler | Enter paste mode |
| Bracketed paste end (ESC[201~) | paste handler | Write accumulated paste to PTY |

### Pass-through Mode

Khi `emulator.IsAltScreen()` trả về true (vim, less, htop đang chạy), tất cả keystrokes được forward thẳng qua PTY thay vì buffer local. Điều này đảm bảo full-screen apps hoạt động đúng.

### PTY Write Protection

PTY writes sử dụng mutex để tránh race với SIGWINCH handler:

```go
func (d *Dispatcher) ptyWrite(data []byte) {
    d.ptyMu.Lock()
    _, err := d.ptyOut.Write(data)
    d.ptyMu.Unlock()
}
```

### Public API

```go
func NewDispatcher(ptyOut *os.File, emulator *terminal.Emulator, output render.OutputChan, ptyMu *sync.Mutex) *Dispatcher
func (d *Dispatcher) RunWithLiner() error  // Main input loop
func (d *Dispatcher) Stop()                 // Signal exit
func (d *Dispatcher) GetCWD() string        // Current working directory from OSC 6973
func (d *Dispatcher) IsAltScreen() bool     // Terminal mode query
```

---

## CORE 2: Buffer + Parser (`internal/buffer`)

### Responsibility
Quản lý dòng lệnh đang gõ (cursor-aware UTF-8 buffer) và phân tích ngữ cảnh để xác định loại suggestion cần hiển thị.

### LineBuf

```go
type LineBuf struct {
    runes  []rune   // UTF-8 runes
    cursor int      // Cursor position (0 = before first char)
}
```

**Operations:**

```go
func (lb *LineBuf) Append(r rune)                    // Insert rune at cursor
func (lb *LineBuf) Delete() bool                       // Delete rune after cursor (backspace)
func (lb *LineBuf) DeleteWord()                       // Delete word before cursor
func (lb *LineBuf) TruncateFrom(pos int)              // Kill from pos to end
func (lb *LineBuf) MoveCursorToStart()                 // Ctrl+U support
func (lb *LineBuf) MoveCursorLeft() bool               // Arrow left
func (lb *LineBuf) MoveCursorRight() bool              // Arrow right
func (lb *LineBuf) MoveCursorToEnd()                  // Move to end
func (lb *LineBuf) SetCursor(pos int)                  // Set cursor position
func (lb *LineBuf) Cursor() int                       // Get cursor position
func (lb *LineBuf) ReplaceLastWord(s string)           // Replace partial word with suggestion
func (lb *LineBuf) AppendWord(s string)                // Append word with trailing space
func (lb *LineBuf) String() string                     // Get full content
func (lb *LineBuf) SetString(s string)                 // Replace content
func (lb *LineBuf) Len() int                           // Content length
func (lb *LineBuf) Reset()                             // Clear content
func (lb *LineBuf) DisplayWidth() int                  // Visual width (Unicode-aware)
func (lb *LineBuf) CursorDisplayWidth() int            // Cursor visual position
```

### Parser

```go
type ContextLevel int

const (
    ContextCommand ContextLevel = iota
    ContextCommandPartial          // "gi" - typing command
    ContextSubcommand             // "git " - after command
    ContextSubcommandPartial      // "git co" - typing subcommand
    ContextFlag                   // "git commit -" - flag context
    ContextFlagPartial            // "git commit --v" - typing flag
    ContextArg                    // "git commit -m " - after flag
    ContextArgPartial             // "git commit -m \"mes" - typing arg
)

type Context struct {
    Level       ContextLevel
    Command     string
    Subcommand  string
    Flag        string
    PartialWord string
}
```

**Classification Rules:**

| Input | Context |
|-------|---------|
| `""` | `ContextCommand` |
| `"gi"` | `ContextCommandPartial{Command: "gi"}` |
| `"git "` | `ContextSubcommand{Command: "git"}` |
| `"git che"` | `ContextSubcommandPartial{Command: "git", Subcommand: "che"}` |
| `"git checkout "` | `ContextArg{Command: "git", Subcommand: "checkout"}` |
| `"git checkout ma"` | `ContextArgPartial{Command: "git", Subcommand: "checkout"}` |
| `"git -"` | `ContextFlag{Command: "git"}` |
| `"git --v"` | `ContextFlagPartial{Command: "git", Flag: "--v"}` |

---

## CORE 3: Suggestion Engine (`internal/suggest`)

### Responsibility
Từ Context (từ Parser) và partial input → tạo danh sách suggestions phù hợp bằng cách tra registry và filter theo prefix.

### Engine

```go
type Engine struct{}

func NewEngine() *Engine
func (e *Engine) GetSuggestions(buf *buffer.LineBuf, ctx *buffer.Context) []specs.Suggestion
```

**Suggestion Flow by Context:**

```
ContextCommandPartial:
  → matchCommand(prefix)
  → Filter specs.RegisteredCommands() by HasPrefix(prefix)

ContextCommand:
  → getSubcommandSuggestions(spec, "")
  → If spec has no subcommands, fall through to args

ContextSubcommand:
  → getSubcommandSuggestions(spec, "")
  → If no match, suggestFromArgSpecs(spec.Args)

ContextSubcommandPartial:
  → getSubcommandSuggestions(spec, partial)
  → If no match, suggestFromArgSpecs(spec.Args, partial)

ContextFlag / ContextFlagPartial:
  → getFlagSuggestions(ctx)
  → Collect global options + subcommand options
  → Filter by flag prefix

ContextArg / ContextArgPartial:
  → getArgSuggestions(ctx)
  → Prefer subcommand Args over spec Args
  → Call generators / read filesystem based on ArgSpec.Template
```

### Specs Registry (`internal/suggest/specs`)

```go
package specs

// Suggestion types
type SuggestionKind int
const (
    KindSubcommand SuggestionKind = iota
    KindOption
    KindArg
    KindFile
    KindFolder
    KindValue
)

// Argument template for filesystem suggestions
type Template int
const (
    TemplateNone Template = iota
    TemplateFileSystem
    TemplateFolder
)

// Core types
type Spec struct {
    Name        string
    Subcommands []Subcommand
    Options     []Option
    Args        []ArgSpec
}

type Subcommand struct {
    Name        string
    Description string
    Subcommands []Subcommand  // Nested subcommands
    Options     []Option
    Args        []ArgSpec
}

type Option struct {
    Names       []string  // e.g., ["--verbose", "-v"]
    Description string
    Args        []ArgSpec
}

type ArgSpec struct {
    Name       string
    Generator  string  // Dynamic suggestion via generator function
    IsVariadic bool
    Template   Template // For filesystem templates
}

// Registry functions
func Register(name string, s *Spec)
func Get(name string) *Spec
func RegisteredCommands() []string

// Generator registry for dynamic suggestions
func RegisterGenerator(name string, fn func() []Suggestion)
func GetGenerator(name string) func() []Suggestion
```

### YAML Loading

Specs được load từ YAML files trong `specs/data/`:

```go
//go:embed data/*.yaml
var dataFS embed.FS

func init() {
    entries, _ := dataFS.ReadDir("data")
    for _, entry := range entries {
        data, _ := dataFS.ReadFile("data/" + entry.Name())
        spec := MustLoadYAML(data)
        Register(spec.Name, spec)
    }
}
```

### Adding New Command Spec

Tạo file `data/<command>.yaml`:

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

Hoặc đăng ký trực tiếp bằng Go:

```go
func init() {
    Register("mycmd", &Spec{
        Name: "mycmd",
        Subcommands: []Subcommand{
            {Name: "run", Description: "Run something"},
        },
    })
}
```

---

## CORE 4: Render/UI (`internal/render`)

### Responsibility
Hiển thị suggestion popup bằng ANSI escape sequences và serializing tất cả stdout writes qua một goroutine để tránh race conditions.

### Output Serialization

```go
type OutputOp struct {
    Data []byte
}

type OutputChan interface {
    WriteOp(data []byte)
}

// cmd/main.go
type outputWriter struct {
    ch chan<- render.OutputOp
}

func (w *outputWriter) WriteOp(op render.OutputOp) {
    w.ch <- op  // Blocking send
}

go func() {
    for op := range ch {
        os.Stdout.Write(op.Data)
    }
}()
```

### Popup Rendering

```go
type Popup struct {
    maxHeight int  // Max visible suggestions (default 10)
    width     int  // Max name width for alignment (default 40)
    output    OutputChan
}

func NewPopup(output OutputChan) *Popup
```

**Render Strategy:**

```
1. ESC[s           — Save cursor (on input line)
2. ESC[1B          — Move down 1 line (to first popup line)
3. [content]       — Build popup content
4. ESC[u           — Restore cursor to input line
```

**Content Building:**

```
For each visible suggestion (max maxHeight):
  - "│ " prefix (box drawing)
  - "● " or "○ " bullet (selected vs unselected)
  - Name (truncated to 18 chars)
  - " : Description" for selected item (when > 5 items)
  - "◆ - n/n rules" footer when > 5 items (cyan, bold)
  - ESC[J — Clear to end of screen
```

**Erase Strategy:**

```
1. ESC[s   — Save cursor (on input line)
2. ESC[1B  — Move down into popup area
3. ESC[0G  — Column 0
4. ESC[J   — Erase to end of screen (popup only)
5. ESC[u   — Restore cursor to input line
```

**AcceptAndRedraw (after Tab/Enter on suggestion):**

```
1. ESC[s   — Save cursor
2. ESC[nB  — Move down past popup (n = numLines)
3. ESC[n+1A — Move up back to input line, col 0
4. ESC[0G  — Column 0
5. ESC[J   — Erase popup area
6. [newLine] — Print input with accepted suggestion
7. ESC[u   — Restore cursor
```

### ANSI Escape Sequences Used

| Sequence | Meaning |
|----------|---------|
| `\033[s` | Save cursor position |
| `\033[u` | Restore cursor position |
| `\033[K` | Erase from cursor to end of line |
| `\033[J` | Erase to end of screen |
| `\033[nB` | Move cursor down n lines |
| `\033[nA` | Move cursor up n lines |
| `\033[nD` | Move cursor left n columns |
| `\033[0G` | Move cursor to column 0 |
| `\033[?25l` | Hide cursor |
| `\033[?25h` | Show cursor |
| `\033[1;36m` | Cyan color |
| `\033[1m` | Bold |
| `\033[0m` | Reset attributes |

---

## CORE 5: PTY (`internal/pty`)

### Responsibility
Tạo và quản lý pseudo-terminal cho shell process, xử lý echo và resize.

### PTY Creation

```go
cmd := exec.Command(shellPath)  // bash or $SHELL
ptm, err := pty.Start(cmd)      // Returns PTY master
// PTY slave is stdin/stdout/stderr of cmd
```

### Echo Handling

Shell's termios echo bị disable để tránh double echo (goshell tự echo):

```go
func DisableEcho(ptm *pty.File) error {
    // Get current termios
    // Disable ICANON | ECHO
    // Set new termios
}
```

Tại sao cần: Khi user nhấn Enter, shell echo lại command. Nếu không disable, sẽ có race giữa shell echo và goshell's eraseInputDisplay.

### Window Resize

SIGWINCH handler resize PTY khi terminal window thay đổi:

```go
signal.Notify(sigChan, syscall.SIGWINCH)
go func() {
    for range sigChan {
        ptyMu.Lock()
        pty.Resizepty(ptm)
        ptyMu.Unlock()
    }
}()
```

---

## CORE 6: Terminal Emulator (`internal/terminal`)

### Responsibility
Wrap vt10x để track terminal state, đặc biệt là alt-screen mode detection để pass-through keystrokes khi full-screen apps đang chạy.

### VT10x Integration

```go
type Emulator struct {
    term   vt10x.Terminal
    handler OSCHandler
}
```

vt10x là VT100/VT10x terminal emulator implementation, track mode flags:

```go
func (e *Emulator) IsAltScreen() bool {
    return e.term.Mode() & vt10x.ModeAltScreen != 0
}
```

### Alt-Screen Mode

DCEB (DECSET 47) or DECNM (DECRST 1047) bật alternate screen buffer. Full-screen apps như vim, less, htop dùng mode này.

**Without Detection Problem:**
- vim bật alt-screen, cursor at (0,0) trên new screen
- goshell không biết, vẫn echo keystrokes ở old screen position
- Kết quả: ghosting, cursor flicker, corrupted display

**With Detection Solution:**
```go
if d.emulator.IsAltScreen() {
    d.ptyWrite(buf[:n])  // Forward keystrokes directly
    return
}
```

### OSC 6973 Handler (Working Directory Tracking)

Shell có thể gửi current working directory qua OSC (Operating System Command):

```bash
# In shell prompt (e.g., starship)
printf "\033]6973;CWD;%s\007" "$(pwd)"
```

goshell parse sequence này để track CWD:

```go
func (e *Emulator) parseOSC(b []byte) {
    // Detect ESC ] 6973 ; CWD ; <path> BEL/ST
    // Call e.handler.OnCWD(path)
}
```

Dispatcher implements `OSCHandler.OnCWD()` để update `currentCWD`.

---

## Data Flow Diagrams

### Input → Suggestion → Accept

```
User types "git co"
    │
    ▼
liner_input.RunWithLiner() reads 'g'
    │
    ▼
handleRuneInteractive('g')
    │
    ├── linebuf.Append('g')
    ├── outputChan.WriteOp(echo 'g')
    └── refreshSuggestions("g")
            │
            ▼
    Parser.GetCurrentContext(linebuf)
            │
            ▼
    Context{Level: ContextCommandPartial, Command: "g"}
            │
            ▼
    Engine.GetSuggestions(buf, ctx)
            │
            ▼
    specs.RegisteredCommands() → filter "git" matches
            │
            ▼
    Popup.Render([Suggestion{Name: "git", Kind: KindSubcommand}])
            │
            ▼
User presses Tab
    │
    ▼
Tab handler: ReplaceLastWord("git"), Erase popup
    │
    ▼
User types " checkout --v"
    │
    ▼
handleRuneInteractive(' ')
    │
    ├── linebuf.Append(' ')
    └── refreshSuggestions("git checkout --v")
            │
            ▼
    Context{Level: ContextFlagPartial, Command: "git", Subcommand: "checkout", Flag: "--v"}
            │
            ▼
    Engine.GetFlagSuggestions(ctx)
            │
            ▼
    Suggestion{Name: "--verbose", Description: "Verbose output"}
            │
            ▼
Popup.Render([--verbose])
    │
    ▼
User presses Enter
    │
    ▼
handleSubmitInteractive()
    │
    ├── popup.Erase()
    ├── eraseInputDisplay()  -- erase locally-echoed input
    ├── ptyWriteString("git checkout --verbose")
    ├── ptyWrite([]byte{10})  -- LF
    └── linebuf.Reset()
            │
            ▼
Shell receives "git checkout --verbose\n"
```

### PTY Output Pass-through

```
Shell runs: ls -la
    │
    ▼
ptm.Read() on PTY master
    │
    ▼
emulator.Write(buf[:n])  -- update vt10x state
    │
    ▼
ch <- OutputOp{Data: buf[:n]}
    │
    ▼
Output goroutine: os.Stdout.Write(op.Data)
    │
    ▼
Terminal displays output
```

### Alt-Screen Pass-through

```
User types: vim
    │
    ▼
ptyWriteString("vim")
ptyWrite([]byte{10})
    │
    ▼
Shell starts vim
    │
    ▼
vim sends ESC[?1047h (enter alt-screen)
    │
    ▼
pty output → emulator.Write()
    │
    ▼
emulator.term.Mode() now has ModeAltScreen
    │
    ▼
User types 'i' (insert mode)
    │
    ▼
liner_input reads 'i'
    │
    ▼
IsAltScreen() == true
    │
    ▼
ptyWrite([]byte{'i'})  -- direct forward
    │
    ▼
vim receives 'i' directly
```

---

## Error Handling

### PTY Write Failure
```go
func (d *Dispatcher) ptyWrite(data []byte) {
    d.ptyMu.Lock()
    _, err := d.ptyOut.Write(data)
    d.ptyMu.Unlock()
    if err != nil {
        fmt.Fprintf(os.Stderr, "ptyWrite: %v\n", err)
    }
}
```
Lỗi được log nhưng không fail để tránh crash khi shell exit đột ngột.

### Shell Process Exit
```go
go func() {
    buf := make([]byte, 4096)
    for {
        n, err := ptm.Read(buf)
        if err != nil {
            close(ch)
            dispatcher.Stop()  // Signal input loop to exit
            return
        }
        emulator.Write(buf[:n])
        ch <- render.OutputOp{Data: append([]byte(nil), buf[:n]...)}
    }
}()

// In dispatcher.RunWithLiner():
select {
case <-d.done:
    return nil  // Clean exit when shell exits
case res = <-byteCh:
    // Process input
}
```

### Raw Mode Restore
```go
oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
defer term.Restore(int(os.Stdin.Fd()), oldState)
```
Đảm bảo terminal state được khôi phục khi goshell exit (cả clean lẫn crash).

---

## Concurrency Model

```
Main goroutine:
  dispatcher.RunWithLiner()  -- blocks on input loop

Background goroutines:
  1. PTY reader:
     for { ptm.Read() → emulator.Write() → ch <- OutputOp }

  2. Output writer:
     for { ch → os.Stdout.Write() }

  3. SIGWINCH handler:
     for { sigChan → ptyMu.Lock() → Resizepty() → ptyMu.Unlock() }
```

**Synchronization:**
- `ptyMu` mutex: protects PTY writes vs concurrent resize
- `ch chan OutputOp`: serializes stdout writes (single goroutine owns stdout)
- `d.done chan struct{}`: signals input loop exit from PTY reader

**Why single output goroutine:**
PTy output và input echo có thể write đồng thời → race condition → interleaved output. Channel serializes all writes through one goroutine.
