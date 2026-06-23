package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

func Input(in *os.File, out io.Writer, title string, value string) (string, error) {
	if !term.IsTerminal(int(in.Fd())) {
		return "", errors.New("interactive terminal required")
	}

	oldState, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return "", err
	}
	defer term.Restore(int(in.Fd()), oldState)

	reader := bufio.NewReader(in)
	current := value
	renderedLines := 0

	render := func() error {
		clearRendered(out, renderedLines)
		renderedLines = 2
		_, err := fmt.Fprintf(out, "%s◇%s %s\r\n│ > %s", promptColor, resetColor, title, current)
		return err
	}

	if err := render(); err != nil {
		return "", err
	}

	for {
		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}

		switch b {
		case keyCtrlC:
			clearRendered(out, renderedLines)
			return "", errors.New("cancelled")
		case keyEnter, keyReturn:
			clearRendered(out, renderedLines)
			printResult(out, title, []string{current})
			return current, nil
		case keyBackspace, keyDelete:
			current = trimLastRune(current)
		case keyEsc:
			clearRendered(out, renderedLines)
			return "", nil
		default:
			if b >= 32 && b != 127 {
				current += string(rune(b))
			}
		}

		if err := render(); err != nil {
			return "", err
		}
	}
}

func trimLastRune(value string) string {
	if value == "" {
		return ""
	}
	runes := []rune(value)
	return string(runes[:len(runes)-1])
}

func clearRendered(out io.Writer, lines int) {
	if lines <= 0 {
		return
	}
	_, _ = fmt.Fprintf(out, "\r\033[%dA\033[J", lines-1)
}

func printResult(out io.Writer, title string, values []string) {
	fmt.Fprintf(out, "%s◆%s %s\r\n", promptColor, resetColor, title)
	if len(values) == 0 {
		fmt.Fprint(out, "│\r\n")
	} else {
		for _, value := range values {
			fmt.Fprintf(out, "│ %s\r\n", value)
		}
	}
	fmt.Fprint(out, "│\r\n")
}

func normalizeDefaultIndex(length int, index int) int {
	if length == 0 {
		return 0
	}
	if index < 0 {
		return 0
	}
	if index >= length {
		return length - 1
	}
	return index
}

func normalizeDefaultIndexes(length int, indexes []int) []bool {
	marked := make([]bool, length)
	for _, index := range indexes {
		if index >= 0 && index < length {
			marked[index] = true
		}
	}
	return marked
}

func selectedValues(items []string, marked []bool) []string {
	values := make([]string, 0)
	for i, item := range items {
		if marked[i] {
			values = append(values, item)
		}
	}
	return values
}

func readEscapeSeq(reader *bufio.Reader) []byte {
	first, err := reader.ReadByte()
	if err != nil || first != '[' {
		return nil
	}
	second, err := reader.ReadByte()
	if err != nil {
		return nil
	}
	return []byte{first, second}
}

func moveCursor(cursor int, length int, delta int) int {
	if length == 0 {
		return 0
	}
	cursor += delta
	if cursor < 0 {
		return length - 1
	}
	if cursor >= length {
		return 0
	}
	return cursor
}

func handleMoveKey(cursor int, length int, key byte) int {
	switch key {
	case 'j':
		return moveCursor(cursor, length, 1)
	case 'k':
		return moveCursor(cursor, length, -1)
	default:
		return cursor
	}
}

func handleEscapeMove(cursor int, length int, seq []byte) int {
	if len(seq) < 2 {
		return cursor
	}
	switch seq[1] {
	case 'A':
		return moveCursor(cursor, length, -1)
	case 'B':
		return moveCursor(cursor, length, 1)
	default:
		return cursor
	}
}

func safeItems(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

const (
	promptColor = "\033[1;36m"
	cursorColor = "\033[1;37m"
	selectColor = "\033[1;32m"
	resetColor  = "\033[0m"

	keyCtrlC     = 3
	keyEnter     = 10
	keyReturn    = 13
	keyEsc       = 27
	keyBackspace = 8
	keyDelete    = 127
)
