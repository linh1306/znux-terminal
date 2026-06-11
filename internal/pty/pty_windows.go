//go:build windows

package pty

import (
	"context"
	"os"

	"github.com/UserExistsError/conpty"
	"golang.org/x/term"
)

type State = term.State

func NewPTY() (*conpty.ConPty, error) {
	return conpty.Start("cmd.exe")
}

func SetRawMode(fd uintptr) (*term.State, error) {
	return term.MakeRaw(int(fd))
}

func RestoreMode(fd uintptr, state *term.State) error {
	return term.Restore(int(fd), state)
}

func Resizepty(pt *conpty.ConPty) error {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	return pt.Resize(width, height)
}

func SpawnProcess(shellPath string, _ *os.File) (*conpty.ConPty, *conpty.ConPty, error) {
	pt, err := conpty.Start(shellPath)
	if err != nil {
		return nil, nil, err
	}
	return pt, pt, nil
}

func DisableEcho(master *conpty.ConPty) error {
	return nil
}

func Wait(pt *conpty.ConPty) error {
	_, err := pt.Wait(context.Background())
	return err
}
