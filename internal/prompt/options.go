package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

func Options(in *os.File, out io.Writer, title string, choices []string, defaultIndexes []int) ([]string, error) {
	choices = safeItems(choices)
	if len(choices) == 0 {
		return nil, errors.New("options requires at least one choice")
	}
	if !term.IsTerminal(int(in.Fd())) {
		return nil, errors.New("interactive terminal required")
	}

	oldState, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return nil, err
	}
	defer term.Restore(int(in.Fd()), oldState)

	reader := bufio.NewReader(in)
	cursor := 0
	marked := normalizeDefaultIndexes(len(choices), defaultIndexes)
	renderedLines := 0

	render := func() error {
		clearRendered(out, renderedLines)
		renderedLines = len(choices) + 1
		fmt.Fprintf(out, "%s◇%s %s\r\n", promptColor, resetColor, title)
		for i, choice := range choices {
			marker := "□"
			color := ""
			if marked[i] {
				marker = "■"
				color = selectColor
			}
			if i == cursor {
				marker = "■"
				color = cursorColor
			}
			fmt.Fprintf(out, "│ %s%s%s %s", color, marker, resetColor, choice)
			if i < len(choices)-1 {
				fmt.Fprint(out, "\r\n")
			}
		}
		return nil
	}

	if err := render(); err != nil {
		return nil, err
	}

	for {
		b, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}

		switch b {
		case keyCtrlC:
			clearRendered(out, renderedLines)
			return nil, errors.New("cancelled")
		case keyEnter, keyReturn:
			clearRendered(out, renderedLines)
			values := selectedValues(choices, marked)
			printResult(out, title, values)
			return values, nil
		case keyEsc:
			seq := readEscapeSeq(reader)
			if len(seq) == 0 {
				clearRendered(out, renderedLines)
				return nil, nil
			}
			cursor = handleEscapeMove(cursor, len(choices), seq)
		case 'j', 'k':
			cursor = handleMoveKey(cursor, len(choices), b)
		case ' ':
			marked[cursor] = !marked[cursor]
		}

		if err := render(); err != nil {
			return nil, err
		}
	}
}
