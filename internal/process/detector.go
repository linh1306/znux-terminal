//go:build !windows

package process

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

// IsForegroundProcess checks if current process group ID matches the process's PID
// by checking if we're the foreground process group of our controlling terminal
func IsForegroundProcess(pid int32) (bool, error) {
	// Get the process group ID of the current process
	pgid, err := syscall.Getpgid(int(pid))
	if err != nil {
		return false, err
	}

	// Get current process group
	myPgid, err := syscall.Getpgid(syscall.Getpid())
	if err != nil {
		return false, err
	}

	// If process group IDs match, it's a foreground process
	return pgid == myPgid, nil
}

// ForegroundChecker monitors foreground process state.
type ForegroundChecker struct{}

// NewForegroundChecker creates a new foreground checker
func NewForegroundChecker() *ForegroundChecker {
	return &ForegroundChecker{}
}

// CheckPTY returns whether pid's process group owns the foreground of pty.
func (f *ForegroundChecker) CheckPTY(pty *os.File, pid int) (bool, error) {
	fgPgrp, err := unix.IoctlGetInt(int(pty.Fd()), unix.TIOCGPGRP)
	if err != nil {
		return false, err
	}
	pgrp, err := syscall.Getpgid(pid)
	if err != nil {
		return false, err
	}
	return fgPgrp == pgrp, nil
}

// IsInteractiveShell checks if the command is running in an interactive shell
func IsInteractiveShell(cmd *exec.Cmd) bool {
	return cmd.SysProcAttr != nil && cmd.SysProcAttr.Setctty
}
