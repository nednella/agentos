// Package app wires the services of the desktop app together.
package app

import (
	"cmp"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/nednella/agentos/desktop/projects"
	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/term"
)

const defaultTmuxSocket = "agentos"

// Config is what the app reads from the config file and the environment.
type Config struct {
	Registry *projects.Registry
	Project  project.Project
	StateDir string
	DataDir  string // notes, evidence and stats: may be a synced folder
	LocalDir string // PR tracking and the clean-up log: never synced
	Tmux     *term.Tmux
	Agent    agent.Agent
}

// Load reads the config file and the environment. The project is the one holding AGENTOS_DIR
// or the current folder. An app started from Finder (folder "/") gets the first
// configured project, else the home folder.
func Load() (Config, error) {
	path := os.Getenv("AGENTOS_CONFIG")
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("finding home dir: %w", err)
	}
	if path == "" {
		path = filepath.Join(home, ".config", "agentos", "config.yaml")
	}
	cfg, err := project.Load(path)
	if err != nil {
		return Config{}, fmt.Errorf("loading config: %w", err)
	}
	stateDir, err := bus.DefaultDir()
	if err != nil {
		return Config{}, err
	}
	dataDir := cmp.Or(cfg.DataDir, filepath.Join(home, ".local", "share", "agentos"))
	localDir := filepath.Join(home, ".local", "share", "agentos")
	if override := os.Getenv("AGENTOS_DATA_DIR"); override != "" {
		dataDir, localDir = override, override
	}
	dir := os.Getenv("AGENTOS_DIR")
	if dir == "" {
		if dir, err = os.Getwd(); err != nil {
			return Config{}, fmt.Errorf("finding the current folder: %w", err)
		}
	}
	current := project.Project{}
	if dir == "/" {
		last := projects.ReadLast(stateDir)
		here := slices.DeleteFunc(slices.Clone(cfg.Projects), func(p project.Project) bool { return !p.Present() })
		switch i := slices.IndexFunc(here, func(p project.Project) bool { return p.Name == last }); {
		case i >= 0:
			current = here[i]
		case len(here) > 0:
			current = here[0]
		default:
			current = cfg.Resolve(home)
		}
	} else {
		current = cfg.Resolve(dir)
	}
	socket := os.Getenv("AGENTOS_TMUX_SOCKET")
	if socket == "" {
		socket = defaultTmuxSocket
	}
	tmux, err := term.NewTmux(socket, stateDir)
	if err != nil {
		return Config{}, err
	}
	return Config{Registry: projects.NewRegistry(path, cfg), Project: current, StateDir: stateDir, DataDir: dataDir, LocalDir: localDir, Tmux: tmux, Agent: pickAgent(cfg.Agent, slices.Compact([]string{dataDir, localDir}))}, nil
}

// pickAgent builds the agent adapter. Without the agentos command on PATH the
// agents' hooks cannot call back, so Claude runs plain and sessions stay idle.
func pickAgent(name string, dirs []string) agent.Agent {
	exe, err := exec.LookPath("agentos")
	if err == nil {
		return agent.New(name, exe, dirs...)
	}
	log.Printf("agentos: the agentos command is not on PATH, so agents will not report their state: %v", err)
	if name == "" || name == "claude" {
		return agent.Plain{Argv: []string{"claude"}}
	}
	return agent.New(name, "")
}
