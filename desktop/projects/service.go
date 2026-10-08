package projects

import (
	"context"
	"fmt"
	"log"

	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/desktop/notes"
	"github.com/nednella/agentos/desktop/sessions"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/prompts"
	"github.com/nednella/agentos/internal/version"
)

// Sessions is what the projects need of the session list, which also knows the current project.
type Sessions interface {
	Current() project.Project
	Projects() []sessions.Project
	List() []sessions.Session
	ShellIDs() []string
	Create(title, text string, send bool, issue int) (sessions.Session, error)
	SwitchProject(name string) (project.Project, error)
	Forget(name string) (project.Project, error)
}

// Notes lists a project's notes.
type Notes interface {
	List(key string) ([]notes.Note, error)
}

// Repos names the GitHub repo of a folder.
type Repos interface {
	Repo(ctx context.Context, dir string) string
}

// Releases knows whether a newer release of the app is out.
type Releases interface{ Available() string }

// Snapshot is everything the screen needs on load.
type Snapshot struct {
	Project  sessions.Project   `json:"project"`
	Projects []sessions.Project `json:"projects"`
	Sessions []sessions.Session `json:"sessions"`
	Notes    []notes.Note       `json:"notes"`
	Version  string             `json:"version"`
	Update   string             `json:"update"` // a newer release's version, "" when the app is current
	Shells   []string           `json:"shells"` // the current project's shell sessions in tab order, empty until one is opened
}

// Service is bound to the front end.
type Service struct {
	registry *Registry
	sessions Sessions
	notes    Notes
	repos    Repos
	releases Releases
	run      run.Runner
	stateDir string
	pickDir  func() (string, error) // "" when the user cancels
	ctx      func() context.Context
}

func NewService(r *Registry, s Sessions, n Notes, repos Repos, releases Releases, runner run.Runner, stateDir string, pickDir func() (string, error), ctx func() context.Context) *Service {
	return &Service{registry: r, sessions: s, notes: n, repos: repos, releases: releases, run: runner, stateDir: stateDir, pickDir: pickDir, ctx: ctx}
}

// Snapshot is everything the screen needs; call it on load.
func (s *Service) Snapshot() Snapshot {
	cur := s.sessions.Current()
	repo := s.repos.Repo(s.ctx(), cur.Dir)
	projects := s.sessions.Projects()
	snap := Snapshot{Projects: projects, Sessions: s.sessions.List(), Version: version.Version, Update: s.releases.Available(), Shells: s.sessions.ShellIDs()}
	snap.Project = sessions.Project{Key: cur.Key(), Name: cur.Name, Dir: cur.Dir}
	for i, p := range projects {
		if p.Dir == cur.Dir {
			projects[i].Repo = repo
			snap.Project = projects[i]
		}
	}
	var err error
	if snap.Notes, err = s.notes.List(cur.Key()); err != nil {
		log.Printf("agentos: %v", err)
		snap.Notes = []notes.Note{}
	}
	return snap
}

// SwitchProject makes the project current; it is remembered for the next start.
func (s *Service) SwitchProject(name string) (Snapshot, error) {
	p, err := s.sessions.SwitchProject(name)
	if err != nil {
		return Snapshot{}, err
	}
	s.remember(p)
	return s.Snapshot(), nil
}

func (s *Service) remember(p project.Project) {
	if err := WriteLast(s.stateDir, p.Name); err != nil {
		log.Printf("agentos: remembering the project: %v", err)
	}
}

// AddProject asks for a folder and makes it the current project.
func (s *Service) AddProject() (Snapshot, error) {
	dir, err := s.pickDir()
	if err != nil {
		return Snapshot{}, fmt.Errorf("choosing a folder: %w", err)
	}
	if dir == "" {
		return s.Snapshot(), nil
	}
	return s.AddProjectDir(dir)
}

// AddProjectDir adds a folder as a project and makes it the current one.
func (s *Service) AddProjectDir(dir string) (Snapshot, error) {
	p, err := s.registry.Add(dir)
	if err != nil {
		return Snapshot{}, err
	}
	return s.SwitchProject(p.Name)
}

// RemoveProject forgets a configured project; its sessions keep running.
func (s *Service) RemoveProject(name string) (Snapshot, error) {
	p, err := s.sessions.Forget(name)
	if err != nil {
		return Snapshot{}, err
	}
	s.remember(p)
	return s.Snapshot(), nil
}

// SetUpProject writes a basic agentos block into the current project's config and starts a session that sets up
// the repository with the user.
func (s *Service) SetUpProject() (sessions.Session, error) {
	key, err := s.configure()
	if err != nil {
		return sessions.Session{}, err
	}
	if err := s.registry.SetUp(key); err != nil {
		return sessions.Session{}, err
	}
	return s.sessions.Create("Set up for agentos", prompts.SetupStart(), s.sessions.Current().SendsPrompt(), 0)
}

// DismissSetup stops offering to set up the current project.
func (s *Service) DismissSetup() (Snapshot, error) {
	key, err := s.configure()
	if err != nil {
		return Snapshot{}, err
	}
	if err := s.registry.DismissSetup(key); err != nil {
		return Snapshot{}, err
	}
	return s.Snapshot(), nil
}

// configure is the current project's key, once the project is in the config: the folder the app started in may not be.
func (s *Service) configure() (string, error) {
	cur := s.sessions.Current()
	if s.registry.Has(cur.Key()) {
		return cur.Key(), nil
	}
	snap, err := s.AddProjectDir(cur.Dir)
	return snap.Project.Key, err
}
