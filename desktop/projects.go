package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/nednella/agentos/internal/atomicfile"
	"github.com/nednella/agentos/internal/project"
)

const lastProjectFile = "last-project"

// Registry is the configured projects, saved to the config file on every change.
type Registry struct {
	path string

	mu  sync.Mutex
	cfg project.Config
}

func (r *Registry) List() []project.Project {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.cfg.Projects)
}

// Add registers a folder under its own name, or under name-2, name-3 when another
// folder already has that name. A folder that is registered already comes back as is.
func (r *Registry) Add(dir string) (project.Project, error) {
	dir, err := filepath.Abs(project.ExpandHome(dir))
	if err != nil {
		return project.Project{}, fmt.Errorf("resolving %q: %w", dir, err)
	}
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		return project.Project{}, fmt.Errorf("%s is not a folder", dir)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if i := slices.IndexFunc(r.cfg.Projects, func(p project.Project) bool { return p.Dir == dir }); i >= 0 {
		return r.cfg.Projects[i], nil
	}
	base := filepath.Base(dir)
	name := base
	for n := 2; slices.ContainsFunc(r.cfg.Projects, func(p project.Project) bool { return p.Key() == (project.Project{Name: name}).Key() }); n++ {
		name = fmt.Sprintf("%s-%d", base, n)
	}
	added := project.Project{Name: name, Dir: dir}
	next := r.cfg
	next.Projects = append(slices.Clone(r.cfg.Projects), added)
	if err := project.Save(r.path, next); err != nil {
		return project.Project{}, err
	}
	r.cfg = next
	return added, nil
}

// Remove forgets a configured project and reports whether there was one.
func (r *Registry) Remove(name string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i := slices.IndexFunc(r.cfg.Projects, func(p project.Project) bool { return p.Name == name })
	if i < 0 {
		return false, nil
	}
	next := r.cfg
	next.Projects = slices.Delete(slices.Clone(r.cfg.Projects), i, i+1)
	if err := project.Save(r.path, next); err != nil {
		return false, err
	}
	r.cfg = next
	return true, nil
}

func readLastProject(stateDir string) string {
	data, _ := os.ReadFile(filepath.Join(stateDir, lastProjectFile))
	return strings.TrimSpace(string(data))
}

func writeLastProject(stateDir, name string) error {
	return atomicfile.Write(filepath.Join(stateDir, lastProjectFile), []byte(name+"\n"), 0o600)
}
