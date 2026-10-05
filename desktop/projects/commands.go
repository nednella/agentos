package projects

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ctl "github.com/nednella/agentos/internal/control"
)

// Commands answers agentos project.
type Commands struct{ s *Service }

func NewCommands(s *Service) *Commands { return &Commands{s: s} }

// Project lists the projects, adds or removes one, or switches to the one named.
func (c *Commands) Project(_ context.Context, req ctl.Request) (string, error) {
	args := req.Args
	if len(args) == 0 {
		return c.list(), nil
	}
	switch args[0] {
	case "add":
		if len(args) < 2 {
			return "", errors.New("usage: agentos project add [path]")
		}
		snap, err := c.s.AddProjectDir(args[1])
		if err != nil {
			return "", err
		}
		return "added project " + snap.Project.Name, nil
	case "remove":
		if len(args) < 2 {
			return "", errors.New("usage: agentos project remove <name>")
		}
		if _, err := c.s.RemoveProject(args[1]); err != nil {
			return "", err
		}
		return "removed project " + args[1], nil
	}
	snap, err := c.s.SwitchProject(strings.Join(args, " "))
	if err != nil {
		return "", err
	}
	return "project " + snap.Project.Name, nil
}

func (c *Commands) list() string {
	var lines []string
	cur := c.s.sessions.Current().Name
	for _, p := range c.s.sessions.Projects() {
		mark := " "
		if p.Name == cur {
			mark = "*"
		}
		lines = append(lines, fmt.Sprintf("%s %s  %s  %d running", mark, p.Name, p.Dir, p.Sessions))
	}
	return strings.Join(lines, "\n")
}
