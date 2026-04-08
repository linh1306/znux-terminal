package buffer

import (
	"strings"
	"unicode/utf8"
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

// Append adds runes at cursor position
func (l *LineBuf) Append(r rune) {
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

// MoveCursorLeft moves cursor left
func (l *LineBuf) MoveCursorLeft() bool {
	if l.cursor == 0 {
		return false
	}
	l.cursor--
	return true
}

// MoveCursorRight moves cursor right
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
	for l.cursor > 0 && l.buf[l.cursor-1] == ' ' {
		l.cursor--
	}
	for l.cursor > 0 && l.buf[l.cursor-1] != ' ' {
		l.cursor--
	}
}

// WordRight moves cursor to start of next word
func (l *LineBuf) WordRight() {
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
	}
}

// Len returns buffer length
func (l *LineBuf) Len() int {
	return len(l.buf)
}

// DisplayWidth returns the visual width of the buffer (for cursor positioning)
func (l *LineBuf) DisplayWidth() int {
	return utf8.RuneCountInString(l.String())
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
