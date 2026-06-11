//go:build windows

package process

import "os/exec"

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

// Check returns whether the given PID is in foreground.
func (f *ForegroundChecker) Check(pid int32) (bool, error) {
	return false, nil
}

// IsInteractiveShell checks if the command is running in an interactive shell.
func IsInteractiveShell(cmd *exec.Cmd) bool {
	return false
}
