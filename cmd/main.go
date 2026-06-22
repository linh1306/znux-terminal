//go:build !windows

package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

type screenHistoryWriter struct {
	current   []byte
	history   [][]byte
	viewIndex int
}

func newScreenHistoryWriter() *screenHistoryWriter {
	return &screenHistoryWriter{viewIndex: -1}
}

func (w *screenHistoryWriter) handle(op render.OutputOp) bool {
	switch op.Kind {
	case render.OutputOpPTYWrite:
		w.restoreCurrentIfViewing()
		w.current = append(w.current, op.Data...)
		return writeStdout(op.Data)
	case render.OutputOpClearCurrent:
		w.saveCurrent()
		w.current = w.current[:0]
		w.viewIndex = -1
		return true
	case render.OutputOpHistoryPrev:
		w.showPrevious()
		return true
	case render.OutputOpHistoryNext:
		w.showNext()
		return true
	default:
		w.restoreCurrentIfViewing()
		return writeStdout(op.Data)
	}
}

func (w *screenHistoryWriter) saveCurrent() {
	if len(w.current) == 0 {
		return
	}
	snapshot := append([]byte(nil), w.current...)
	w.history = append(w.history, snapshot)
	if len(w.history) > 100 {
		copy(w.history, w.history[1:])
		w.history = w.history[:len(w.history)-1]
	}
}

func (w *screenHistoryWriter) showPrevious() {
	if len(w.history) == 0 {
		return
	}
	if w.viewIndex == -1 {
		w.viewIndex = len(w.history) - 1
	} else if w.viewIndex > 0 {
		w.viewIndex--
	}
	w.renderSnapshot(w.history[w.viewIndex])
}

func (w *screenHistoryWriter) showNext() {
	if w.viewIndex == -1 {
		return
	}
	if w.viewIndex < len(w.history)-1 {
		w.viewIndex++
		w.renderSnapshot(w.history[w.viewIndex])
		return
	}
	w.viewIndex = -1
	w.renderSnapshot(w.current)
}

func (w *screenHistoryWriter) restoreCurrentIfViewing() {
	if w.viewIndex == -1 {
		return
	}
	w.viewIndex = -1
	w.renderSnapshot(w.current)
}

func (w *screenHistoryWriter) renderSnapshot(snapshot []byte) {
	_ = writeStdout([]byte("\033[H\033[2J\033[3J"))
	_ = writeStdout(snapshot)
}

func writeStdout(data []byte) bool {
	_, err := os.Stdout.Write(data)
	return err == nil
}

func main() {
	clearScreen()

	shellPath := getShellPath()

	cmd, ptm, err := gshell_pty.SpawnProcess(shellPath, nil)
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
			ch <- render.OutputOp{
				Kind: render.OutputOpPTYWrite,
				Data: append([]byte(nil), buf[:n]...),
			}
		}
	}()

	// Output writer goroutine — single goroutine owns all stdout writes.
	go func() {
		writer := newScreenHistoryWriter()
		for op := range ch {
			if !writer.handle(op) {
				return
			}
		}
	}()

	if err := dispatcher.RunWithLiner(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "dispatcher error: %v\n", err)
	}

	waitOrTerminate(cmd)
}

func clearScreen() {
	fmt.Print("\033[H\033[2J\033[3J")
}

func waitOrTerminate(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		return
	default:
	}

	_ = cmd.Process.Signal(syscall.SIGHUP)

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		_ = cmd.Process.Kill()
		<-done
	}
}

func getShellPath() string {
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}
	return "/bin/bash"
}
