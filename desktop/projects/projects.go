// Package projects holds the configured projects and switches between them.
package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/atomicfile"
	"github.com/nednella/agentos/internal/project"
)

// NewRegistry keeps cfg, read from path, and saves it to path whenever a project is added or removed.
func NewRegistry(path string, cfg project.Config) *Registry {
	return &Registry{path: path, cfg: cfg, stamp: stampOf(path)}
}

const lastProjectFile = "last-project"

// Registry is the configured projects, saved to the config file on every change.
type Registry struct {
	path string

	mu    sync.Mutex
	cfg   project.Config
	stamp stamp // the file as the app last read or wrote it
}

type stamp struct {
	mod  time.Time
	size int64
}

func stampOf(path string) stamp {
	info, err := os.Stat(path)
	if err != nil {
		return stamp{}
	}
	return stamp{info.ModTime(), info.Size()}
}

// Reload reads the config file again when it changed since the app last read or wrote it, and says whether the
// config changed. A file that does not load leaves the config as it was.
func (r *Registry) Reload() (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reload()
}

func (r *Registry) reload() (bool, error) {
	now := stampOf(r.path)
	if now == r.stamp {
		return false, nil
	}
	cfg, err := project.Load(r.path)
	if err != nil {
		return false, err
	}
	r.stamp = now
	changed := !reflect.DeepEqual(cfg, r.cfg)
	r.cfg = cfg
	return changed, nil
}

// List is the configured projects whose folder exists here. The config may be shared with a
// machine that has others; those stay in the file but out of the app.
func (r *Registry) List() []project.Project {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.DeleteFunc(slices.Clone(r.cfg.Projects), func(p project.Project) bool { return !p.Present() })
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
	return r.edit(func(c *project.Config) error {
		c.Theme = theme
		return nil
	})
}

// TextScale is the configured text size, 0 when the app uses its default.
func (r *Registry) TextScale() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cfg.TextScale
}

// SetTextScale saves the text size; 0 makes the app use its default.
func (r *Registry) SetTextScale(scale float64) error {
	return r.edit(func(c *project.Config) error {
		c.TextScale = scale
		return nil
	})
}

// SetDataDir saves the folder for notes, evidence, stats and digests; the app reads it when it starts.
func (r *Registry) SetDataDir(dir string) error {
	return r.edit(func(c *project.Config) error {
		c.DataDir = dir
		return nil
	})
}

// KeepsAwake is the config's own choice on idle sleep, which a project may override.
func (r *Registry) KeepsAwake() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cfg.KeepsAwake(project.Project{})
}

// SetKeepAwake saves whether a working session holds off idle sleep.
func (r *Registry) SetKeepAwake(on bool) error {
	return r.edit(func(c *project.Config) error {
		c.KeepAwake = &on
		return nil
	})
}

// Cleanup is the project's clean-up settings, the defaults filled in. A project that is not configured has the defaults.
func (r *Registry) Cleanup(key string) project.Cleanup {
	return r.project(key).CleanupMode.Resolved()
}

// SetCleanup saves how the project cleans up after an event, "merge" or "close", to "auto" or "manual".
func (r *Registry) SetCleanup(key, event, mode string) error {
	return r.editProject(key, func(p *project.Project) error { return p.CleanupMode.Set(event, mode) })
}

// BrowserEnabled says whether the project's sessions get the browser. A project that is not configured has the default.
func (r *Registry) BrowserEnabled(key string) bool { return r.project(key).BrowserOn() }

// SetBrowserEnabled saves whether the project's sessions get the browser and evidence commands.
func (r *Registry) SetBrowserEnabled(key string, on bool) error {
	return r.editProject(key, func(p *project.Project) error {
		p.Browser = &on
		return nil
	})
}

// DigestSchedule is the project's digest schedule, the default filled in. A project that is not configured has the default.
func (r *Registry) DigestSchedule(key string) string { return r.project(key).DigestSchedule() }

// SetDigestSchedule saves the project's digest schedule, "weekly" or "off".
func (r *Registry) SetDigestSchedule(key, schedule string) error {
	if schedule != project.DigestWeekly && schedule != project.DigestOff {
		return fmt.Errorf("digest schedule must be %s or %s, not %q", project.DigestWeekly, project.DigestOff, schedule)
	}
	return r.editProject(key, func(p *project.Project) error {
		p.Digest = schedule
		return nil
	})
}

// PromptSend is the project's prompt send mode, the default filled in. A project that is not configured has the default.
func (r *Registry) PromptSend(key string) string { return r.project(key).PromptSend() }

// SetPromptSend saves whether the project sends the text a session starts with at once, "auto", or waits for Enter, "manual".
func (r *Registry) SetPromptSend(key, mode string) error {
	if mode != project.PromptAuto && mode != project.PromptManual {
		return fmt.Errorf("prompt send must be %s or %s, not %q", project.PromptAuto, project.PromptManual, mode)
	}
	return r.editProject(key, func(p *project.Project) error {
		p.SessionPromptSend = mode
		return nil
	})
}

// SetUp fills the project's empty keys with a basic agentos block.
func (r *Registry) SetUp(key string) error {
	return r.editProject(key, func(p *project.Project) error {
		p.SetUp()
		return nil
	})
}

// DismissSetup saves that the owner turned down the offer to set the project up.
func (r *Registry) DismissSetup(key string) error {
	return r.editProject(key, func(p *project.Project) error {
		p.SetupDismissed = true
		return nil
	})
}

// project is the configured project of the key, or the zero project.
func (r *Registry) project(key string) project.Project {
	r.mu.Lock()
	defer r.mu.Unlock()
	if i := r.index(key); i >= 0 {
		return r.cfg.Projects[i]
	}
	return project.Project{}
}

// edit applies change to a copy of the config, saves it, and keeps it only when the save worked. It reads the file
// first, so a hand edit made while the app runs is kept.
func (r *Registry) edit(change func(*project.Config) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.reload(); err != nil {
		return err
	}
	next := r.cfg
	if err := change(&next); err != nil {
		return err
	}
	return r.save(next)
}

// save writes next to the file and keeps it. It needs mu.
func (r *Registry) save(next project.Config) error {
	if err := project.Save(r.path, next); err != nil {
		return err
	}
	r.cfg = next
	r.stamp = stampOf(r.path)
	return nil
}

func (r *Registry) editProject(key string, change func(*project.Project) error) error {
	return r.edit(func(c *project.Config) error {
		i := r.index(key)
		if i < 0 {
			return fmt.Errorf("project %s is not in the config", key)
		}
		c.Projects = slices.Clone(c.Projects)
		return change(&c.Projects[i])
	})
}

// ByDir is the configured project whose folder is dir.
func (r *Registry) ByDir(dir string) (project.Project, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	i := slices.IndexFunc(r.cfg.Projects, func(p project.Project) bool { return p.Dir == dir })
	if i < 0 {
		return project.Project{}, false
	}
	return r.cfg.Projects[i], true
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
	if _, err := r.reload(); err != nil {
		return project.Project{}, err
	}
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
	if err := r.save(next); err != nil {
		return project.Project{}, err
	}
	return added, nil
}

// Remove forgets a configured project and reports whether there was one.
func (r *Registry) Remove(name string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, err := r.reload(); err != nil {
		return false, err
	}
	i := slices.IndexFunc(r.cfg.Projects, func(p project.Project) bool { return p.Name == name })
	if i < 0 {
		return false, nil
	}
	next := r.cfg
	next.Projects = slices.Delete(slices.Clone(r.cfg.Projects), i, i+1)
	if err := r.save(next); err != nil {
		return false, err
	}
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
