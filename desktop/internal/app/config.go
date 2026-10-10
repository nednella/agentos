// Package app wires the services of the desktop app together.
package app

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/nednella/agentos/desktop/projects"
	"github.com/nednella/agentos/desktop/stats"
	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/atomicfile"
	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/session"
	"github.com/nednella/agentos/internal/term"
)

const defaultTmuxSocket = "agentos"

// Config is what the app reads from the config file and the environment.
type Config struct {
	Registry     *projects.Registry
	Project      project.Project
	StateDir     string
	Machine      string // names this machine's event files in a data folder other machines share
	DataDir      string // notes, evidence and stats: may be a synced folder
	LocalDir     string // PR tracking and the clean-up log: never synced
	DataDirFixed bool   // AGENTOS_DEV_DATA_DIR sets the data folder, so the settings cannot change it
	Tmux         *term.Tmux
	Agent        agent.Agent
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
	machine, err := machineID(stateDir)
	if err != nil {
		return Config{}, err
	}
	dataDir := cmp.Or(cfg.DataDir, filepath.Join(home, ".local", "share", "agentos"))
	localDir := filepath.Join(home, ".local", "share", "agentos")
	// A check still setting the old name would otherwise write to the real data folder.
	if os.Getenv("AGENTOS_DATA_DIR") != "" {
		return Config{}, errors.New("AGENTOS_DATA_DIR is now AGENTOS_DEV_DATA_DIR: rename it")
	}
	override := os.Getenv("AGENTOS_DEV_DATA_DIR")
	if override != "" {
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
	return Config{Registry: projects.NewRegistry(path, cfg), Project: current, StateDir: stateDir, Machine: machine, DataDir: dataDir, LocalDir: localDir, DataDirFixed: override != "", Tmux: tmux, Agent: pickAgent(cfg.Agent, slices.Compact([]string{dataDir, localDir}))}, nil
}

// machineID is this machine's id, made on first use and kept in the state folder, which is never synced.
func machineID(stateDir string) (string, error) {
	path := filepath.Join(stateDir, "machine-id")
	b, err := os.ReadFile(path)
	if id := strings.TrimSpace(string(b)); err == nil && stats.ValidMachine(id) {
		return id, nil
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("reading machine id: %w", err)
	}
	id := session.NewToken()
	if err := atomicfile.Write(path, []byte(id+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("saving machine id: %w", err)
	}
	return id, nil
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
