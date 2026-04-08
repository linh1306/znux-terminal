package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
	gshell_pty "github.com/nguyenlinh13602/goshell/internal/pty"
	"github.com/nguyenlinh13602/goshell/internal/render"
	"github.com/nguyenlinh13602/goshell/internal/terminal"

	goshell_input "github.com/nguyenlinh13602/goshell/internal/input"
)

// outputWriter implements render.OutputChan via a channel
type outputWriter struct {
	ch chan<- render.OutputOp
}

func (w *outputWriter) WriteOp(op render.OutputOp) {
	select {
	case w.ch <- op:
	default:
	}
}

func main() {
	shellPath := getShellPath()

	// Start shell with PTY — creack/pty handles fork+raw mode+exec
	cmd := exec.Command(shellPath)
	ptm, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start PTY: %v\n", err)
		os.Exit(1)
	}
	defer ptm.Close()

	// Set raw mode on stdin for reading keyboard input
	oldState, err := gshell_pty.SetRawMode(os.Stdin.Fd())
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set raw mode on stdin: %v\n", err)
		os.Exit(1)
	}
	defer gshell_pty.RestoreMode(os.Stdin.Fd(), oldState)

	// Handle window resize
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	go func() {
		for range sigChan {
			gshell_pty.Resizepty(ptm)
		}
	}()

	// Serialized output channel
	ch := make(chan render.OutputOp, 200)

	output := &outputWriter{ch: ch}
	emulator := terminal.NewEmulator()

	dispatcher := goshell_input.NewDispatcher(os.Stdin, ptm, emulator, output)

	// PTY → serialized output goroutine
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptm.Read(buf)
			if err != nil {
				close(ch)
				return
			}
			emulator.Write(buf[:n])
			select {
			case ch <- render.OutputOp{Data: append([]byte(nil), buf[:n]...)}:
			default:
			}
		}
	}()

	// Output writer goroutine — owns stdout writes
	go func() {
		for op := range ch {
			os.Stdout.Write(op.Data)
		}
	}()

	if err := dispatcher.Run(); err != nil && err != io.EOF {
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