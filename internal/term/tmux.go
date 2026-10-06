package term

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nednella/agentos/internal/session"
)

const (
	titleOption  = "@agentos-name"
	issueOption  = "@agentos-issue"
	modelOption  = "@agentos-model"
	effortOption = "@agentos-effort"
)

const conf = `
set -g status off
set -g prefix None
unbind C-b
set -g default-terminal tmux-256color
set -sa terminal-features ',*:RGB:extkeys'
set -s escape-time 0
set -g focus-events on
set -g mouse on
set -s extended-keys always
set -s extended-keys-format csi-u
set -g allow-passthrough on
set -g set-clipboard on
set -g history-limit 50000
`

// Tmux drives the private tmux server that keeps agents alive while agentos is closed.
type Tmux struct {
	socket string
	conf   string
}

// NewTmux prepares the config file in dir. socket names the private server.
func NewTmux(socket, dir string) (*Tmux, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating state dir: %w", err)
	}
	path := filepath.Join(dir, "tmux.conf")
	if err := os.WriteFile(path, []byte(conf), 0o600); err != nil {
		return nil, fmt.Errorf("writing tmux config: %w", err)
	}
	return &Tmux{socket: socket, conf: path}, nil
}

// Info is one agent session as tmux reports it.
type Info struct {
	Name    session.Name
	Title   string
	Path    string
	Issue   string // the GitHub issue number the session was started for, or ""
	Model   string // the model the agent was started with, or ""
	Effort  string // the effort the agent was started with, or ""
	Created time.Time
}

// argv runs tmux with -u: without a UTF-8 locale (an app started from Finder has none)
// tmux turns the tabs in list output into \011 and draws non-ASCII text as underscores.
func (t *Tmux) argv(args ...string) []string {
	return append([]string{"tmux", "-u", "-L", t.socket, "-f", t.conf}, args...)
}

func (t *Tmux) run(ctx context.Context, args ...string) (string, error) {
	argv := t.argv(args...)
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = cleanEnv()
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("tmux %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

// cleanEnv drops TMUX so an app started inside another tmux can still attach.
func cleanEnv() []string {
	env := os.Environ()
	out := env[:0:0]
	for _, kv := range env {
		if !strings.HasPrefix(kv, "TMUX=") {
			out = append(out, kv)
		}
	}
	return out
}

// ListAll returns the agent sessions and the projects that have a shell session.
// No running server means no sessions; any other failure is an error, because
// callers must not read it as "everything ended".
func (t *Tmux) ListAll(ctx context.Context) ([]Info, []string, error) {
	out, err := t.run(ctx, "list-sessions", "-F", "#{session_name}\t#{"+titleOption+"}\t#{session_path}\t#{"+issueOption+"}\t#{"+modelOption+"}\t#{"+effortOption+"}\t#{session_created}")
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && noServer(err.Error()) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var infos []Info
	var shells []string
	for line := range strings.SplitSeq(out, "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 7 {
			continue
		}
		name, err := session.ParseName(f[0])
		if err != nil {
			continue
		}
		if name.IsShell() {
			shells = append(shells, name.Project)
			continue
		}
		info := Info{Name: name, Title: f[1], Path: f[2], Issue: f[3], Model: f[4], Effort: f[5]}
		if secs, err := strconv.ParseInt(f[6], 10, 64); err == nil {
			info.Created = time.Unix(secs, 0)
		}
		infos = append(infos, info)
	}
	return infos, shells, nil
}

// noServer recognises tmux's message for a socket nobody listens on.
func noServer(msg string) bool {
	return strings.Contains(msg, "no server running") ||
		strings.Contains(msg, "error connecting") && (strings.Contains(msg, "No such file") || strings.Contains(msg, "Connection refused"))
}

// target names a session for -t. The "=" makes tmux match the whole name: without
// it "p/1" also matches "p/10". The ":" lets commands that take a pane accept it.
func target(name session.Name) string { return "=" + name.String() + ":" }

// arg escapes a trailing ";", which tmux would take as the end of the command.
func arg(s string) string {
	if strings.HasSuffix(s, ";") {
		return s[:len(s)-1] + `\;`
	}
	return s
}

// Has says whether a tmux session of that name exists.
func (t *Tmux) Has(ctx context.Context, name session.Name) bool {
	_, err := t.run(ctx, "has-session", "-t", target(name))
	return err == nil
}

// NewSession starts argv detached in dir at the given size.
func (t *Tmux) NewSession(ctx context.Context, name session.Name, title, dir string, env []string, argv []string, cols, rows int) error {
	args := []string{
		"new-session", "-d", "-s", name.String(), "-c", dir,
		"-x", strconv.Itoa(cols), "-y", strconv.Itoa(rows),
	}
	for _, kv := range env {
		args = append(args, "-e", kv)
	}
	args = append(args, "--")
	args = append(args, argv...)
	args = append(args, ";", "set-option", "-t", target(name), titleOption, arg(oneLine(title)))
	if _, err := t.run(ctx, args...); err != nil {
		return fmt.Errorf("starting session %s: %w", name, err)
	}
	return nil
}

func (t *Tmux) Kill(ctx context.Context, name session.Name) error {
	if _, err := t.run(ctx, "kill-session", "-t", target(name)); err != nil {
		return fmt.Errorf("stopping session %s: %w", name, err)
	}
	return nil
}

func (t *Tmux) Rename(ctx context.Context, name session.Name, title string) error {
	if _, err := t.run(ctx, "set-option", "-t", target(name), titleOption, arg(oneLine(title))); err != nil {
		return fmt.Errorf("renaming session %s: %w", name, err)
	}
	return nil
}

// SetIssue records which GitHub issue a session was started for.
func (t *Tmux) SetIssue(ctx context.Context, name session.Name, issue int) error {
	if _, err := t.run(ctx, "set-option", "-t", target(name), issueOption, strconv.Itoa(issue)); err != nil {
		return fmt.Errorf("tagging session %s: %w", name, err)
	}
	return nil
}

// SetModel records the model and effort a session's agent was started with.
func (t *Tmux) SetModel(ctx context.Context, name session.Name, model, effort string) error {
	if _, err := t.run(ctx, "set-option", "-t", target(name), modelOption, model, ";", "set-option", "-t", target(name), effortOption, effort); err != nil {
		return fmt.Errorf("recording the model of session %s: %w", name, err)
	}
	return nil
}

// AttachArgv is the command that shows a session in a terminal.
func (t *Tmux) AttachArgv(name session.Name) []string {
	return t.argv("attach-session", "-t", target(name))
}

// Env is the environment for a process that attaches to the server.
func (t *Tmux) Env() []string { return cleanEnv() }

// Type puts text into the session's prompt without pressing Enter. Text with
// line breaks goes in as a paste, so the breaks cannot submit it.
func (t *Tmux) Type(ctx context.Context, name session.Name, text string) error {
	if strings.Contains(text, "\n") {
		const buf = "agentos-prefill"
		if _, err := t.run(ctx, "set-buffer", "-b", buf, "--", arg(text), ";", "paste-buffer", "-p", "-r", "-d", "-b", buf, "-t", target(name)); err != nil {
			return fmt.Errorf("pasting into session %s: %w", name, err)
		}
		return nil
	}
	if _, err := t.run(ctx, "send-keys", "-t", target(name), "-l", "--", arg(text)); err != nil {
		return fmt.Errorf("typing into session %s: %w", name, err)
	}
	return nil
}

// Submit presses Enter in the session's prompt.
func (t *Tmux) Submit(ctx context.Context, name session.Name) error {
	if _, err := t.run(ctx, "send-keys", "-t", target(name), "Enter"); err != nil {
		return fmt.Errorf("submitting in session %s: %w", name, err)
	}
	return nil
}

// oneLine keeps a title from breaking the tab-separated list: tabs and line breaks become spaces.
func oneLine(title string) string { return strings.Join(strings.Fields(title), " ") }
