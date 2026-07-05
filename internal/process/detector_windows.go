//go:build windows

package process

import (
	"os"
	"os/exec"
)

// IsForegroundProcess is not implemented on Windows yet.
func IsForegroundProcess(pid int32) (bool, error) {
	return false, nil
}

// ForegroundChecker monitors foreground process state.
type ForegroundChecker struct{}

// NewForegroundChecker creates a new foreground checker.
func NewForegroundChecker() *ForegroundChecker {
	return &ForegroundChecker{}
}

// CheckPTY returns whether pid's process group owns the foreground of pty.
func (f *ForegroundChecker) CheckPTY(_ *os.File, _ int) (bool, error) {
	return true, nil
}

// IsInteractiveShell checks if the command is running in an interactive shell.
func IsInteractiveShell(cmd *exec.Cmd) bool {
	return false
}
