// Package run is how the services call other programs: gh, git and shell commands.
package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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

// Streamer starts a long-running command in dir and hands back its output as it comes.
// Reading reaches EOF when the command ends; Close kills a command still running, waits,
// and returns how it ended, with the first line of stderr as Exec reports it.
type Streamer func(ctx context.Context, dir, name string, args ...string) (io.ReadCloser, error)

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

// Stream is the Streamer that starts the real command.
func Stream(ctx context.Context, dir, name string, args ...string) (io.ReadCloser, error) {
	if name == "" {
		return nil, errors.New("running a command: no command given")
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = waitDelay
	stderr := &capped{max: outputCap}
	cmd.Stderr = stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("%s %s: %w", name, strings.Join(args[:min(len(args), 1)], ""), err)
	}
	return &stream{Reader: out, cmd: cmd, stderr: stderr, args: args}, nil
}

type stream struct {
	io.Reader
	cmd    *exec.Cmd
	stderr *capped
	args   []string
}

func (s *stream) Close() error {
	_ = s.cmd.Cancel()
	err := s.cmd.Wait()
	if err == nil {
		return nil
	}
	msg, _, _ := strings.Cut(strings.TrimSpace(s.stderr.String()), "\n")
	return fmt.Errorf("%s %s: %w: %s", s.cmd.Path, strings.Join(s.args[:min(len(s.args), 1)], ""), err, msg)
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
