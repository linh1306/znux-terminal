//go:build windows

package pty

import (
	"os"

	"github.com/UserExistsError/conpty"
	"golang.org/x/term"
)

type State = term.State

func NewPTY() (*conpty.PTY, error) {
	return conpty.New()
}

func SetRawMode(fd uintptr) (*term.State, error) {
	return term.MakeRaw(int(fd))
}

func RestoreMode(fd uintptr, state *term.State) error {
	return term.Restore(int(fd), state)
}

func Resizepty(pt *conpty.PTY) error {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	return pt.Resize(uint16(width), uint16(height))
}
