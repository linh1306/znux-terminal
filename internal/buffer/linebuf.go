package buffer

import (
	"strings"
)

// LineBuf holds the current input line with cursor position
type LineBuf struct {
	buf   []rune
	cursor int
}

// NewLineBuf creates a new empty line buffer
func NewLineBuf() *LineBuf {
	return &LineBuf{
		buf:   []rune{},
		cursor: 0,
	}
}

// normalizeCursor is a no-op: buf is []rune so cursor is always rune-aligned.
func (l *LineBuf) normalizeCursor() {}

// RuneWidth returns the terminal display width of a rune.
// Returns 0 for non-printable control characters, 2 for wide characters (CJK, emoji).
func RuneWidth(r rune) int {
	switch {
	case r < 32 || r == 127:
		return 0
	case r >= 0x1100 &&
		(r <= 0x115F || // Hangul Jamo
		r == 0x2329 || r == 0x232A ||
		r >= 0x2E80 && r <= 0x303F || // CJK Radicals
		r >= 0x3040 && r <= 0xA4CF || // CJK
		r >= 0xAC00 && r <= 0xD7A3 || // Hangul Syllables
		r >= 0xF900 && r <= 0xFAFF || // CJK Compatibility Ideographs
		r >= 0xFE10 && r <= 0xFE1F || // Vertical forms
		r >= 0xFE30 && r <= 0xFE6F || // CJK Compatibility Forms
		r >= 0xFF00 && r <= 0xFF60 || // Fullwidth forms
		r >= 0xFFE0 && r <= 0xFFE6 ||
		r >= 0x20000 && r <= 0x2FFFD ||
		r >= 0x30000 && r <= 0x3FFFD):
		return 2
	default:
		return 1
	}
}

// Append adds a rune at cursor position
func (l *LineBuf) Append(r rune) {
	l.normalizeCursor()
	if l.cursor == len(l.buf) {
		l.buf = append(l.buf, r)
	} else {
		l.buf = append(l.buf[:l.cursor+1], l.buf[l.cursor:]...)
		l.buf[l.cursor] = r
	}
	l.cursor++
}

// Insert inserts runes at cursor position
func (l *LineBuf) Insert(rs []rune) {
	l.normalizeCursor()
	if l.cursor == len(l.buf) {
		l.buf = append(l.buf, rs...)
	} else {
		l.buf = append(l.buf[:l.cursor], append(rs, l.buf[l.cursor:]...)...)
	}
	l.cursor += len(rs)
}

// Delete removes a character before cursor (backspace)
func (l *LineBuf) Delete() bool {
	if l.cursor == 0 {
		return false
	}
	// Move cursor to rune boundary before deleting
	l.normalizeCursor()
	if l.cursor == 0 {
		return false
	}
	copy(l.buf[l.cursor-1:], l.buf[l.cursor:])
	l.buf = l.buf[:len(l.buf)-1]
	l.cursor--
	return true
}

// DeleteAfter removes character after cursor (delete key)
func (l *LineBuf) DeleteAfter() bool {
	if l.cursor >= len(l.buf) {
		return false
	}
	copy(l.buf[l.cursor:], l.buf[l.cursor+1:])
	l.buf = l.buf[:len(l.buf)-1]
	return true
}

// MoveCursorLeft moves cursor left by one rune
func (l *LineBuf) MoveCursorLeft() bool {
	if l.cursor == 0 {
		return false
	}
	l.cursor--
	l.normalizeCursor()
	return true
}

// MoveCursorRight moves cursor right by one rune
func (l *LineBuf) MoveCursorRight() bool {
	if l.cursor >= len(l.buf) {
		return false
	}
	l.cursor++
	return true
}

// MoveCursorToStart moves cursor to start
func (l *LineBuf) MoveCursorToStart() {
	l.cursor = 0
}

// MoveCursorToEnd moves cursor to end
func (l *LineBuf) MoveCursorToEnd() {
	l.cursor = len(l.buf)
}

// WordLeft moves cursor to start of previous word
func (l *LineBuf) WordLeft() {
	l.normalizeCursor()
	for l.cursor > 0 && l.buf[l.cursor-1] == ' ' {
		l.cursor--
	}
	for l.cursor > 0 && l.buf[l.cursor-1] != ' ' {
		l.cursor--
		l.normalizeCursor()
	}
}

// WordRight moves cursor to start of next word
func (l *LineBuf) WordRight() {
	l.normalizeCursor()
	for l.cursor < len(l.buf) && l.buf[l.cursor] != ' ' {
		l.cursor++
	}
	for l.cursor < len(l.buf) && l.buf[l.cursor] == ' ' {
		l.cursor++
	}
}

// DeleteToStart deletes from cursor to start of line
func (l *LineBuf) DeleteToStart() []rune {
	if l.cursor == 0 {
		return nil
	}
	deleted := make([]rune, l.cursor)
	copy(deleted, l.buf[:l.cursor])
	l.buf = l.buf[l.cursor:]
	l.cursor = 0
	return deleted
}

// DeleteToEnd deletes from cursor to end of line
func (l *LineBuf) DeleteToEnd() []rune {
	if l.cursor >= len(l.buf) {
		return nil
	}
	deleted := make([]rune, len(l.buf)-l.cursor)
	copy(deleted, l.buf[l.cursor:])
	l.buf = l.buf[:l.cursor]
	return deleted
}

// DeleteWord deletes word before cursor
func (l *LineBuf) DeleteWord() []rune {
	start := l.cursor
	l.WordLeft()
	if l.cursor == start {
		return nil
	}
	deleted := make([]rune, start-l.cursor)
	copy(deleted, l.buf[l.cursor:start])
	copy(l.buf[l.cursor:], l.buf[start:])
	l.buf = l.buf[:len(l.buf)-(start-l.cursor)]
	return deleted
}

// Reset clears the buffer
func (l *LineBuf) Reset() {
	l.buf = l.buf[:0]
	l.cursor = 0
}

// String returns the buffer content as string
func (l *LineBuf) String() string {
	return string(l.buf)
}

// Runes returns the buffer content as runes
func (l *LineBuf) Runes() []rune {
	return l.buf
}

// Cursor returns current cursor position
func (l *LineBuf) Cursor() int {
	return l.cursor
}

// SetCursor sets cursor position
func (l *LineBuf) SetCursor(pos int) {
	if pos < 0 {
		l.cursor = 0
	} else if pos > len(l.buf) {
		l.cursor = len(l.buf)
	} else {
		l.cursor = pos
		l.normalizeCursor()
	}
}

// Len returns buffer length
func (l *LineBuf) Len() int {
	return len(l.buf)
}

// DisplayWidth returns the terminal visual width of the buffer content.
// This accounts for double-width characters (CJK, emoji, Vietnamese diacritics).
func (l *LineBuf) DisplayWidth() int {
	w := 0
	for _, r := range l.buf {
		w += RuneWidth(r)
	}
	return w
}

// CursorDisplayWidth returns the terminal visual width from start to cursor position.
func (l *LineBuf) CursorDisplayWidth() int {
	w := 0
	for i := 0; i < l.cursor && i < len(l.buf); i++ {
		w += RuneWidth(l.buf[i])
	}
	return w
}

// TruncateFrom truncates from position to end
func (l *LineBuf) TruncateFrom(pos int) {
	if pos < 0 || pos > len(l.buf) {
		return
	}
	l.buf = l.buf[:pos]
	if l.cursor > pos {
		l.cursor = pos
	}
}

// SetString sets buffer from string
func (l *LineBuf) SetString(s string) {
	l.buf = []rune(s)
	l.cursor = len(l.buf)
}

// GetWordAtCursor returns the word at cursor position
func (l *LineBuf) GetWordAtCursor() (start, end int) {
	// Find start of current word (looking backward from cursor)
	start = l.cursor
	for start > 0 && isWordChar(l.buf[start-1]) {
		start--
	}
	// Find end of current word (looking forward from cursor)
	end = l.cursor
	for end < len(l.buf) && isWordChar(l.buf[end]) {
		end++
	}
	return
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.'
}

// ParseCommand parses the buffer and returns cmd structure
func ParseCommand(buf *LineBuf) *ParsedCommand {
	text := buf.String()
	if text == "" {
		return &ParsedCommand{}
	}

	// Simple parser: split by spaces, tracking position
	// Note: This is a basic implementation. Phase 2 will have a more robust parser.
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return &ParsedCommand{}
	}

	cmd := &ParsedCommand{
		Command: parts[0],
	}

	// Simple flag detection (starts with -)
	for i, p := range parts[1:] {
		if strings.HasPrefix(p, "-") {
			cmd.Flags = append(cmd.Flags, p)
		} else if cmd.Subcommand == "" && i < len(parts)-2 {
			cmd.Subcommand = p
		} else {
			cmd.Args = append(cmd.Args, p)
		}
	}

	return cmd
}

// ParsedCommand represents a parsed command
type ParsedCommand struct {
	Command    string
	Subcommand string
	Flags      []string
	Args       []string
}

// ReplaceLastWord replaces the trailing whitespace-delimited word with `word`.
// If the buffer ends with whitespace (or is empty), `word` is appended.
// Cursor is moved to the end of the buffer.
func (l *LineBuf) ReplaceLastWord(word string) {
	i := len(l.buf) - 1
	for i >= 0 && l.buf[i] != ' ' && l.buf[i] != '\t' {
		i--
	}
	kept := l.buf[:i+1]
	newBuf := make([]rune, 0, len(kept)+len([]rune(word)))
	newBuf = append(newBuf, kept...)
	newBuf = append(newBuf, []rune(word)...)
	l.buf = newBuf
	l.cursor = len(l.buf)
}

// AppendWord appends a word at the end (after space if needed)
func (l *LineBuf) AppendWord(word string) {
	// Add space if buffer doesn't end with space and is not empty
	if l.Len() > 0 {
		last := l.buf[l.Len()-1]
		if last != ' ' && last != '\t' {
			l.Append(' ')
		}
	}
	// Append each rune of the word
	for _, r := range word {
		l.Append(r)
	}
}
