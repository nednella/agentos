package projects

import (
	"context"
	"fmt"
	"log"

	"github.com/nednella/agentos/desktop/notes"
	"github.com/nednella/agentos/desktop/sessions"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/version"
)

// Sessions is what the projects need of the session list, which also knows the current project.
type Sessions interface {
	Current() project.Project
	Projects() []sessions.Project
	List() []sessions.Session
	ShellID() string
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

// Snapshot is everything the screen needs on load.
type Snapshot struct {
	Project  sessions.Project   `json:"project"`
	Projects []sessions.Project `json:"projects"`
	Sessions []sessions.Session `json:"sessions"`
	Notes    []notes.Note       `json:"notes"`
	Version  string             `json:"version"`
	Shell    string             `json:"shell"` // the current project's shell session, "" until it is opened
}

// Service is bound to the front end.
type Service struct {
	registry *Registry
	sessions Sessions
	notes    Notes
	repos    Repos
	stateDir string
	pickDir  func() (string, error) // "" when the user cancels
	ctx      func() context.Context
}

func NewService(r *Registry, s Sessions, n Notes, repos Repos, stateDir string, pickDir func() (string, error), ctx func() context.Context) *Service {
	return &Service{registry: r, sessions: s, notes: n, repos: repos, stateDir: stateDir, pickDir: pickDir, ctx: ctx}
}

// Snapshot is everything the screen needs; call it on load.
func (s *Service) Snapshot() Snapshot {
	cur := s.sessions.Current()
	repo := s.repos.Repo(s.ctx(), cur.Dir)
	projects := s.sessions.Projects()
	snap := Snapshot{Projects: projects, Sessions: s.sessions.List(), Version: version.Version, Shell: s.sessions.ShellID()}
	snap.Project = sessions.Project{Name: cur.Name, Dir: cur.Dir}
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
