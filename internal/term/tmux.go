package term

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
	chatOption   = "@agentos-chat"
	actionOption = "@agentos-action"
	agentOption  = "@agentos-agent"
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
	Model   string // the model the agent runs: the start choice until its first turn, then what it used last, or ""
	Effort  string // the effort the agent runs, kept like Model, or ""
	Agent   string // the provider, or the plain command, the session was started with, or ""
	Chat    bool   // a chat: a read-only session for questions, not work
	Action  string // the queue action the session was started with, or ""
	Created time.Time
	PID     int // the process the session's pane runs: the agent, or the shell that started it
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

// inherited are dropped from the environment the tmux server starts with: TMUX so an app started
// inside another tmux can still attach, and a session's identity so an app or test started inside
// an agentos session does not pass it on to every session and shell it starts.
var inherited = []string{"TMUX", "AGENTOS_SESSION", "AGENTOS_ISSUE", "AGENTOS_PROJECT"}

func cleanEnv() []string {
	return slices.DeleteFunc(os.Environ(), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return slices.Contains(inherited, name)
	})
}

// ListAll returns the agent sessions and the shell sessions.
// No running server means no sessions; any other failure is an error, because
// callers must not read it as "everything ended".
func (t *Tmux) ListAll(ctx context.Context) ([]Info, []session.Name, error) {
	out, err := t.run(ctx, "list-sessions", "-F", "#{session_name}\t#{"+titleOption+"}\t#{session_path}\t#{"+issueOption+"}\t#{"+modelOption+"}\t#{"+effortOption+"}\t#{"+chatOption+"}\t#{"+actionOption+"}\t#{"+agentOption+"}\t#{session_created}\t#{pane_pid}")
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && noServer(err.Error()) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var infos []Info
	var shells []session.Name
	for line := range strings.SplitSeq(out, "\n") {
		f := strings.Split(line, "\t")
		if len(f) != 11 {
			continue
		}
		name, err := session.ParseName(f[0])
		if err != nil {
			continue
		}
		if name.IsShell() {
			shells = append(shells, name)
			continue
		}
		info := Info{Name: name, Title: f[1], Path: f[2], Issue: f[3], Model: f[4], Effort: f[5], Chat: f[6] != "", Action: f[7], Agent: f[8]}
		if secs, err := strconv.ParseInt(f[9], 10, 64); err == nil {
			info.Created = time.Unix(secs, 0)
		}
		info.PID, _ = strconv.Atoi(f[10])
		infos = append(infos, info)
	}
	return infos, shells, nil
}

// ServerPID is the tmux server's process, or 0 when no server runs.
func (t *Tmux) ServerPID(ctx context.Context) (int, error) {
	out, err := t.run(ctx, "display-message", "-p", "#{pid}")
	if err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && noServer(err.Error()) {
			return 0, nil
		}
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(out))
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

// Kill stops a session and every process under its pane. tmux only hangs the pane up, and a child that ignores
// SIGHUP or runs in a process group of its own would outlive the session.
func (t *Tmux) Kill(ctx context.Context, name session.Name) error {
	var procs map[int]proc
	if out, err := t.run(ctx, "display-message", "-p", "-t", target(name), "#{pane_pid}"); err == nil {
		pid, _ := strconv.Atoi(strings.TrimSpace(out))
		if all, err := processes(ctx); err == nil && pid > 0 {
			procs = tree(all, pid)
		}
	}
	if _, err := t.run(ctx, "kill-session", "-t", target(name)); err != nil {
		return fmt.Errorf("stopping session %s: %w", name, err)
	}
	stopTree(ctx, procs)
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

// SetChat records that a session is a chat.
func (t *Tmux) SetChat(ctx context.Context, name session.Name) error {
	if _, err := t.run(ctx, "set-option", "-t", target(name), chatOption, "1"); err != nil {
		return fmt.Errorf("tagging chat %s: %w", name, err)
	}
	return nil
}

// SetAction records the queue action a session was started with.
func (t *Tmux) SetAction(ctx context.Context, name session.Name, action string) error {
	if _, err := t.run(ctx, "set-option", "-t", target(name), actionOption, action); err != nil {
		return fmt.Errorf("recording the action of session %s: %w", name, err)
	}
	return nil
}

// SetAgent records the provider, or the plain command, a session was started with.
func (t *Tmux) SetAgent(ctx context.Context, name session.Name, agent string) error {
	if _, err := t.run(ctx, "set-option", "-t", target(name), agentOption, arg(agent)); err != nil {
		return fmt.Errorf("recording the agent of session %s: %w", name, err)
	}
	return nil
}

// SetModel records the model and effort a session's agent runs.
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

// Capture is the text the session's pane shows now.
func (t *Tmux) Capture(ctx context.Context, name session.Name) (string, error) {
	out, err := t.run(ctx, "capture-pane", "-p", "-t", target(name))
	if err != nil {
		return "", fmt.Errorf("reading session %s: %w", name, err)
	}
	return out, nil
}

// oneLine keeps a title from breaking the tab-separated list: tabs and line breaks become spaces.
func oneLine(title string) string { return strings.Join(strings.Fields(title), " ") }
