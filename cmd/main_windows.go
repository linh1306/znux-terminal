//go:build windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "goshell Windows runtime support is not implemented yet")
	os.Exit(1)
}
