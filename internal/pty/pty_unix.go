//go:build !windows

package pty

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func NewPTY() (master, slave *os.File, err error) {
	// pty.Open returns (master, slave, err)
	return pty.Open()
}

// SpawnProcess spawns a shell using pty.Start.
// pty.Start forks a child process, creates a new session (setsid),
// opens the slave PTY, dups it to stdin/stdout/stderr, and sets it as
// the controlling terminal (Setctty). The child execs the shell.
// The master is returned for the parent to read/write.
func SpawnProcess(shellPath string, _ *os.File) (*exec.Cmd, *os.File, error) {
	cmd := exec.Command(shellPath)
	master, err := pty.Start(cmd)
	if err != nil {
		return nil, nil, err
	}
	return cmd, master, nil
}

// Resizepty resizes the PTY to match the current terminal size.
func Resizepty(master *os.File) error {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	return pty.Setsize(master, &pty.Winsize{Cols: uint16(width), Rows: uint16(height)})
}

func SetRawMode(fd uintptr) (*term.State, error) {
	return term.MakeRaw(int(fd))
}

func RestoreMode(fd uintptr, state *term.State) error {
	return term.Restore(int(fd), state)
}
