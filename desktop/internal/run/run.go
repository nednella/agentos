// Package run is how the services call other programs: gh, git and shell commands.
package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	// GHTimeout bounds a call to gh.
	GHTimeout = 20 * time.Second

	// outputCap bounds what a command may print before the rest is dropped.
	outputCap = 4 << 20
	// waitDelay is how long a cancelled command's output pipes may stay open.
	waitDelay = 5 * time.Second
)

// Runner runs a command in dir and returns its stdout. Services take one as an
// argument so tests never reach GitHub.
type Runner func(ctx context.Context, dir, name string, args ...string) ([]byte, error)

// EnvRunner is a Runner that also sets the command's environment (nil keeps ours).
type EnvRunner func(ctx context.Context, dir string, env []string, name string, args ...string) ([]byte, error)

// Exec is the Runner that runs the real command.
func Exec(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	return ExecEnv(ctx, dir, nil, name, args...)
}

// ExecEnv is the EnvRunner that runs the real command. The command and what it starts
// share a process group, and the whole group is killed when ctx ends.
func ExecEnv(ctx context.Context, dir string, env []string, name string, args ...string) ([]byte, error) {
	if name == "" {
		return nil, errors.New("running a command: no command given")
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir, cmd.Env = dir, env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = waitDelay
	stdout, stderr := &capped{max: outputCap}, &capped{max: outputCap}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	if err := cmd.Run(); err != nil {
		msg, _, _ := strings.Cut(strings.TrimSpace(stderr.String()), "\n")
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args[:min(len(args), 1)], ""), err, msg)
	}
	return stdout.Bytes(), nil
}

// capped keeps the first max bytes written and drops the rest.
type capped struct {
	buf bytes.Buffer // not embedded: its ReadFrom would let io.Copy bypass the cap
	max int
}

func (c *capped) Write(p []byte) (int, error) {
	if room := c.max - c.buf.Len(); room > 0 {
		c.buf.Write(p[:min(len(p), room)])
	}
	return len(p), nil
}

func (c *capped) Bytes() []byte  { return c.buf.Bytes() }
func (c *capped) String() string { return c.buf.String() }
