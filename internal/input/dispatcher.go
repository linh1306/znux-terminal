package input

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/nguyenlinh13602/goshell/internal/buffer"
	"github.com/nguyenlinh13602/goshell/internal/config"
	"github.com/nguyenlinh13602/goshell/internal/render"
	"github.com/nguyenlinh13602/goshell/internal/suggest"
	"github.com/nguyenlinh13602/goshell/internal/terminal"
	"github.com/nguyenlinh13602/goshell/internal/suggest/specs"
)

// Dispatcher manages the input loop and coordinates between PTY and suggest engine.
// Line editing is handled by liner (in liner_input.go).
// This struct retains suggestion engine, popup rendering, and PTY write coordination.
type Dispatcher struct {
	ptyOut      *os.File
	ptyMu       *sync.Mutex // protects PTY writes from concurrent resize
	emulator    *terminal.Emulator
	outputChan  render.OutputChan
	config      *config.Config
	done        chan struct{}

	// Line editing state (used by liner_input.go)
	linebuf   *buffer.LineBuf
	parser    *buffer.Parser
	runeBuf   []byte // UTF-8 byte accumulator for multi-byte characters

	// Suggestion support
	suggestEngine *suggest.Engine
	popup          *render.Popup
	suggestions    []specs.Suggestion
	selected       int
	showing        bool

	// Bracketed paste state
	pasteActive  bool
	pasteBuffer  strings.Builder

	// History navigation state
	history      []string
	historyPos   int
	historySaved string // current input saved when navigating history

	// screenCol tracks the screen cursor column relative to input start
	// (i.e., the point after the shell prompt). Used to perform relative
	// cursor motions when redrawing without erasing the shell-rendered prompt.
	screenCol int

	// currentCWD tracks the shell's current working directory via OSC 6973
	currentCWD string
}

// NewDispatcher creates a new input dispatcher
func NewDispatcher(ptyOut *os.File, emulator *terminal.Emulator, output render.OutputChan, ptyMu *sync.Mutex) *Dispatcher {
	d := &Dispatcher{
		ptyOut:        ptyOut,
		ptyMu:         ptyMu,
		emulator:      emulator,
		outputChan:    output,
		config:        config.DefaultConfig(),
		linebuf:       buffer.NewLineBuf(),
		parser:        buffer.NewParser(),
		suggestEngine: suggest.NewEngine(),
		popup:         render.NewPopup(output),
		suggestions:   nil,
		selected:      0,
		showing:       false,
		done:          make(chan struct{}),
	}
	emulator.SetOSCHandler(d)
	return d
}

// OnCWD implements terminal.OSCHandler, called when shell sends OSC 6973 CWD
func (d *Dispatcher) OnCWD(path string) {
	d.currentCWD = path
}

// GetCWD returns the shell's current working directory
func (d *Dispatcher) GetCWD() string {
	return d.currentCWD
}

// SetConfig sets the configuration (keybindings, theme)
func (d *Dispatcher) SetConfig(cfg *config.Config) {
	d.config = cfg
}

// Stop signals the input loop to exit, used when the underlying shell process exits.
func (d *Dispatcher) Stop() {
	select {
	case <-d.done:
	default:
		close(d.done)
	}
}

// ptyWrite writes to the PTY with mutex protection against concurrent resize.
func (d *Dispatcher) ptyWrite(data []byte) {
	d.ptyMu.Lock()
	_, err := d.ptyOut.Write(data)
	d.ptyMu.Unlock()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ptyWrite: %v\n", err)
	}
}

// ptyWriteString writes a string to the PTY.
func (d *Dispatcher) ptyWriteString(s string) {
	d.ptyWrite([]byte(s))
}

// IsAltScreen returns true if the emulator is in alt screen mode.
func (d *Dispatcher) IsAltScreen() bool {
	return d.emulator.IsAltScreen()
}

// hideSuggestions hides the suggestions popup
func (d *Dispatcher) hideSuggestions() {
	if d.showing {
		d.popup.Erase(len(d.suggestions))
		d.showing = false
		d.suggestions = nil
		d.redrawInputLine()
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
		d.redrawInputLine()
	}
}

// nextSuggestion cycles to the next suggestion
func (d *Dispatcher) nextSuggestion() {
	if len(d.suggestions) == 0 {
		return
	}
	d.selected = (d.selected + 1) % len(d.suggestions)
	d.popup.Render(d.suggestions, d.selected, 0)
	d.redrawInputLine()
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
	d.redrawInputLine()
}

// redrawInputLine redraws the current input line on the terminal using
// relative cursor motion, so the shell-rendered prompt (e.g. starship) to
// the left of the input is preserved.
func (d *Dispatcher) redrawInputLine() {
	d.redrawCurrentLine()
}

// GetSuggestions returns suggestions for the given text using the suggestion engine.
func (d *Dispatcher) GetSuggestions(text string) []specs.Suggestion {
	d.linebuf.SetString(text)
	d.linebuf.SetCursor(len(text))
	ctx := d.parser.GetCurrentContext(d.linebuf)
	return d.suggestEngine.GetSuggestions(d.linebuf, &ctx)
}

// ShowSuggestions shows suggestions for the given text.
func (d *Dispatcher) ShowSuggestions(text string) {
	d.linebuf.SetString(text)
	d.linebuf.SetCursor(len(text))
	d.showSuggestions()
}

// HideSuggestions hides the current suggestions popup.
func (d *Dispatcher) HideSuggestions() {
	d.hideSuggestions()
}

// NextSuggestion cycles to the next suggestion.
func (d *Dispatcher) NextSuggestion() {
	d.nextSuggestion()
}

// PrevSuggestion cycles to the previous suggestion.
func (d *Dispatcher) PrevSuggestion() {
	d.prevSuggestion()
}

// AcceptSuggestion accepts the currently selected suggestion by replacing
// the partial word at the end of the buffer with the suggestion's name.
func (d *Dispatcher) AcceptSuggestion() string {
	if len(d.suggestions) == 0 || !d.showing || d.selected >= len(d.suggestions) {
		return d.linebuf.String()
	}
	s := d.suggestions[d.selected]
	d.linebuf.ReplaceLastWord(s.Name)
	numLines := len(d.suggestions)
	d.hideSuggestions()
	d.popup.AcceptAndRedraw(d.linebuf.String(), numLines)
	return d.linebuf.String()
}

// SelectedSuggestion returns the currently selected suggestion text.
func (d *Dispatcher) SelectedSuggestion() string {
	if len(d.suggestions) == 0 || d.selected >= len(d.suggestions) {
		return ""
	}
	return d.suggestions[d.selected].Name
}

// IsShowingSuggestions returns true if the suggestions popup is showing.
func (d *Dispatcher) IsShowingSuggestions() bool {
	return d.showing
}

// LineBuf returns the line buffer.
func (d *Dispatcher) LineBuf() *buffer.LineBuf {
	return d.linebuf
}
