package process

import (
	"os/exec"
	"syscall"
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

// ForegroundChecker monitors foreground process state
type ForegroundChecker struct {
	lastPID int32
	isFg    bool
	err     error
}

// NewForegroundChecker creates a new foreground checker
func NewForegroundChecker() *ForegroundChecker {
	return &ForegroundChecker{}
}

// Check returns whether the given PID is in foreground
func (f *ForegroundChecker) Check(pid int32) (bool, error) {
	if f.lastPID == pid && f.err == nil {
		return f.isFg, nil
	}
	f.lastPID = pid
	f.isFg, f.err = IsForegroundProcess(pid)
	return f.isFg, f.err
}

// IsInteractiveShell checks if the command is running in an interactive shell
func IsInteractiveShell(cmd *exec.Cmd) bool {
	return cmd.SysProcAttr != nil && cmd.SysProcAttr.Setctty
}