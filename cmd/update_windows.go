//go:build windows

package main

import (
	"fmt"
	"os"
)

func handleCLI(args []string) (bool, int) {
	if len(args) == 0 {
		return false, 0
	}

	switch args[0] {
	case "update":
		fmt.Fprintln(os.Stderr, "znux update is not implemented on Windows yet")
		return true, 1
	case "suggest":
		fmt.Fprintln(os.Stderr, "znux suggest is not implemented on Windows yet")
		return true, 1
	case "-h", "--help", "help":
		fmt.Fprintln(os.Stdout, "Usage:")
		fmt.Fprintln(os.Stdout, "  znux          start terminal")
		fmt.Fprintln(os.Stdout, "  znux update   download latest znux to ~/.local/bin")
		fmt.Fprintln(os.Stdout, "  znux suggest  manage command suggestion files")
		return true, 0
	default:
		return false, 0
	}
}
