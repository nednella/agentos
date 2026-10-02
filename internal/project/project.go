package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/nednella/agentos/internal/atomicfile"
)

// Project is the unit agentos is scoped to: a name and the folder agents start in.
type Project struct {
	Name     string            `yaml:"name"`
	Dir      string            `yaml:"dir"`
	Commands map[string]string `yaml:"commands,omitempty"` // keys: ready, plan, note
	Lanes    map[string]string `yaml:"lanes,omitempty"`    // GitHub label -> lane
	Branch   string            `yaml:"branch,omitempty"`   // branch of an issue's work; {n} is the number
	Cleanup  string            `yaml:"cleanup,omitempty"`  // shell command that removes a worktree; {branch} and {worktree}
	Guard    []string          `yaml:"guard,omitempty"`    // regexes for shell commands agents may not run
	URL      string            `yaml:"url,omitempty"`      // the page a session's browser opens first
	Browser  *bool             `yaml:"browser,omitempty"`  // give sessions the browser and evidence commands; on unless false
	Digest   string            `yaml:"digest,omitempty"`   // weekly (the default) or off
}

// Config is the optional ~/.config/agentos/config.yaml.
type Config struct {
	DataDir  string    `yaml:"data_dir,omitempty"` // where notes, stats and digests live; may sit in a synced folder
	Agent    string    `yaml:"agent"`
	Projects []Project `yaml:"projects"`
}

// Key is the form of the name that is safe inside a tmux session name and a file path.
func (p Project) Key() string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(p.Name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	if key := strings.TrimRight(b.String(), "-"); key != "" {
		return key
	}
	return "project"
}

// Load reads the config file. A missing file is normal and gives an empty config.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	if cfg.DataDir != "" {
		cfg.DataDir = ExpandHome(cfg.DataDir)
	}
	for i, p := range cfg.Projects {
		if p.Name == "" || p.Dir == "" {
			return Config{}, fmt.Errorf("%s: project %d needs a name and a dir", path, i+1)
		}
		cfg.Projects[i].Dir = ExpandHome(p.Dir)
	}
	return cfg, nil
}

// Resolve picks the configured project whose dir contains cwd (the deepest one
// wins), or an ad-hoc project named after the cwd folder.
func (c Config) Resolve(cwd string) Project {
	var best Project
	bestLen := -1
	for _, p := range c.Projects {
		if contains(p.Dir, cwd) && len(p.Dir) > bestLen {
			best, bestLen = p, len(p.Dir)
		}
	}
	if bestLen >= 0 {
		return best
	}
	return Project{Name: filepath.Base(cwd), Dir: cwd}
}

func contains(dir, path string) bool {
	rel, err := filepath.Rel(realPath(dir), realPath(path))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func realPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

// ExpandHome turns a leading ~ into the home folder.
func ExpandHome(p string) string {
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, p[1:])
}

// Save writes the config file, creating its folder.
func Save(path string, c Config) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	return atomicfile.Write(path, data, 0o600)
}
