package sessions

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	ctl "github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/session"
)

// Scene shows things on the screen of the project a command comes from.
type Scene interface {
	UI(req ctl.Request, name string, args ...string) string
}

// Commands answers the agentos commands that work on sessions.
type Commands struct {
	s     *Sessions
	scene Scene
}

func NewCommands(s *Sessions, scene Scene) *Commands { return &Commands{s: s, scene: scene} }

// find is the session with that number in the current project.
func (c *Commands) find(arg string) (Session, error) {
	n, err := strconv.Atoi(strings.TrimPrefix(arg, "#"))
	if err != nil {
		return Session{}, fmt.Errorf("%q is not a session number", arg)
	}
	for _, s := range c.s.List() {
		if s.N == n {
			return s, nil
		}
	}
	return Session{}, fmt.Errorf("no session %d in this project", n)
}

func started(s Session) string { return fmt.Sprintf("started session %d: %s", s.N, s.Title) }

// New starts a session titled with the words given.
func (c *Commands) New(_ context.Context, req ctl.Request) (string, error) {
	s, err := c.s.Create(strings.Join(req.Args, " "), "", false, 0)
	if err != nil {
		return "", err
	}
	return started(s), nil
}

// Harness starts the harness check.
func (c *Commands) Harness(context.Context, ctl.Request) (string, error) {
	s, err := c.s.HarnessCheck()
	if err != nil {
		return "", err
	}
	return started(s), nil
}

// Open shows a session, named by its number or by words of its title, once it is one and only one.
func (c *Commands) Open(_ context.Context, req ctl.Request) (string, error) {
	if len(req.Args) == 0 {
		return "", errors.New("usage: agentos open <n|title>")
	}
	want := strings.Join(req.Args, " ")
	list := c.s.List()
	if n, err := strconv.Atoi(strings.TrimPrefix(want, "#")); err == nil {
		for _, s := range list {
			if s.N == n {
				return c.scene.UI(req, "open", want), nil
			}
		}
		return "", fmt.Errorf("no session %d in this project", n)
	}
	var hits int
	for _, s := range list {
		if strings.Contains(strings.ToLower(s.Title), strings.ToLower(want)) {
			hits++
		}
	}
	switch hits {
	case 0:
		return "", fmt.Errorf("no session matching %q", want)
	case 1:
		return c.scene.UI(req, "open", want), nil
	}
	return "", fmt.Errorf("%d sessions match %q: use a number", hits, want)
}

// Kill stops the session with that number.
func (c *Commands) Kill(_ context.Context, req ctl.Request) (string, error) {
	if len(req.Args) == 0 {
		return "", errors.New("usage: agentos kill <n>")
	}
	s, err := c.find(req.Args[0])
	if err != nil {
		return "", err
	}
	if s.State == session.Ended {
		return "", fmt.Errorf("session %d has ended", s.N)
	}
	if err := c.s.Kill(s.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("killed session %d: %s", s.N, s.Title), nil
}

// PR lists the pull requests of the sessions, or of one session.
func (c *Commands) PR(_ context.Context, req ctl.Request) (string, error) {
	list := c.s.List()
	one := len(req.Args) > 0
	if one {
		s, err := c.find(req.Args[0])
		if err != nil {
			return "", err
		}
		list = []Session{s}
	}
	var lines []string
	for _, s := range list {
		if s.PR == nil {
			if one {
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

// Cleanup cleans up after the session with that number, or lists what waits for clean-up.
func (c *Commands) Cleanup(ctx context.Context, req ctl.Request) (string, error) {
	if len(req.Args) == 0 {
		var lines []string
		for _, s := range c.s.List() {
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
	s, err := c.find(req.Args[0])
	if err != nil {
		return "", err
	}
	if err := c.s.life.Cleanup(ctx, s.ID, false); err != nil {
		return "", err
	}
	for _, now := range c.s.List() {
		if now.ID == s.ID && now.Cleanup == "blocked" {
			return "", fmt.Errorf("blocked: %s", now.CleanupReason)
		}
	}
	return fmt.Sprintf("cleaned up session %d", s.N), nil
}
