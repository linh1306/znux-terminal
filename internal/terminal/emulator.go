package terminal

import (
	"github.com/hinshun/vt10x"
)

// outputChan is used by cmd/main.go output goroutine
type OutputWriter interface {
	WriteOp(data []byte)
}

// Emulator wraps vt10x to track terminal state
type Emulator struct {
	term vt10x.Terminal
}

// NewEmulator creates a new terminal emulator state
func NewEmulator() *Emulator {
	return &Emulator{
		term: vt10x.New(),
	}
}

// Write processes a sequence of bytes through the emulator
func (e *Emulator) Write(b []byte) {
	e.term.Write(b)
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
