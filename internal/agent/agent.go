package agent

import (
	"os"
	"path/filepath"
	"strings"
)

// Launch is what a session is started with besides its name. An empty field leaves the choice to the agent.
// Resume is a conversation to continue, which an agent that cannot resume ignores.
// Chat starts the agent read-only, able to run only the commands that hand work off; an agent that cannot ignores it.
type Launch struct {
	Model, Effort, Resume string
	Chat                  bool
}

// Agent builds the command a session runs.
type Agent interface {
	Command(sessionName string, l Launch) []string
	// Fill sets the model and effort l leaves empty to what the agent's own settings for dir choose.
	Fill(dir string, l Launch) Launch
}

// New returns the adapter for the configured agent. "claude" (or empty) gets
// hooks and the use of dirs; any other value runs as a plain command with no state reports.
func New(name, exe string, dirs ...string) Agent {
	switch name {
	case "", "claude":
		return Claude{Exe: exe, ConfigDir: claudeConfigDir(), Dirs: dirs}
	}
	return Plain{Argv: strings.Fields(name)}
}

// Plain runs a command as is. Its sessions stay idle because nothing reports for them.
type Plain struct{ Argv []string }

func (p Plain) Command(string, Launch) []string { return p.Argv }
func (p Plain) Fill(_ string, l Launch) Launch  { return l }

// claudeConfigDir is where claude keeps the user's settings, "" when there is no home.
func claudeConfigDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".claude")
}
