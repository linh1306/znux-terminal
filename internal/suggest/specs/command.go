package specs

import (
	"context"
	"os/exec"
	"time"
)

const generatorTimeout = 100 * time.Millisecond

func commandOutput(timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}
