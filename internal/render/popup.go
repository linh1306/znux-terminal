package render

import (
	"github.com/nguyenlinh13602/goshell/specs"
)

// Popup renders autocomplete suggestions using ANSI escape codes
type Popup struct {
	maxHeight int
	width     int
	output    OutputChan
}

// NewPopup creates a new popup renderer
func NewPopup(output OutputChan) *Popup {
	return &Popup{
		maxHeight: 10,
		width:     40,
		output:    output,
	}
}

// Render displays the suggestions popup at cursor position
func (p *Popup) Render(suggestions []specs.Suggestion, selected int, col int) {
	if len(suggestions) == 0 {
		return
	}

	// Build popup content
	content := p.buildPopupContent(suggestions, selected)

	// Send via output channel: save cursor, move down, write, restore cursor
	seq := make([]byte, 0, 256)
	seq = append(seq, "\033[s"...)       // Save cursor
	seq = append(seq, "\033[1B"...)      // Move down 1 line
	seq = append(seq, "\033[0G"...)      // Go to column 0
	seq = append(seq, content...)
	seq = append(seq, "\033[u"...)       // Restore cursor

	p.output.WriteOp(OutputOp{Data: seq})
}

// Erase removes the popup from screen
func (p *Popup) Erase(numLines int) {
	if numLines <= 0 {
		return
	}

	seq := make([]byte, 0, 128)
	seq = append(seq, "\033[s"...)  // Save cursor
	seq = append(seq, "\033[0G"...) // Go to column 0

	for i := 0; i < numLines; i++ {
		seq = append(seq, "\033[1B"...) // Move down 1 line
	}
	seq = append(seq, "\033[0G"...)  // Back to column 0
	seq = append(seq, "\033[J"...)    // Erase from cursor to end of screen
	seq = append(seq, "\033[u"...)    // Restore cursor

	p.output.WriteOp(OutputOp{Data: seq})
}

// buildPopupContent builds the formatted suggestion list
func (p *Popup) buildPopupContent(suggestions []specs.Suggestion, selected int) []byte {
	height := len(suggestions)
	if height > p.maxHeight {
		height = p.maxHeight
	}

	var result []byte
	for i := 0; i < height; i++ {
		if i > 0 {
			result = append(result, "\033[1B\033[0G"...) // down + home
		}

		name := suggestions[i].Name
		if len(name) > 18 {
			name = name[:18]
		}

		desc := suggestions[i].Description
		if len(desc) > 40 {
			desc = desc[:40]
		}

		padding := 18 - len(name)
		if padding < 0 {
			padding = 0
		}

		if i == selected {
			// Highlight selected
			result = append(result, "\033[7m"...) // Reverse video
			result = append(result, name...)
			for j := 0; j < padding; j++ {
				result = append(result, ' ')
			}
			result = append(result, "  "...)
			result = append(result, desc...)
			result = append(result, "\033[0m"...) // Clear attributes
		} else {
			result = append(result, name...)
			for j := 0; j < padding; j++ {
				result = append(result, ' ')
			}
			result = append(result, "  "...)
			result = append(result, desc...)
		}
	}

	// Clear rest of screen
	result = append(result, "\033[J"...)

	return result
}

// AcceptAndRedraw erases popup + input line, then reprints with accepted suggestion
func (p *Popup) AcceptAndRedraw(newLine string, numLines int) {
	seq := make([]byte, 0, 256)

	// Erase popup lines
	for i := 0; i < numLines; i++ {
		seq = append(seq, "\033[1B\033[2K"...) // down 1 line, clear line
	}

	// Erase the line where input cursor currently is
	seq = append(seq, "\033[2K"...)        // Clear entire current line
	seq = append(seq, "\033[0G"...)        // Go to column 0
	seq = append(seq, newLine...)          // Print new line
	seq = append(seq, "\033[0G"...)        // Home
	seq = append(seq, "\033[J"...)         // Clear to end of screen

	p.output.WriteOp(OutputOp{Data: seq})
}
