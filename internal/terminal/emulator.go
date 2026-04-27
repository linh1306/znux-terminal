package terminal

import (
	"strings"

	"github.com/hinshun/vt10x"
)

// OSCHandler handles events detected by the emulator
type OSCHandler interface {
	OnCWD(path string)
}

// Emulator wraps vt10x to track terminal state
type Emulator struct {
	term   vt10x.Terminal
	handler OSCHandler
}

// outputChan is used by cmd/main.go output goroutine
type OutputWriter interface {
	WriteOp(data []byte)
}

// NewEmulator creates a new terminal emulator state
func NewEmulator() *Emulator {
	return &Emulator{
		term:   vt10x.New(),
		handler: nil,
	}
}

// SetOSCHandler sets the handler for OSC events
func (e *Emulator) SetOSCHandler(h OSCHandler) {
	e.handler = h
}

// Write processes a sequence of bytes through the emulator
func (e *Emulator) Write(b []byte) {
	// Check for OSC sequences before passing to vt10x
	if e.handler != nil {
		e.parseOSC(b)
	}
	e.term.Write(b)
}

// OSC 6973: CWD notification
// Format: ESC ] 6973 ; CWD ; <path> BEL (or ESC \)
// Recognized terminators: BEL (\a), ESC \ (\x1b\\)
func (e *Emulator) parseOSC(b []byte) {
	osc6973 := []byte{0x1b, 0x5d, 0x36, 0x39, 0x37, 0x33, 0x3b} // \e]6973;
	idx := 0
	for idx < len(b) {
		i := indexBytes(b[idx:], osc6973)
		if i == -1 {
			return
		}
		idx += i

		// Found \e]6973; - now parse the rest
		rest := b[idx+len(osc6973):]
		if len(rest) < 3 {
			return
		}

		// Expect "CWD;" prefix
		if !strings.HasPrefix(string(rest), "CWD;") {
			idx += len(osc6973)
			continue
		}
		rest = rest[4:] // skip "CWD;"

		// Find terminator: BEL (\a) or ESC \ (\x1b\)
		var pathEnd int
		var found bool
		for pathEnd = 0; pathEnd < len(rest); pathEnd++ {
			if rest[pathEnd] == 0x07 { // BEL
				found = true
				break
			}
			if pathEnd+1 < len(rest) && rest[pathEnd] == 0x1b && rest[pathEnd+1] == 0x5c { // ESC \
				found = true
				pathEnd++ // include ESC in path (won't matter)
				break
			}
		}
		if !found {
			return
		}

		path := string(rest[:pathEnd])
		if path != "" {
			e.handler.OnCWD(path)
		}
		idx += len(osc6973) + 4 + pathEnd + 1
	}
}

// indexBytes finds the first occurrence of pattern in b
func indexBytes(b, pattern []byte) int {
	for i := 0; i <= len(b)-len(pattern); i++ {
		if string(b[i:i+len(pattern)]) == string(pattern) {
			return i
		}
	}
	return -1
}

// IsAltScreen returns true if the terminal is in alternate screen mode
func (e *Emulator) IsAltScreen() bool {
	return e.term.Mode()&vt10x.ModeAltScreen != 0
}

// Cursor returns the current cursor position
func (e *Emulator) Cursor() (row, col int) {
	cur := e.term.Cursor()
	return cur.Y, cur.X
}
