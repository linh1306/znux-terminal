package input

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"github.com/nguyenlinh13602/goshell/internal/buffer"
	"github.com/nguyenlinh13602/goshell/internal/render"
	"github.com/nguyenlinh13602/goshell/internal/suggest/specs"
	"github.com/peterh/liner"
	"golang.org/x/term"
)

type readResult struct {
	b   byte
	err error
}

// navigateHistoryUp moves to the previous history entry.
func (d *Dispatcher) navigateHistoryUp() {
	if len(d.history) == 0 || d.historyPos == 0 {
		return
	}
	if d.historyPos == len(d.history) {
		d.historySaved = d.linebuf.String()
	}
	d.historyPos--
	d.linebuf.SetString(d.history[d.historyPos])
	d.redrawCurrentLine()
}

// navigateHistoryDown moves to the next history entry (or restores saved input).
func (d *Dispatcher) navigateHistoryDown() {
	if d.historyPos >= len(d.history) {
		return
	}
	d.historyPos++
	if d.historyPos == len(d.history) {
		d.linebuf.SetString(d.historySaved)
	} else {
		d.linebuf.SetString(d.history[d.historyPos])
	}
	d.redrawCurrentLine()
}

// RunWithLiner starts the input loop with real-time suggestion display.
// It uses a custom input loop for keystroke control and liner for history.
func (d *Dispatcher) RunWithLiner() error {
	// Use liner only for history management
	line := liner.NewLiner()
	defer line.Close()

	// Load history from file into both liner (persistence) and dispatcher (Up/Down nav)
	historyPath := d.historyPath()
	if f, err := os.Open(historyPath); err == nil {
		line.ReadHistory(f)
		f.Close()
	}
	d.loadHistory(historyPath)
	defer func() {
		if f, err := os.Create(historyPath); err == nil {
			line.WriteHistory(f)
			f.Close()
		}
	}()

	// Set up raw mode on stdin for reading keystrokes
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	reader := bufio.NewReaderSize(os.Stdin, 8192)

	// Reset screen cursor tracking — input starts right after shell prompt.
	d.screenCol = 0

	byteCh := make(chan readResult, 8)
	go func() {
		for {
			b, err := reader.ReadByte()
			byteCh <- readResult{b, err}
			if err != nil {
				return
			}
		}
	}()

	for {
		var res readResult
		select {
		case <-d.done:
			return nil
		case res = <-byteCh:
		}

		if res.err != nil {
			if res.err == io.EOF {
				return nil
			}
			return res.err
		}
		b := res.b

		if d.shouldPassThroughInput() {
			d.ptyWrite([]byte{b})
			continue
		}

		// Handle escape sequences (arrow keys, etc.)
		if b == 27 {
			d.handleEscape(byteCh)
			continue
		}

		// Handle Ctrl+C
		if b == 3 {
			line.AppendHistory("")
			d.linebuf.Reset()
			d.hideSuggestions()
			// Show the interrupt marker locally; the shell will print the new
			// line and prompt after receiving SIGINT.
			d.outputChan.WriteOp(render.OutputOp{Data: []byte("^C")})
			d.screenCol = 0
			d.ptyWrite([]byte{3})
			continue
		}

		// Handle Ctrl+D (EOF)
		if b == 4 {
			d.ptyWrite([]byte{4})
			continue
		}

		// Handle Ctrl+Z (suspend)
		if b == 26 {
			d.ptyWrite([]byte{26})
			continue
		}

		// Handle Tab — accept current suggestion immediately into linebuf,
		// then refresh suggestions for the new buffer content.
		if b == 9 {
			if d.showing && len(d.suggestions) > 0 {
				s := d.suggestions[d.selected]
				if s.Kind == specs.KindInstall {
					d.linebuf.SetString(s.Completion())
				} else {
					d.linebuf.ReplaceLastWord(s.Completion())
				}
				d.popup.Erase(len(d.suggestions))
				d.showing = false
				d.suggestions = nil
				d.redrawCurrentLine()
				d.refreshSuggestions(d.linebuf.String())
			} else if !d.showing {
				d.showSuggestionsFromCurrentLine()
			}
			continue
		}

		// Handle Enter — submit
		if b == 10 || b == 13 {
			d.handleSubmitInteractive(line)
			continue
		}

		// Handle Backspace
		if b == 127 || b == 8 {
			if d.linebuf.Len() > 0 {
				d.linebuf.Delete()
				lastText := d.linebuf.String()
				d.redrawCurrentLine()
				d.refreshSuggestions(lastText)
			}
			continue
		}

		// Handle Ctrl+U (kill from cursor to start)
		if b == 21 {
			d.linebuf.MoveCursorToStart()
			d.linebuf.Reset()
			d.redrawCurrentLine()
			d.refreshSuggestions("")
			continue
		}

		// Handle Ctrl+K (kill from cursor to end)
		if b == 11 {
			d.linebuf.TruncateFrom(d.linebuf.Cursor())
			lastText := d.linebuf.String()
			d.redrawCurrentLine()
			d.refreshSuggestions(lastText)
			continue
		}

		// Handle Ctrl+W (kill word before cursor)
		if b == 23 {
			d.linebuf.DeleteWord()
			lastText := d.linebuf.String()
			d.redrawCurrentLine()
			d.refreshSuggestions(lastText)
			continue
		}

		// Accumulate UTF-8 bytes into rune
		runes, valid := d.feedByte(b)
		if !valid {
			continue
		}

		for _, r := range runes {
			d.handleRuneInteractive(r)
		}
	}
}

// feedByte accumulates a byte into the current rune being built.
func (d *Dispatcher) feedByte(b byte) (runes []rune, valid bool) {
	d.runeBuf = append(d.runeBuf, b)

	r, size := utf8.DecodeRune(d.runeBuf)
	if r == utf8.RuneError && size == 1 && len(d.runeBuf) < utf8.UTFMax {
		return nil, false
	}

	runes = []rune{r}
	d.runeBuf = d.runeBuf[:0]
	return runes, true
}

// handleEscape reads and processes an escape sequence from the terminal.
// Reads subsequent bytes from byteCh (same channel as the main loop) to avoid
// racing with the reader goroutine.
func (d *Dispatcher) handleEscape(byteCh <-chan readResult) {
	seq := []byte{27}
	timeout := time.After(50 * time.Millisecond)

	for len(seq) < 8 {
		select {
		case res := <-byteCh:
			if res.err != nil {
				break
			}
			seq = append(seq, res.b)
			if isCompleteEscapeSeq(seq) {
				goto done
			}
		case <-timeout:
			goto done
		}
	}
done:
	d.processEscapeSeq(seq)
}

// isCompleteEscapeSeq checks if an escape sequence is complete.
func isCompleteEscapeSeq(seq []byte) bool {
	if len(seq) < 2 {
		return false
	}
	if seq[1] == '[' {
		if len(seq) >= 3 {
			b := seq[2]
			if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') {
				return true
			}
			if b == '~' {
				return true
			}
		}
		if len(seq) >= 4 && seq[2] == '1' && seq[3] == '~' {
			return true
		}
		if len(seq) >= 4 && seq[2] == '4' && seq[3] == '~' {
			return true
		}
		if len(seq) >= 5 && seq[2] == '2' && seq[3] == '0' && seq[4] == '0' {
			return true
		}
		if len(seq) >= 5 && seq[2] == '2' && seq[3] == '0' && seq[4] == '1' {
			return true
		}
		if len(seq) >= 6 && seq[2] == '1' && seq[3] == ';' && seq[4] >= '2' && seq[4] <= '8' {
			b := seq[5]
			return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
		}
	}
	if seq[1] == 'O' && len(seq) >= 3 {
		b := seq[2]
		if b >= 'A' && b <= 'D' {
			return true
		}
	}
	return false
}

// processEscapeSeq handles the parsed escape sequence.
func (d *Dispatcher) processEscapeSeq(seq []byte) {
	// Bracketed paste start: ESC [ 2 0 0 ~
	if len(seq) >= 5 && seq[0] == 27 && seq[1] == '[' && seq[2] == '2' && seq[3] == '0' && seq[4] == '0' {
		d.pasteActive = true
		d.pasteBuffer.Reset()
		return
	}

	// Bracketed paste end: ESC [ 2 0 1 ~
	if len(seq) >= 5 && seq[0] == 27 && seq[1] == '[' && seq[2] == '2' && seq[3] == '0' && seq[4] == '1' {
		if d.pasteActive {
			d.pasteActive = false
			if d.pasteBuffer.Len() > 0 {
				d.ptyWriteString(d.pasteBuffer.String())
				d.pasteBuffer.Reset()
			}
		}
		return
	}

	// Paste mode — accumulate bytes
	if d.pasteActive {
		d.pasteBuffer.Write(seq)
		return
	}

	// Arrow key navigation — always works, not just when showing suggestions
	if len(seq) >= 3 && seq[0] == 27 {
		var key rune
		shift := false
		if seq[1] == '[' {
			if len(seq) >= 6 && seq[2] == '1' && seq[3] == ';' {
				shift = seq[4] == '2'
				switch seq[5] {
				case 'A':
					key = 'u' // up
				case 'B':
					key = 'd' // down
				case 'C':
					key = 'r' // right
				case 'D':
					key = 'l' // left
				}
			} else {
				switch seq[2] {
				case 'A':
					key = 'u' // up
				case 'B':
					key = 'd' // down
				case 'C':
					key = 'r' // right
				case 'D':
					key = 'l' // left
				}
			}
		} else if seq[1] == 'O' && len(seq) >= 3 {
			switch seq[2] {
			case 'A':
				key = 'u'
			case 'B':
				key = 'd'
			case 'C':
				key = 'r'
			case 'D':
				key = 'l'
			}
		}

		if shift {
			switch key {
			case 'u':
				d.hideSuggestions()
				d.outputChan.WriteOp(render.OutputOp{Kind: render.OutputOpHistoryPrev})
			case 'd':
				d.hideSuggestions()
				d.outputChan.WriteOp(render.OutputOp{Kind: render.OutputOpHistoryNext})
			}
			return
		}

		switch key {
		case 'u':
			if d.showing {
				d.prevSuggestion()
			} else {
				d.navigateHistoryUp()
			}
		case 'd':
			if d.showing {
				d.nextSuggestion()
			} else {
				d.navigateHistoryDown()
			}
		case 'r':
			// Move cursor right
			d.moveCursorRight()
		case 'l':
			// Move cursor left
			d.moveCursorLeft()
		}
		return
	}

	// Plain Escape exits the program. Treat repeated ESC presses the same way
	// so they are not forwarded to the shell as visible ^[ characters.
	if len(seq) == 1 || seq[1] == 27 {
		d.hideSuggestions()
		d.outputChan.WriteOp(render.OutputOp{Data: []byte("\r\n")})
		d.screenCol = 0
		d.Stop()
		return
	}

	// Unknown escape — pass through to PTY
	d.ptyWrite(seq)
}

// moveCursorLeft moves cursor left by one rune position
func (d *Dispatcher) moveCursorLeft() {
	if d.linebuf.MoveCursorLeft() {
		d.redrawCurrentLine()
	}
}

// moveCursorRight moves cursor right by one rune position
func (d *Dispatcher) moveCursorRight() {
	if d.linebuf.MoveCursorRight() {
		d.redrawCurrentLine()
	}
}

// handleRuneInteractive processes a printable Unicode rune.
func (d *Dispatcher) handleRuneInteractive(r rune) {
	if d.shouldPassThroughInput() {
		var buf [utf8.UTFMax]byte
		n := utf8.EncodeRune(buf[:], r)
		d.ptyWrite(buf[:n])
		return
	}

	// Control characters
	if r >= 0 && r < 32 {
		return
	}

	// Printable character — add to buffer
	d.clearOnNextEmptyEnter = false
	d.linebuf.Append(r)

	// Echo via output channel to ensure proper ordering
	var buf [utf8.UTFMax]byte
	n := utf8.EncodeRune(buf[:], r)
	d.outputChan.WriteOp(render.OutputOp{Data: append([]byte(nil), buf[:n]...)})
	d.screenCol += buffer.RuneWidth(r)

	// Live-update suggestions on every input change.
	d.refreshSuggestions(d.linebuf.String())
}

// refreshSuggestions recomputes suggestions for the given text and renders
// or hides the popup accordingly. Called on every edit so suggestions stay
// live without needing Tab. Empty text hides the popup.
func (d *Dispatcher) refreshSuggestions(text string) {
	if text == "" {
		d.hideSuggestions()
		return
	}
	d.updateSuggestionsFromText(text)
}

// handleSubmitInteractive submits the current command to the PTY.
func (d *Dispatcher) handleSubmitInteractive(line *liner.State) {
	if d.showing && len(d.suggestions) > 0 && d.selected < len(d.suggestions) {
		s := d.suggestions[d.selected]
		if s.Kind == specs.KindInstall {
			d.popup.Erase(len(d.suggestions))
			d.showing = false
			d.suggestions = nil
			d.linebuf.SetString(s.Completion())
			d.redrawCurrentLine()
			return
		}
	}

	// Erase the popup area if visible; Enter always submits linebuf as-is,
	// without auto-accepting any suggestion.
	if d.showing {
		d.popup.Erase(len(d.suggestions))
		d.showing = false
		d.suggestions = nil
	}

	cmd := d.linebuf.String()
	if cmd == "" && d.clearOnNextEmptyEnter && !d.emulator.IsAltScreen() {
		d.outputChan.WriteOp(render.OutputOp{Kind: render.OutputOpClearCurrent})
		d.ptyWrite([]byte{12})
		d.clearOnNextEmptyEnter = false
		d.screenCol = 0
		return
	}

	if cmd != "" {
		line.AppendHistory(cmd)
		d.history = append(d.history, cmd)
	}
	d.historyPos = len(d.history)
	d.historySaved = ""

	d.ptyWriteString(cmd)
	d.ptyWrite([]byte{10})

	d.linebuf.Reset()
	d.screenCol = 0
	d.clearOnNextEmptyEnter = cmd != ""
}

// eraseInputDisplay removes the locally-echoed input characters from the
// current line using relative motion, preserving the shell-rendered prompt
// to its left. Resets screenCol to 0 (the column right after the prompt).
func (d *Dispatcher) eraseInputDisplay() {
	if d.screenCol <= 0 {
		return
	}
	out := fmt.Sprintf("\033[%dD\033[K", d.screenCol)
	d.outputChan.WriteOp(render.OutputOp{Data: []byte(out)})
	d.screenCol = 0
}

// writePrompt writes the prompt to the terminal
func (d *Dispatcher) writePrompt() {
	prompt := d.promptFunc()
	d.outputChan.WriteOp(render.OutputOp{Data: []byte(prompt)})
}

// clearLine clears the current line and positions cursor at beginning.
// Resets screenCol because the screen cursor is now at column 0.
func (d *Dispatcher) clearLine() {
	seq := "\033[2K\033[0G"
	d.outputChan.WriteOp(render.OutputOp{Data: []byte(seq)})
	d.screenCol = 0
}

// redrawCurrentLine redraws the current input line in place without touching
// the shell-rendered prompt to its left.
//
// Strategy: move the cursor left by d.screenCol (the tracked offset from
// input start), write the new content, erase any leftover from a previous
// longer render with \033[K, then move the cursor back to the logical
// position. Update d.screenCol to reflect the new cursor column.
func (d *Dispatcher) redrawCurrentLine() {
	out := make([]byte, 0, 32)

	if d.screenCol > 0 {
		out = append(out, []byte(fmt.Sprintf("\033[%dD", d.screenCol))...)
	}

	content := d.linebuf.String()
	if content != "" {
		out = append(out, content...)
	}

	out = append(out, []byte("\033[K")...)

	totalWidth := d.linebuf.DisplayWidth()
	curWidth := d.linebuf.CursorDisplayWidth()
	if totalWidth > curWidth {
		out = append(out, []byte(fmt.Sprintf("\033[%dD", totalWidth-curWidth))...)
	}

	d.outputChan.WriteOp(render.OutputOp{Data: out})
	d.screenCol = curWidth
}

// updateSuggestionsFromText updates suggestions based on current text.
func (d *Dispatcher) updateSuggestionsFromText(text string) {
	d.linebuf.SetString(text)
	d.linebuf.SetCursor(len(text))

	ctx := d.parser.GetCurrentContext(d.linebuf)
	d.suggestEngine.SetCWD(d.currentCWD)
	suggestions := d.suggestEngine.GetSuggestions(d.linebuf, &ctx)

	if len(suggestions) == 0 {
		d.hideSuggestions()
		return
	}

	d.suggestions = suggestions
	d.selected = 0
	d.showing = true
	d.popup.Render(suggestions, d.selected, 0)
}

// showSuggestionsFromCurrentLine shows suggestions based on current linebuf.
func (d *Dispatcher) showSuggestionsFromCurrentLine() {
	ctx := d.parser.GetCurrentContext(d.linebuf)
	d.suggestEngine.SetCWD(d.currentCWD)
	suggestions := d.suggestEngine.GetSuggestions(d.linebuf, &ctx)

	if len(suggestions) == 0 {
		return
	}

	d.suggestions = suggestions
	d.selected = 0
	d.showing = true
	d.popup.Render(suggestions, d.selected, 0)
}

// acceptSuggestionInteractive accepts the selected suggestion.
func (d *Dispatcher) acceptSuggestionInteractive() {
	if len(d.suggestions) == 0 || d.selected >= len(d.suggestions) {
		return
	}
	s := d.suggestions[d.selected]
	if s.Kind == specs.KindInstall {
		d.linebuf.SetString(s.Completion())
	} else {
		d.linebuf.AppendWord(s.Completion())
	}
	numLines := len(d.suggestions)
	d.hideSuggestions()
	d.popup.AcceptAndRedraw(d.linebuf.String(), numLines)
}

// promptFunc generates the prompt string.
func (d *Dispatcher) promptFunc() string {
	ps1 := os.Getenv("PS1")
	if ps1 == "" {
		return "$ "
	}
	return ps1
}

// historyPath returns the path to the history file.
func (d *Dispatcher) historyPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".goshell_history"
	}
	return filepath.Join(home, ".goshell_history")
}

// loadHistory reads history entries from the file into d.history for Up/Down navigation.
func (d *Dispatcher) loadHistory(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if entry := scanner.Text(); entry != "" {
			d.history = append(d.history, entry)
		}
	}
	d.historyPos = len(d.history)
}

// handleSubmit sends a command to the PTY for execution.
func (d *Dispatcher) handleSubmit(cmd string) {
	d.ptyWrite([]byte("\033[2K\033[0G"))
	d.ptyWriteString(cmd)
	d.ptyWrite([]byte{10})
	d.linebuf.Reset()
}
