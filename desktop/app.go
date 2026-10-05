package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/term"
	"github.com/nednella/agentos/internal/version"
)

const defaultTmuxSocket = "agentos"

type Snapshot struct {
	Project  Project   `json:"project"`
	Projects []Project `json:"projects"`
	Sessions []Session `json:"sessions"`
	Version  string    `json:"version"`
}

// host is what the window provides. The logic never touches Wails directly.
type host struct {
	emit func(event string, payload any)
}

type config struct {
	projects []project.Project
	project  project.Project
	stateDir string
	tmux     *term.Tmux
	agent    agent.Agent
}

// loadConfig reads the config file and the environment. The project is the one holding AGENTOS_DIR
// or the current folder. An app started from Finder (folder "/") gets the first
// configured project, else the home folder.
func loadConfig() (config, error) {
	path := os.Getenv("AGENTOS_CONFIG")
	home, err := os.UserHomeDir()
	if err != nil {
		return config{}, fmt.Errorf("finding home dir: %w", err)
	}
	if path == "" {
		path = filepath.Join(home, ".config", "agentos", "config.yaml")
	}
	cfg, err := project.Load(path)
	if err != nil {
		return config{}, fmt.Errorf("loading config: %w", err)
	}
	dir := os.Getenv("AGENTOS_DIR")
	if dir == "" {
		if dir, err = os.Getwd(); err != nil {
			return config{}, fmt.Errorf("finding the current folder: %w", err)
		}
	}
	current := cfg.Resolve(dir)
	if dir == "/" {
		current = cfg.Resolve(home)
		if len(cfg.Projects) > 0 {
			current = cfg.Projects[0]
		}
	}
	stateDir, err := bus.DefaultDir()
	if err != nil {
		return config{}, err
	}
	socket := os.Getenv("AGENTOS_TMUX_SOCKET")
	if socket == "" {
		socket = defaultTmuxSocket
	}
	tmux, err := term.NewTmux(socket, stateDir)
	if err != nil {
		return config{}, err
	}
	return config{projects: cfg.Projects, project: current, stateDir: stateDir, tmux: tmux, agent: pickAgent(cfg.Agent)}, nil
}

// pickAgent builds the agent adapter. Without the agentos command on PATH the
// agents' hooks cannot call back, so Claude runs plain and sessions stay idle.
func pickAgent(name string) agent.Agent {
	exe, err := exec.LookPath("agentos")
	if err == nil {
		return agent.New(name, exe)
	}
	log.Printf("agentos: the agentos command is not on PATH, so agents will not report their state: %v", err)
	if name == "" || name == "claude" {
		return agent.Plain{Argv: []string{"claude"}}
	}
	return agent.New(name, "")
}

// App is the one struct bound to the front end. Every exported method is part of the contract.
type App struct {
	ctx      context.Context
	sessions *Sessions
}

func newApp(c config, h host) *App {
	return &App{
		ctx:      context.Background(),
		sessions: newSessions(c.tmux, c.agent, c.stateDir, c.projects, c.project, h.emit),
	}
}

// start begins listening for hooks; it runs until ctx ends.
func (a *App) start(ctx context.Context) error {
	a.ctx = ctx
	return a.sessions.Start(ctx)
}

func (a *App) Snapshot() Snapshot {
	cur := a.sessions.Current()
	projects := a.sessions.Projects()
	snap := Snapshot{Projects: projects, Sessions: a.sessions.List(), Version: version.Version}
	snap.Project = Project{Name: cur.Name, Dir: cur.Dir}
	for i, p := range projects {
		if p.Dir == cur.Dir {
			snap.Project = projects[i]
		}
	}
	return snap
}

func (a *App) NewSession(title, prefill string) (Session, error) {
	return a.sessions.Create(title, prefill)
}

func (a *App) KillSession(id string) error { return a.sessions.Kill(id) }

// DismissSession removes the row of an ended session.
func (a *App) DismissSession(id string) error { return a.sessions.Dismiss(id) }

func (a *App) RenameSession(id, title string) error { return a.sessions.Rename(id, title) }

func (a *App) SwitchProject(name string) (Snapshot, error) {
	if _, err := a.sessions.SwitchProject(name); err != nil {
		return Snapshot{}, err
	}
	return a.Snapshot(), nil
}
