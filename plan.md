# Plan: Thay thế input layer bằng `liner`

## Mục tiêu
Thay thế phần xử lý input thủ công (`dispatcher.go` input loop, `RuneAccumulator`, escape sequence logic, key mapping) bằng `github.com/peterh/liner`, giữ nguyên custom suggestion engine và popup rendering.

## Cách tiếp cận: Hybrid — Liner cho line editing, giữ popup riêng

Liner không hỗ trợ popup rendering độc lập như goshell hiện tại. Thay vì bỏ hoàn toàn custom logic, ta dùng liner làm **line editor engine** và giữ suggestion/popup system hiện tại:

```
User keystrokes
    ↓
liner (line editing, cursor, history, word movement)
    ↓
On Tab/Enter/signal
    ↓
goshell suggestion engine + popup renderer
```

## Các bước thực hiện

### Bước 1: Thêm dependency
```bash
go get github.com/peterh/liner
```

### Bước 2: Refactor `dispatcher.go` — tách phần xử lý input

**Giữ lại** (vì liner không cover):
- Suggestion engine integration (`suggestEngine.GetSuggestions`)
- Popup rendering (`popup.Render`, `popup.Erase`, `popup.AcceptAndRedraw`)
- Parser context (`parser.GetCurrentContext`)
- Alt screen passthrough
- Bracketed paste mode
- Signal forwarding (Ctrl+C/D/Z)
- PTY write coordination (mutex)

**Thay thế bằng liner:**
- Raw byte reading → `liner.ReadLine()`
- Rune accumulation → liner handle
- Escape sequence parsing (arrows, Home, End) → liner handle
- Key type classification (`KeyType`, `KeyTypeFromRune`, `KeyTypeFromEscapeSeq`) → bỏ
- Line editing state (`LineBuf`) → liner quản lý nội bộ
- Line editing operations (append, delete, cursor movement) → liner handle
- `actionForKey`, `parseKeyName` → bỏ (liner có keymap riêng)
- `handleControl`, `handleEscapeByte`, `handleEscapeSeq`, `handleNavKey` → bỏ hoặc rút gọn

**Refactor cụ thể:**

Tạo `liner_input.go` mới trong `internal/input/`:

```go
func (d *Dispatcher) RunWithLiner() error {
    line := liner.NewLiner()
    defer line.Close()

    // History
    if f, err := os.UserHomeDir(); err == nil {
        line.ReadHistory(filepath.Join(f, ".goshell_history"))
        defer line.WriteHistory(filepath.Join(f, ".goshell_history"))
    }

    for {
        cmd, err := line.Prompt(d.prompt())
        if err == liner.ErrPromptAborted {
            line.AppendHistory("")
            continue
        }
        if err != nil {
            return err
        }
        line.AppendHistory(cmd)

        // Intercept Ctrl+C
        // Intercept Tab → show suggestions
        // Submit → d.handleSubmit(cmd)
    }
}
```

Sửa `dispatcher.go` — giữ lại suggestion/popup, rút gọn input handling:

```go
// Chỉ giữ: suggestion methods, popup, parser, PTY write coordination
// Bỏ: RuneAccumulator, KeyType/Action, handleControl/escape sequence logic
// Bỏ: hàm Run() cũ — thay bằng RunWithLiner()
```

### Bước 3: Cập nhật `main.go`

```go
// Bỏ: SetRawMode (liner tự làm)
// Bỏ: bufio.Reader, raw stdin (liner tự quản lý)
// Giữ: PTY setup, emulator, output channel, SIGWINCH resize
// Đổi: gọi dispatcher.RunWithLiner() thay vì dispatcher.Run()
```

### Bước 4: Giải quyết autocomplete integration

Dùng liner's `Completer` interface để trigger suggestion popup:

```go
line.SetCompleter(func(line string) []string {
    // Dùng parser + suggestEngine để lấy suggestions
    // Trả về list để liner hiển thị dạng simple completion
    // HOẶC: trigger custom popup và return nil
})
```

**Quyết định:** Dùng liner's built-in completion (simple dropdown) thay vì custom popup. Nếu muốn giữ popup riêng, trigger popup trên Tab và clear khi user chọn.

### Bước 5: Xử lý edge cases còn lại

- **Ctrl+C/D/Z signal forwarding**: dùng liner's interrupt handling, nếu cần thì intercept riêng
- **Bracketed paste**: bỏ (liner handle paste tốt hơn)
- **Alt screen passthrough**: kiểm tra `emulator.IsAltScreen()` trước khi gọi `line.Prompt()` — nếu active thì pass through trực tiếp
- **Config keybinding**: bỏ hoàn toàn `KeybindingConfig` + `actionForKey`, dùng liner's key bindings mặc định

### Bước 6: Dọn dẹp

Xoá các file/func không còn dùng:
- `RuneAccumulator` (liner handle)
- `KeyType`, `Action` enum và related functions
- `handleControl`, `handleEscapeByte`, `handleEscapeSeq`, `handleNavKey`
- `KeyTypeFromRune`, `KeyTypeFromEscapeSeq`, `parseKeyName`, `actionForKey`

Giản lược `linebuf.go` — chỉ giữ `RuneWidth()` và `RuneWidth` cho display width calculations (dùng cho popup).

### Bước 7: Fix bug trùng lặp output goroutine

`cmd/main.go` dòng 86-100: có 2 goroutine đọc cùng 1 channel. Xoá 1 cái.

## Files thay đổi

| File | Thay đổi |
|------|----------|
| `go.mod` | Thêm `github.com/peterh/liner` |
| `cmd/main.go` | Bỏ raw mode setup, fix duplicate goroutine, gọi `RunWithLiner()` |
| `internal/input/dispatcher.go` | Refactor — giữ suggestion/popup/PTY coordination, bỏ input loop |
| `internal/input/liner_input.go` | **MỚI** — integration liner với suggestion trigger |
| `internal/buffer/linebuf.go` | Giản lược, chỉ giữ `RuneWidth()` |
| `internal/config/config.go` | Bỏ `KeybindingConfig` (không cần nữa) |
| `internal/input/keytypes.go` | **XOÁ** — KeyType, Action, key mapping functions |
| `internal/input/escape.go` | **XOÁ** — escape sequence handling (liner handle) |

## Effort estimate
- **Low-Medium**: khoảng 150-200 dòng code mới, xoá ~400 dòng thủ công
- Risk thấp vì suggestion engine, popup, specs giữ nguyên
- Main risk: liner's display output format khác current popup format → có thể cần điều chỉnh `Popup.Render`

## Test plan
1. Build và chạy goshell — verify line editing hoạt động
2. Test cursor movement (arrow keys, Home/End)
3. Test backspace, Ctrl+U/K/W
4. Test autocomplete với Tab
5. Test Ctrl+C interrupt
6. Test Ctrl+D EOF, Ctrl+Z suspend
7. Test command execution (Enter)
8. Test history (up/down arrows sau khi chạy lệnh)
9. Test Unicode input (tiếng Việt, emoji, CJK)
10. Test Alt screen passthrough (vim/htop)
