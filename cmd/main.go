package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
	gshell_pty "github.com/nguyenlinh13602/goshell/internal/pty"
	"github.com/nguyenlinh13602/goshell/internal/render"
	"github.com/nguyenlinh13602/goshell/internal/terminal"

	goshell_input "github.com/nguyenlinh13602/goshell/internal/input"
)

// outputWriter implements render.OutputChan via a channel.
type outputWriter struct {
	ch chan<- render.OutputOp
}

func (w *outputWriter) WriteOp(op render.OutputOp) {
	w.ch <- op // blocking send — never drop output silently
}

func main() {
	shellPath := getShellPath()

	cmd := exec.Command(shellPath)
	ptm, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start PTY: %v\n", err)
		os.Exit(1)
	}
	defer ptm.Close()

	// Disable PTY echo: goshell echoes every keystroke itself, so leaving the
	// shell's termios echo on would produce a second copy of the submitted
	// command that races with our own output on Enter.
	if err := gshell_pty.DisableEcho(ptm); err != nil {
		fmt.Fprintf(os.Stderr, "warning: DisableEcho failed: %v\n", err)
	}

	ptyMu := &sync.Mutex{}

	// Handle window resize
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	defer signal.Stop(sigChan)
	go func() {
		for range sigChan {
			ptyMu.Lock()
			gshell_pty.Resizepty(ptm)
			ptyMu.Unlock()
		}
	}()

	ch := make(chan render.OutputOp, 200)
	output := &outputWriter{ch: ch}
	emulator := terminal.NewEmulator()

	dispatcher := goshell_input.NewDispatcher(ptm, emulator, output, ptyMu)

	// PTY → serialized output goroutine
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptm.Read(buf)
			if err != nil {
				close(ch)
				dispatcher.Stop()
				return
			}
			emulator.Write(buf[:n])
			ch <- render.OutputOp{Data: append([]byte(nil), buf[:n]...)}
		}
	}()

	// Output writer goroutine — single goroutine owns all stdout writes.
	go func() {
		for op := range ch {
			if _, err := os.Stdout.Write(op.Data); err != nil {
				return
			}
		}
	}()

	if err := dispatcher.RunWithLiner(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "dispatcher error: %v\n", err)
	}

	cmd.Wait()
}

func getShellPath() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/bash"
}
