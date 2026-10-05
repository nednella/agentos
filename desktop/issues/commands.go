package issues

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	ctl "github.com/nednella/agentos/internal/control"
)

// PRs polls the pull requests of the sessions.
type PRs interface{ RefreshPRs() }

// Commands answers the agentos commands that work on issues.
type Commands struct {
	issues *Issues
	prs    PRs
}

func NewCommands(i *Issues, prs PRs) *Commands { return &Commands{issues: i, prs: prs} }

// Issue starts a session for each issue number given, or finds the one already running.
func (c *Commands) Issue(ctx context.Context, req ctl.Request) (string, error) {
	if len(req.Args) == 0 {
		return "", errors.New("usage: agentos issue <number...>")
	}
	var started, running, failed []string
	for _, arg := range req.Args {
		n, err := strconv.Atoi(strings.TrimPrefix(arg, "#"))
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s (not a number)", arg))
			continue
		}
		had := c.hasSession(ctx, n)
		s, err := c.issues.Start(ctx, n)
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
		noun := "sessions"
		if len(started) == 1 {
			noun = "session"
		}
		parts = append(parts, fmt.Sprintf("started %d %s: %s", len(started), noun, strings.Join(started, ", ")))
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

func (c *Commands) hasSession(ctx context.Context, number int) bool {
	list, err := c.issues.List(ctx, c.issues.sessions.Current(), false)
	if err != nil {
		return false
	}
	return slices.ContainsFunc(c.issues.WithSessions(list), func(is Issue) bool { return is.Number == number && is.SessionID != "" })
}

// Refresh reads the issues and the pull requests afresh.
func (c *Commands) Refresh(ctx context.Context, _ ctl.Request) (string, error) {
	list, err := c.issues.List(ctx, c.issues.sessions.Current(), true)
	if err != nil {
		return "", err
	}
	c.prs.RefreshPRs()
	return fmt.Sprintf("refreshed %d issues and the pull requests", len(list)), nil
}
