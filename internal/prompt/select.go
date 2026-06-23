package prompt

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

func Select(in *os.File, out io.Writer, title string, choices []string, defaultIndex int) (string, error) {
	choices = safeItems(choices)
	if len(choices) == 0 {
		return "", errors.New("select requires at least one choice")
	}
	if !term.IsTerminal(int(in.Fd())) {
		return "", errors.New("interactive terminal required")
	}

	oldState, err := term.MakeRaw(int(in.Fd()))
	if err != nil {
		return "", err
	}
	defer term.Restore(int(in.Fd()), oldState)

	reader := bufio.NewReader(in)
	cursor := normalizeDefaultIndex(len(choices), defaultIndex)
	selected := cursor
	renderedLines := 0

	render := func() error {
		clearRendered(out, renderedLines)
		renderedLines = len(choices) + 1
		fmt.Fprintf(out, "%s◇%s %s\r\n", promptColor, resetColor, title)
		for i, choice := range choices {
			marker := "○"
			color := ""
			if i == selected {
				marker = "●"
				color = selectColor
			}
			if i == cursor && i != selected {
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
			printResult(out, title, []string{choices[selected]})
			return choices[selected], nil
		case keyEsc:
			seq := readEscapeSeq(reader)
			if len(seq) == 0 {
				clearRendered(out, renderedLines)
				return "", nil
			}
			cursor = handleEscapeMove(cursor, len(choices), seq)
			selected = cursor
		case 'j', 'k':
			cursor = handleMoveKey(cursor, len(choices), b)
			selected = cursor
		case ' ':
			selected = cursor
		}

		if err := render(); err != nil {
			return "", err
		}
	}
}
