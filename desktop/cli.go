package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/session"
)

// viewCommands only change what the front end shows; the app tells it with a ui:command event.
var viewCommands = map[string]bool{"queue": true, "notes": true, "evidence": true, "term": true, "next": true}

// ui asks the front end to run a command, and answers ok.
func (a *App) ui(name string, args ...string) string {
	if args == nil {
		args = []string{}
	}
	a.host.emit("ui:command", map[string]any{"name": name, "args": args})
	return "ok"
}

// contextProject makes the project the command comes from the current one: the
// shell's project, else the asking session's. The screen follows it.
func (a *App) contextProject(req control.Request) {
	key := req.Project
	if name, err := session.ParseName(req.Session); key == "" && err == nil {
		key = name.Project
	}
	if key == "" || key == a.sessions.Current().Key() {
		return
	}
	_, _ = a.SwitchProject(key) // an unknown project leaves the current one
}

// sessionN finds the session with that number in the current project.
func (a *App) sessionN(arg string) (Session, error) {
	n, err := strconv.Atoi(strings.TrimPrefix(arg, "#"))
	if err != nil {
		return Session{}, fmt.Errorf("%q is not a session number", arg)
	}
	for _, s := range a.sessions.List() {
		if s.N == n {
			return s, nil
		}
	}
	return Session{}, fmt.Errorf("no session %d in this project", n)
}

// controlApp answers the commands that work the app. ok is false for any other command.
func (a *App) controlApp(ctx context.Context, req control.Request) (out string, ok bool, err error) {
	ok = true
	switch cmd := req.Cmd; {
	case viewCommands[cmd]:
		a.contextProject(req)
		return a.ui(cmd), ok, nil
	case cmd == "filter":
		a.contextProject(req)
		return a.ui("filter", req.Args...), ok, nil
	case cmd == "open":
		a.contextProject(req)
		out, err = a.cliOpen(req.Args)
	case cmd == "issue":
		a.contextProject(req)
		out, err = a.cliIssue(req.Args)
	case cmd == "new":
		a.contextProject(req)
		var s Session
		if s, err = a.NewSession(strings.Join(req.Args, " "), ""); err == nil {
			out = fmt.Sprintf("started session %d: %s", s.N, s.Title)
		}
	case cmd == "kill":
		a.contextProject(req)
		out, err = a.cliKill(req.Args)
	case cmd == "project":
		out, err = a.cliProject(req.Args)
	case cmd == "refresh":
		a.contextProject(req)
		var issues []Issue
		if issues, err = a.Issues(true); err == nil {
			a.RefreshPRs()
			out = fmt.Sprintf("refreshed %d issues and the pull requests", len(issues))
		}
	case cmd == "pr":
		a.contextProject(req)
		out, err = a.cliPR(req.Args)
	case cmd == "cleanup":
		a.contextProject(req)
		out, err = a.cliCleanup(req.Args)
	case cmd == "harness":
		a.contextProject(req)
		var s Session
		if s, err = a.HarnessCheck(); err == nil {
			out = fmt.Sprintf("started session %d: %s", s.N, s.Title)
		}
	case cmd == "digest":
		a.contextProject(req)
		if req.Opts["run"] == "" {
			return a.ui("digest"), ok, nil
		}
		if err = a.RunDigest(); err == nil {
			out = "digest started"
		}
	default:
		return "", false, nil
	}
	return out, ok, err
}

func (a *App) cliOpen(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("usage: agentos open <n|title>")
	}
	want := strings.Join(args, " ")
	list := a.sessions.List()
	if n, err := strconv.Atoi(strings.TrimPrefix(want, "#")); err == nil {
		for _, s := range list {
			if s.N == n {
				return a.ui("open", want), nil
			}
		}
		return "", fmt.Errorf("no session %d in this project", n)
	}
	var hits []Session
	for _, s := range list {
		if strings.Contains(strings.ToLower(s.Title), strings.ToLower(want)) {
			hits = append(hits, s)
		}
	}
	switch len(hits) {
	case 0:
		return "", fmt.Errorf("no session matching %q", want)
	case 1:
		return a.ui("open", want), nil
	}
	return "", fmt.Errorf("%d sessions match %q: use a number", len(hits), want)
}

func (a *App) cliIssue(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("usage: agentos issue <number...>")
	}
	var started, running, failed []string
	for _, arg := range args {
		n, err := strconv.Atoi(strings.TrimPrefix(arg, "#"))
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (not a number)", arg))
			continue
		}
		before, _ := a.Issues(false)
		had := false
		for _, is := range before {
			had = had || (is.Number == n && is.SessionID != "")
		}
		s, err := a.StartIssue(n)
		switch {
		case err != nil:
			failed = append(failed, fmt.Sprintf("#%d (%v)", n, err))
		case had:
			running = append(running, fmt.Sprintf("#%d (%d)", n, s.N))
		default:
			started = append(started, fmt.Sprintf("#%d (%d)", n, s.N))
		}
	}
	var parts []string
	if len(started) > 0 {
		parts = append(parts, fmt.Sprintf("started %d %s: %s", len(started), plural(len(started), "session", "sessions"), strings.Join(started, ", ")))
	}
	if len(running) > 0 {
		parts = append(parts, "already running: "+strings.Join(running, ", "))
	}
	if len(failed) > 0 {
		parts = append(parts, "failed: "+strings.Join(failed, ", "))
	}
	out := strings.Join(parts, "; ")
	if len(started)+len(running) == 0 {
		return "", errors.New(out)
	}
	return out, nil
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func (a *App) cliKill(args []string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("usage: agentos kill <n>")
	}
	s, err := a.sessionN(args[0])
	if err != nil {
		return "", err
	}
	if s.State == session.Ended {
		return "", fmt.Errorf("session %d has ended", s.N)
	}
	if err := a.KillSession(s.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("killed session %d: %s", s.N, s.Title), nil
}

func (a *App) cliProject(args []string) (string, error) {
	if len(args) == 0 {
		var lines []string
		cur := a.sessions.Current().Name
		for _, p := range a.sessions.Projects() {
			mark := " "
			if p.Name == cur {
				mark = "*"
			}
			lines = append(lines, fmt.Sprintf("%s %s  %s  %d running", mark, p.Name, p.Dir, p.Sessions))
		}
		return strings.Join(lines, "\n"), nil
	}
	switch args[0] {
	case "add":
		if len(args) < 2 {
			return "", errors.New("usage: agentos project add [path]")
		}
		snap, err := a.AddProjectDir(args[1])
		if err != nil {
			return "", err
		}
		return "added project " + snap.Project.Name, nil
	case "remove":
		if len(args) < 2 {
			return "", errors.New("usage: agentos project remove <name>")
		}
		if _, err := a.RemoveProject(args[1]); err != nil {
			return "", err
		}
		return "removed project " + args[1], nil
	}
	snap, err := a.SwitchProject(strings.Join(args, " "))
	if err != nil {
		return "", err
	}
	return "project " + snap.Project.Name, nil
}

func (a *App) cliPR(args []string) (string, error) {
	list := a.sessions.List()
	if len(args) > 0 {
		s, err := a.sessionN(args[0])
		if err != nil {
			return "", err
		}
		list = []Session{s}
	}
	var lines []string
	for _, s := range list {
		if s.PR == nil {
			if len(args) > 0 {
				lines = append(lines, fmt.Sprintf("%d: no pull request for %s", s.N, s.Branch))
			}
			continue
		}
		line := fmt.Sprintf("%d: #%d %s, checks %s, %d comments, %s", s.N, s.PR.Number, s.PR.State, s.PR.Checks, s.PR.Comments, s.PR.URL)
		if s.PRAttention != "" {
			line += " (" + s.PRAttention + " need you)"
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return "no pull requests", nil
	}
	return strings.Join(lines, "\n"), nil
}

func (a *App) cliCleanup(args []string) (string, error) {
	if len(args) == 0 {
		var lines []string
		for _, s := range a.sessions.List() {
			switch s.Cleanup {
			case "ask":
				lines = append(lines, fmt.Sprintf("%d: waiting for you: its pull request is closed or was merged before the session", s.N))
			case "blocked":
				lines = append(lines, fmt.Sprintf("%d: blocked: %s", s.N, s.CleanupReason))
			case "pending":
				lines = append(lines, fmt.Sprintf("%d: running", s.N))
			}
		}
		if len(lines) == 0 {
			return "nothing to clean up", nil
		}
		return strings.Join(lines, "\n"), nil
	}
	s, err := a.sessionN(args[0])
	if err != nil {
		return "", err
	}
	if err := a.Cleanup(s.ID, false); err != nil {
		return "", err
	}
	for _, now := range a.sessions.List() {
		if now.ID == s.ID && now.Cleanup == "blocked" {
			return "", fmt.Errorf("blocked: %s", now.CleanupReason)
		}
	}
	return fmt.Sprintf("cleaned up session %d", s.N), nil
}
