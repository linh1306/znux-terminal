package input

import (
	"bufio"
	"io"
	"os"
	"unicode/utf8"

	"github.com/nguyenlinh13602/goshell/internal/buffer"
	"github.com/nguyenlinh13602/goshell/internal/render"
	"github.com/nguyenlinh13602/goshell/internal/suggest"
	"github.com/nguyenlinh13602/goshell/internal/terminal"
	"github.com/nguyenlinh13602/goshell/specs"
)

// Dispatcher manages the input loop and coordinates between PTY and suggest engine
type Dispatcher struct {
	stdin       *os.File
	ptyOut      *os.File
	emulator    *terminal.Emulator
	reader      *bufio.Reader
	acc         *RuneAccumulator
	quit        chan struct{}
	outputChan  render.OutputChan

	// Suggestion support
	linebuf       *buffer.LineBuf
	parser       *buffer.Parser
	suggestEngine *suggest.Engine
	popup        *render.Popup
	suggestions  []specs.Suggestion
	selected     int
	showing      bool
}

// NewDispatcher creates a new input dispatcher
func NewDispatcher(stdin *os.File, ptyOut *os.File, emulator *terminal.Emulator, output render.OutputChan) *Dispatcher {
	return &Dispatcher{
		stdin:        stdin,
		ptyOut:       ptyOut,
		emulator:     emulator,
		reader:       bufio.NewReaderSize(stdin, 8192),
		acc:          NewRuneAccumulator(),
		quit:         make(chan struct{}),
		outputChan:   output,
		linebuf:      buffer.NewLineBuf(),
		parser:       buffer.NewParser(),
		suggestEngine: suggest.NewEngine(),
		popup:        render.NewPopup(output),
		suggestions:  nil,
		selected:     0,
		showing:      false,
	}
}

// Run starts the input loop
func (d *Dispatcher) Run() error {
	for {
		b, err := d.reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if b == 27 {
			// Escape — could be arrow key or other escape sequence
			seq := []byte{27}

			// Read more bytes to form complete escape sequence
			// Try to read up to 4 more bytes (max for most escape sequences)
			for len(seq) < 5 {
				// Check if more bytes are available
				n := d.reader.Buffered()
				if n == 0 {
					// No more bytes available — could be plain Escape key
					break
				}
				next, err := d.reader.ReadByte()
				if err != nil {
					break
				}
				seq = append(seq, next)

				// Check if this completes a known sequence
				if d.isCompleteEscapeSeq(seq) {
					break
				}
				// Stop if we've read enough and no sequence matched
				if len(seq) >= 4 {
					break
				}
			}

			d.handleEscapeSeq(seq)
			continue
		}

		runes, valid := d.acc.Feed(b)
		if !valid {
			continue
		}

		for _, r := range runes {
			d.handleRune(r)
		}
	}
}

// isCompleteEscapeSeq checks if an escape sequence is complete
func (d *Dispatcher) isCompleteEscapeSeq(seq []byte) bool {
	if len(seq) < 2 {
		return false
	}
	if seq[1] == '[' {
		// CSI sequences: ESC [ <final byte>
		// Final bytes are typically letters or ~ for function keys
		if len(seq) >= 3 {
			b := seq[2]
			// Letters (A-Z, a-z) for arrow keys, home/end, etc.
			if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') {
				return true
			}
			// ~ sequences: Home(1), End(4), PgUp(5), PgDn(6), Delete(3)
			if b == '~' {
				return true
			}
			// 2-byte SS3 sequences: ESC O <A-D>
			if b == 'O' && len(seq) >= 3 {
				return true
			}
		}
		// Extended: ESC [ 1 ~ (Home) etc
		if len(seq) >= 4 && seq[2] == '1' && seq[3] == '~' {
			return true
		}
	}
	if seq[1] == 'O' {
		// SS3 sequences: ESC O <A-D>
		if len(seq) >= 3 {
			b := seq[2]
			if b >= 'A' && b <= 'D' {
				return true
			}
		}
	}
	return false
}

// handleEscapeSeq processes ANSI escape sequences (arrow keys, etc.)
func (d *Dispatcher) handleEscapeSeq(seq []byte) {
	// Arrow keys when popup is showing
	if d.showing && len(seq) >= 3 && seq[0] == 27 && seq[1] == '[' {
		switch seq[2] {
		case 'A': // Arrow Up
			d.prevSuggestion()
			return
		case 'B': // Arrow Down
			d.nextSuggestion()
			return
		case 'C': // Arrow Right — accept suggestion
			d.acceptSuggestion()
			return
		case 'D': // Arrow Left — dismiss popup
			d.hideSuggestions()
			return
		}
	}

	// SS3 sequences (some terminals use ESC O instead of ESC [)
	if d.showing && len(seq) >= 3 && seq[0] == 27 && seq[1] == 'O' {
		switch seq[2] {
		case 'A': // Arrow Up
			d.prevSuggestion()
			return
		case 'B': // Arrow Down
			d.nextSuggestion()
			return
		case 'C': // Arrow Right
			d.acceptSuggestion()
			return
		case 'D': // Arrow Left
			d.hideSuggestions()
			return
		}
	}

	// Plain Escape key — dismiss popup
	if len(seq) == 1 {
		if d.showing {
			d.hideSuggestions()
			return
		}
	}

	// Unknown escape sequence — pass through to PTY
	for _, b := range seq {
		d.ptyOut.Write([]byte{b})
	}
}

func (d *Dispatcher) handleRune(r rune) {
	// Pass through when alt screen is active (vim, htop, etc.)
	if d.emulator.IsAltScreen() {
		var buf [utf8.UTFMax]byte
		n := utf8.EncodeRune(buf[:], r)
		d.ptyOut.Write(buf[:n])
		return
	}

	switch r {
	case 3: // Ctrl+C
		if d.showing {
			d.hideSuggestions()
		}
		d.linebuf.Reset()
		d.ptyOut.Write([]byte{3})

	case 4: // Ctrl+D
		d.ptyOut.Write([]byte{4})

	case 26: // Ctrl+Z
		d.ptyOut.Write([]byte{26})

	case 9: // Tab — next suggestion or show popup
		if d.showing {
			d.nextSuggestion()
		} else {
			d.showSuggestions()
		}

	case 10, 13: // Enter
		if d.showing {
			// Update linebuf with selected suggestion first
			s := d.suggestions[d.selected]
			d.linebuf.AppendWord(s.Name)

			// Erase current line completely
			d.ptyOut.Write([]byte("\033[2K\033[0G"))

			// Send full command to PTY
			d.ptyOut.WriteString(d.linebuf.String())
			d.ptyOut.Write([]byte{10}) // newline to execute

			d.linebuf.Reset()
			d.hideSuggestions()
		} else {
			d.ptyOut.Write([]byte{10})
			d.linebuf.Reset()
		}

	case 127, 8: // Backspace
		if d.showing {
			d.linebuf.Delete()
			d.updateSuggestions()
		} else {
			if d.linebuf.Len() > 0 {
				d.linebuf.Delete()
				var buf [utf8.UTFMax]byte
				n := utf8.EncodeRune(buf[:], 127)
				d.ptyOut.Write(buf[:n])
			}
		}

	default:
		d.linebuf.Append(r)
		var buf [utf8.UTFMax]byte
		n := utf8.EncodeRune(buf[:], r)
		d.ptyOut.Write(buf[:n])

		if d.showing {
			d.updateSuggestions()
		}
	}
}

// showSuggestions displays the autocomplete popup
func (d *Dispatcher) showSuggestions() {
	ctx := d.parser.GetCurrentContext(d.linebuf)
	d.suggestions = d.suggestEngine.GetSuggestions(d.linebuf, &ctx)
	if len(d.suggestions) > 0 {
		d.selected = 0
		d.showing = true
		d.popup.Render(d.suggestions, d.selected, 0)
	}
}

// updateSuggestions refreshes suggestions after input change
func (d *Dispatcher) updateSuggestions() {
	d.hideSuggestions()
	d.showSuggestions()
}

// nextSuggestion cycles to the next suggestion
func (d *Dispatcher) nextSuggestion() {
	if len(d.suggestions) == 0 {
		return
	}
	d.selected = (d.selected + 1) % len(d.suggestions)
	d.popup.Render(d.suggestions, d.selected, 0)
}

// prevSuggestion cycles to the previous suggestion
func (d *Dispatcher) prevSuggestion() {
	if len(d.suggestions) == 0 {
		return
	}
	d.selected = d.selected - 1
	if d.selected < 0 {
		d.selected = len(d.suggestions) - 1
	}
	d.popup.Render(d.suggestions, d.selected, 0)
}

// acceptSuggestion inserts the selected suggestion and redraws the line
func (d *Dispatcher) acceptSuggestion() {
	if len(d.suggestions) == 0 || !d.showing {
		return
	}

	s := d.suggestions[d.selected]
	d.linebuf.AppendWord(s.Name)
	numLines := len(d.suggestions)
	d.hideSuggestions()
	d.popup.AcceptAndRedraw(d.linebuf.String(), numLines)
}

// hideSuggestions hides the suggestions popup
func (d *Dispatcher) hideSuggestions() {
	if d.showing {
		d.popup.Erase(len(d.suggestions))
		d.showing = false
		d.suggestions = nil
	}
}

// Stop signals the dispatcher to stop
func (d *Dispatcher) Stop() {
	close(d.quit)
}

// RuneAccumulator accumulates raw bytes into Unicode runes
type RuneAccumulator struct {
	buffer []byte
}

// NewRuneAccumulator creates a new rune accumulator
func NewRuneAccumulator() *RuneAccumulator {
	return &RuneAccumulator{}
}

// Feed adds a byte and returns complete runes if any
func (a *RuneAccumulator) Feed(b byte) (runes []rune, valid bool) {
	a.buffer = append(a.buffer, b)

	r, size := utf8.DecodeRune(a.buffer)
	if r == utf8.RuneError && size == 1 && len(a.buffer) < 4 {
		return nil, false
	}

	runes = []rune{r}
	a.buffer = a.buffer[:0]
	return runes, true
}

// Reset clears the buffer
func (a *RuneAccumulator) Reset() {
	a.buffer = a.buffer[:0]
}

// KeyType represents the classification of a key press
type KeyType int

const (
	KeyNormal KeyType = iota
	KeyCtrlC
	KeyCtrlD
	KeyCtrlZ
	KeyTab
	KeyEnter
	KeyEscape
	KeyBackspace
	KeyDelete
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown
	KeyCtrlU
	KeyCtrlK
	KeyCtrlW
	KeyUnknown
)
