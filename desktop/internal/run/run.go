// Package run is how the services call other programs: gh, git and shell commands.
package run

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// GHTimeout bounds a call to gh.
const GHTimeout = 20 * time.Second

// Runner runs a command in dir and returns its stdout. Services take one as an
// argument so tests never reach GitHub.
type Runner func(ctx context.Context, dir, name string, args ...string) ([]byte, error)

// Exec is the Runner that runs the real command.
func Exec(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n")
		return nil, fmt.Errorf("%s %s: %w: %s", name, args[0], err, msg)
	}
	return stdout.Bytes(), nil
}
