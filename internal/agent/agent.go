package agent

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/nednella/agentos/internal/session"
	"github.com/nednella/agentos/internal/usage"
)

// Launch is what a session is started with besides its name. An empty field leaves the choice to the agent.
// Resume is a conversation to continue, which an agent that cannot resume ignores.
// Chat starts the agent read-only, able to run only the commands that hand work off; an agent that cannot ignores it.
// Prompt is the text agentos adds to the agent's system prompt. Start is the first prompt, for an agent that takes
// it on its command line. MCP is the server of the page tools, zero when the session has no browser. TmpDir is the
// session's own temp folder.
type Launch struct {
	Model, Effort, Resume string
	Chat                  bool
	Prompt, Start         string
	MCP                   MCPServer
	TmpDir                string
}

// MCPServer is an MCP server the agent runs on stdio, under the name "browser". Denied are the names of its
// tools the agent must not have.
type MCPServer struct {
	Command string
	Args    []string
	Env     map[string]string
	Denied  []string
}

// Capability is something a provider can do with a Launch.
type Capability int

const (
	Resume      Capability = iota // continue a conversation
	StartArg                      // take the first prompt on its command line and submit it once ready
	ModelEffort                   // take a model and an effort
	Browser                       // use the page tools of an MCP server
)

// Provider is one agent CLI as agentos runs it: the command and environment a session starts with, and the way its
// hooks and transcripts are read.
type Provider interface {
	Name() string
	Can(Capability) bool
	Command(l Launch) []string
	// Env is what the agent needs in its environment besides what every session gets.
	Env(l Launch) []string
	// Fill sets the model and effort l leaves empty to what the agent's own settings for dir choose.
	Fill(dir string, l Launch) Launch
	// ValidModel reports a model or effort the agent would not accept; an empty value is accepted.
	ValidModel(model, effort string) error
	// ParseEvent reads the agent's hook payload for its event name into the canonical event. It reports false for a
	// hook agentos does not record.
	ParseEvent(name string, r io.Reader) (session.Event, bool)
	// Usage reads what the agent's transcripts say it used after the time given.
	Usage(after time.Time) usage.Reader
}

// New returns the provider for the configured agent. "claude" (or empty) gets
// hooks and the use of dirs; any other value runs as a plain command with no state reports.
func New(name, exe string, dirs ...string) Provider {
	switch name {
	case "", "claude":
		return Claude{Exe: exe, ConfigDir: claudeConfigDir(), Dirs: dirs}
	}
	return Plain{Argv: strings.Fields(name)}
}

// Names are the providers a session may be started with by name.
var Names = []string{"claude"}

// CheckName rejects a name that is no provider's. "" names the default and passes.
func CheckName(name string) error {
	if name != "" && !slices.Contains(Names, name) {
		return fmt.Errorf("no agent %q: the agents are %s", name, strings.Join(Names, ", "))
	}
	return nil
}

// Registry makes the provider each session starts with. Exe is the agentos command the agents' hooks call, "" when
// it is not on PATH: claude then runs plain, since nothing could report for it. Dirs are the app's folders the
// agent may use.
type Registry struct {
	Exe  string
	Dirs []string
}

// Provider is the provider for command: a provider's name, "" for claude, or any other command, which runs plain.
func (r Registry) Provider(command string) Provider {
	if r.Exe == "" && (command == "" || command == "claude") {
		return Plain{Argv: []string{"claude"}}
	}
	return New(command, r.Exe, r.Dirs...)
}

// Plain runs a command as is. Its sessions stay idle because nothing reports for them.
type Plain struct{ Argv []string }

func (p Plain) Name() string                                       { return strings.Join(p.Argv, " ") }
func (p Plain) Can(Capability) bool                                { return false }
func (p Plain) Command(Launch) []string                            { return p.Argv }
func (p Plain) Env(Launch) []string                                { return nil }
func (p Plain) Fill(_ string, l Launch) Launch                     { return l }
func (p Plain) ValidModel(string, string) error                    { return nil }
func (p Plain) ParseEvent(string, io.Reader) (session.Event, bool) { return session.Event{}, false }
func (p Plain) Usage(after time.Time) usage.Reader                 { return noUsage{after} }

// noUsage reads nothing: a plain agent has no transcript agentos knows.
type noUsage struct{ after time.Time }

func (n noUsage) Read(string) ([]usage.Part, error) { return nil, nil }
func (n noUsage) After() time.Time                  { return n.after }

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
