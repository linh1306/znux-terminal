//go:build !windows

package pty

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
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
	cmd := shellCommand(shellPath)
	master, err := pty.Start(cmd)
	if err != nil {
		return nil, nil, err
	}
	return cmd, master, nil
}

func shellCommand(shellPath string) *exec.Cmd {
	base := filepath.Base(shellPath)
	switch base {
	case "bash":
		if rcPath, err := writeBashPromptRC(); err == nil {
			return exec.Command(shellPath, "--rcfile", rcPath, "-i")
		}
	case "zsh":
		if zdotdir, err := writeZshPromptRC(); err == nil {
			cmd := exec.Command(shellPath, "-i")
			cmd.Env = append(os.Environ(), "ZDOTDIR="+zdotdir)
			return cmd
		}
	}

	cmd := exec.Command(shellPath)
	cmd.Env = append(os.Environ(),
		"PS1=\033[1;36m◆\033[0m \033[1;92m\\W\033[0m \033[1;36m◆\033[0m ",
		"PROMPT=\033[1;36m◆\033[0m \033[1;92m%1~\033[0m \033[1;36m◆\033[0m ",
	)
	return cmd
}

func writeBashPromptRC() (string, error) {
	dir, err := os.MkdirTemp("", "znux-terminal-bash-")
	if err != nil {
		return "", err
	}
	rcPath := filepath.Join(dir, "bashrc")
	home := os.Getenv("HOME")
	content := fmt.Sprintf(
		"if [ -f %s ]; then . %s; fi\nunset PROMPT_COMMAND\nPS1='\\[\\033[1;36m\\]◆\\[\\033[0m\\] \\[\\033[1;92m\\]\\W\\[\\033[0m\\] \\[\\033[1;36m\\]◆\\[\\033[0m\\] '\n",
		shellQuote(filepath.Join(home, ".bashrc")),
		shellQuote(filepath.Join(home, ".bashrc")),
	)
	if err := os.WriteFile(rcPath, []byte(content), 0600); err != nil {
		return "", err
	}
	return rcPath, nil
}

func writeZshPromptRC() (string, error) {
	dir, err := os.MkdirTemp("", "znux-terminal-zsh-")
	if err != nil {
		return "", err
	}
	home := os.Getenv("HOME")
	content := fmt.Sprintf(
		"if [ -f %s ]; then source %s; fi\nPROMPT='%%{\\033[1;36m%%}◆%%{\\033[0m%%} %%{\\033[1;92m%%}%%1~%%{\\033[0m%%} %%{\\033[1;36m%%}◆%%{\\033[0m%%} '\nPS1=\"$PROMPT\"\n",
		shellQuote(filepath.Join(home, ".zshrc")),
		shellQuote(filepath.Join(home, ".zshrc")),
	)
	if err := os.WriteFile(filepath.Join(dir, ".zshrc"), []byte(content), 0600); err != nil {
		return "", err
	}
	return dir, nil
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
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

// DisableEcho disables echo on the PTY master fd.
// This prevents the shell from echoing input that liner already displays.
func DisableEcho(master *os.File) error {
	attrs, err := unix.IoctlGetTermios(int(master.Fd()), unix.TCGETS)
	if err != nil {
		return err
	}
	// Only disable echo flags. Keep ICANON and ISIG so the shell
	// still processes signals (Ctrl+C) and canonical mode normally.
	attrs.Lflag &^= unix.ECHO | unix.ECHOE | unix.ECHOK | unix.ECHOKE | unix.ECHOCTL | unix.ECHOPRT
	return unix.IoctlSetTermios(int(master.Fd()), unix.TCSETS, attrs)
}
