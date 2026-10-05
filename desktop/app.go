package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/version"
)

// Project is a project as the front end sees it.
type Project struct {
	Name string `json:"name"`
	Dir  string `json:"dir"`
}

type Snapshot struct {
	Project  Project   `json:"project"`
	Projects []Project `json:"projects"`
	Version  string    `json:"version"`
}

type config struct {
	projects []project.Project
	project  project.Project
}

// loadConfig reads the config file. The project is the one holding AGENTOS_DIR
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
	return config{projects: cfg.Projects, project: current}, nil
}

// App is the one struct bound to the front end. Every exported method is part of the contract.
type App struct {
	ctx context.Context
	cfg config
}

func newApp(c config) *App {
	return &App{ctx: context.Background(), cfg: c}
}

// start is called once the window is up.
func (a *App) start(ctx context.Context) { a.ctx = ctx }

func (a *App) Snapshot() Snapshot {
	cur := a.cfg.project
	projects := make([]Project, 0, len(a.cfg.projects)+1)
	for _, p := range a.cfg.projects {
		projects = append(projects, Project{Name: p.Name, Dir: p.Dir})
	}
	if !slices.ContainsFunc(a.cfg.projects, func(p project.Project) bool { return p.Key() == cur.Key() }) {
		projects = append(projects, Project{Name: cur.Name, Dir: cur.Dir})
	}
	return Snapshot{Project: Project{Name: cur.Name, Dir: cur.Dir}, Projects: projects, Version: version.Version}
}
