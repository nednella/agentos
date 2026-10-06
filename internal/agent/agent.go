package agent

import "strings"

// Launch is what a session is started with besides its name. An empty field leaves the choice to the agent.
type Launch struct{ Model, Effort string }

// Agent builds the command a session runs.
type Agent interface {
	Command(sessionName string, l Launch) []string
}

// New returns the adapter for the configured agent. "claude" (or empty) gets
// hooks; any other value runs as a plain command with no state reports.
func New(name, exe string) Agent {
	switch name {
	case "", "claude":
		return Claude{Exe: exe}
	}
	return Plain{Argv: strings.Fields(name)}
}

// Plain runs a command as is. Its sessions stay idle because nothing reports for them.
type Plain struct{ Argv []string }

func (p Plain) Command(string, Launch) []string { return p.Argv }
