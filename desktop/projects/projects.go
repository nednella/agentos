// Package projects holds the configured projects and switches between them.
package projects

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

// NewRegistry keeps cfg, and saves it to path whenever a project is added or removed.
func NewRegistry(path string, cfg project.Config) *Registry { return &Registry{path: path, cfg: cfg} }

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

// KeepAwake says whether a working session of the project holds off idle sleep. A project that is
// not configured follows the config.
func (r *Registry) KeepAwake(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	i := slices.IndexFunc(r.cfg.Projects, func(p project.Project) bool { return p.Key() == key })
	if i < 0 {
		return r.cfg.KeepsAwake(project.Project{})
	}
	return r.cfg.KeepsAwake(r.cfg.Projects[i])
}

// Theme is the configured theme, "" when the app follows the system.
func (r *Registry) Theme() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cfg.Theme
}

// SetTheme saves the theme; "" makes the app follow the system.
func (r *Registry) SetTheme(theme string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	next := r.cfg
	next.Theme = theme
	if err := project.Save(r.path, next); err != nil {
		return err
	}
	r.cfg = next
	return nil
}

// Cleanup is the project's clean-up settings, the defaults filled in. A project that is not configured has the defaults.
func (r *Registry) Cleanup(key string) project.Cleanup {
	r.mu.Lock()
	defer r.mu.Unlock()
	i := r.index(key)
	if i < 0 {
		return project.Cleanup{}.Resolved()
	}
	return r.cfg.Projects[i].Cleanup.Resolved()
}

// SetCleanup saves how the project cleans up after an event, "merge" or "close", to "auto" or "manual".
func (r *Registry) SetCleanup(key, event, mode string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	i := r.index(key)
	if i < 0 {
		return fmt.Errorf("project %s is not in the config", key)
	}
	next := r.cfg
	next.Projects = slices.Clone(r.cfg.Projects)
	if err := next.Projects[i].Cleanup.Set(event, mode); err != nil {
		return err
	}
	if err := project.Save(r.path, next); err != nil {
		return err
	}
	r.cfg = next
	return nil
}

func (r *Registry) index(key string) int {
	return slices.IndexFunc(r.cfg.Projects, func(p project.Project) bool { return p.Key() == key })
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

// ReadLast is the name of the project used last, or "".
func ReadLast(stateDir string) string {
	data, _ := os.ReadFile(filepath.Join(stateDir, lastProjectFile))
	return strings.TrimSpace(string(data))
}

// WriteLast remembers the project used last.
func WriteLast(stateDir, name string) error {
	return atomicfile.Write(filepath.Join(stateDir, lastProjectFile), []byte(name+"\n"), 0o600)
}
