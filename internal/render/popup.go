package render

import (
	"fmt"

	"github.com/nguyenlinh13602/goshell/internal/suggest/specs"
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

// Render displays the suggestions popup BELOW the cursor, then restores cursor.
func (p *Popup) Render(suggestions []specs.Suggestion, selected int, col int) {
	if len(suggestions) == 0 {
		return
	}

	content := p.buildPopupContent(suggestions, selected)

	seq := make([]byte, 0, 256)
	seq = append(seq, "\033[s"...)  // Save cursor (at input line)
	seq = append(seq, "\033[1B"...) // Move down 1 line (to first popup line)
	seq = append(seq, content...)
	seq = append(seq, "\033[u"...) // Restore cursor to input line

	p.output.WriteOp(OutputOp{Data: seq})
}

// Erase removes the popup from screen. numLines is accepted for API
// compatibility but no longer used: we simply save the cursor on the input
// line, drop to the first popup row, erase to end of screen, then restore.
// This guarantees we never touch the input line or the prompt above it.
func (p *Popup) Erase(numLines int) {
	_ = numLines
	seq := make([]byte, 0, 32)
	seq = append(seq, "\033[s"...)  // Save cursor (on input line)
	seq = append(seq, "\033[1B"...) // Move down into popup area
	seq = append(seq, "\033[0G"...) // Column 0
	seq = append(seq, "\033[J"...)  // Erase to end of screen (popup only)
	seq = append(seq, "\033[u"...)  // Restore cursor to input line
	p.output.WriteOp(OutputOp{Data: seq})
}

// buildPopupContent builds the formatted suggestion list
func (p *Popup) buildPopupContent(suggestions []specs.Suggestion, selected int) []byte {
	height := len(suggestions)
	if height > p.maxHeight {
		height = p.maxHeight
	}

	var result []byte

	offset, height := visibleRange(len(suggestions), selected, height)

	// Find max visible name length for aligning the description column.
	maxNameLen := 0
	for i := 0; i < height; i++ {
		nameLen := runeLen(truncateRunes(suggestions[i+offset].Name, 18))
		if nameLen > maxNameLen {
			maxNameLen = nameLen
		}
	}

	for i := 0; i < height; i++ {
		if i == 0 {
			result = append(result, "\r\033[K"...) // Carriage return + clear to end of line
		} else {
			result = append(result, "\033[1B\r\033[K"...) // down + home + clear
		}

		name := truncateRunes(suggestions[i+offset].Name, 18)

		bullet := "○ "
		prefix := "│ "

		if i+offset == selected {
			bullet = "● "
		}

		// Build the line
		result = append(result, []byte(prefix)...)
		result = append(result, []byte(bullet)...)
		result = append(result, []byte(name)...)

		desc := suggestions[i+offset].Description
		if desc != "" {
			padding := maxNameLen - runeLen(name) + 2
			result = append(result, []byte(fmt.Sprintf("%*s", padding, ""))...)
			result = append(result, "\033[90m"...) // Gray description column
			result = append(result, []byte(desc)...)
			result = append(result, "\033[0m"...)
		}
	}

	// Clear rest of screen
	result = append(result, "\033[J"...)

	// Show count header at the bottom when more than 5 options
	if len(suggestions) > 5 {
		result = append(result, "\033[1B\r\033[K"...) // down + home + clear
		result = append(result, "\033[1;36m"...)      // Cyan color
		result = append(result, "◆ - "...)
		result = append(result, "\033[1m"...) // Bold
		result = append(result, []byte(fmt.Sprintf("%d/%d quy tắc", selected+1, len(suggestions)))...)
		result = append(result, "\033[0m"...) // Reset attributes
	}

	return result
}

func visibleRange(total, selected, height int) (offset, visible int) {
	if total <= 0 || height <= 0 {
		return 0, 0
	}
	if height > total {
		height = total
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= total {
		selected = total - 1
	}

	offset = selected - 1
	if offset < 0 {
		offset = 0
	}
	if maxOffset := total - height; offset > maxOffset {
		offset = maxOffset
	}
	return offset, height
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}

	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func runeLen(s string) int {
	return len([]rune(s))
}

// AcceptAndRedraw erases popup + input line, then reprints with accepted suggestion.
// Called after hideSuggestions (cursor is on input line).
func (p *Popup) AcceptAndRedraw(newLine string, numLines int) {
	seq := make([]byte, 0, 256)

	// Save cursor on input line, move down past popup, move back up to
	// input line (but now at column 0), erase popup + cleanup.
	seq = append(seq, "\033[s"...) // Save cursor (on input line)

	// Move down past the popup to the bottom of the popup region.
	seq = append(seq, "\033["...)
	seq = append(seq, fmt.Sprintf("%d", numLines)...)
	seq = append(seq, "B"...) // down numLines lines

	// Move up past popup + 1 (back to input line), go to col 0.
	seq = append(seq, "\033["...)
	seq = append(seq, fmt.Sprintf("%d", numLines+1)...)
	seq = append(seq, "A"...) // up numLines+1 lines

	// Erase popup area, then print new input line, clear to end of screen.
	seq = append(seq, "\033[0G"...) // column 0
	seq = append(seq, "\033[J"...)  // erase popup below
	seq = append(seq, newLine...)   // print new input line

	seq = append(seq, "\033[u"...) // restore cursor to input line

	p.output.WriteOp(OutputOp{Data: seq})
}
