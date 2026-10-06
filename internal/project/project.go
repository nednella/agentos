package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/nednella/agentos/internal/atomicfile"
)

// Project is the unit agentos is scoped to: a name and the folder agents start in.
type Project struct {
	Name               string    `yaml:"name"`
	Dir                string    `yaml:"directory"`
	QueueSections      []Section `yaml:"queue_sections,omitempty"`          // the groups of the queue, each with the actions that start a session
	Model              string    `yaml:"session_model,omitempty"`           // overrides the config's model for this project
	Effort             string    `yaml:"session_effort,omitempty"`          // overrides the config's effort for this project
	NoteSessionCommand string    `yaml:"note_session_command,omitempty"`    // typed into a session started from a note; {text} is the note, the note itself when unset
	SessionPromptSend  string    `yaml:"session_prompt_send,omitempty"`     // auto (the default) sends what a session starts with at once; manual waits for Enter
	Branch             string    `yaml:"session_branch_fallback,omitempty"` // branch of an issue's work when the session reports none; {n} is the number
	CleanupCommand     string    `yaml:"session_cleanup_command,omitempty"` // shell command that cleans up the git side of a session; {branch}, {worktree}, {dir} and {force}
	CleanupMode        Cleanup   `yaml:"session_cleanup_mode,omitempty"`    // whether the app cleans up by itself, per event
	URL                string    `yaml:"browser_start_url,omitempty"`       // the page a session's browser opens first
	Browser            *bool     `yaml:"browser_enabled,omitempty"`         // give sessions the browser and evidence commands; on unless false
	Digest             string    `yaml:"digest_schedule,omitempty"`         // weekly (the default) or off
	PRWatch            string    `yaml:"pr_watch_method,omitempty"`         // webhook or poll; unset tries the webhook and polls when it does not work
	PRPoll             string    `yaml:"pr_poll_interval,omitempty"`        // how often to poll pull requests, "30s" by default
	KeepAwake          *bool     `yaml:"keep_mac_awake,omitempty"`          // overrides the config's keep_mac_awake for this project
	OnReview           string    `yaml:"pr_review_command,omitempty"`       // typed into a session whose PR got a review or comment; {n} is the PR number
	OnChecks           string    `yaml:"pr_checks_command,omitempty"`       // typed into a session whose PR has failing checks; {n} is the PR number
}

// Present says whether the project's folder exists on this machine.
func (p Project) Present() bool {
	info, err := os.Stat(p.Dir)
	return err == nil && info.IsDir()
}

// Config is the optional ~/.config/agentos/config.yaml.
type Config struct {
	DataDir   string    `yaml:"data_dir,omitempty"` // where notes, stats and digests live; may sit in a synced folder
	Agent     string    `yaml:"agent_command"`
	Model     string    `yaml:"session_model,omitempty"`  // model of a Claude session; unset leaves it to claude
	Effort    string    `yaml:"session_effort,omitempty"` // effort of a Claude session; unset leaves it to claude
	KeepAwake *bool     `yaml:"keep_mac_awake,omitempty"` // hold off idle sleep while a session works; on unless false
	Theme     string    `yaml:"app_theme,omitempty"`      // light or dark; unset follows the system
	TextScale float64   `yaml:"app_text_scale,omitempty"` // size of the app's text, between MinTextScale and MaxTextScale; unset is 1
	Projects  []Project `yaml:"projects"`
}

// Themes are the values of Config.Theme.
var Themes = []string{"light", "dark"}

// The text sizes the app accepts, as a multiple of its default.
const (
	MinTextScale = 0.85
	MaxTextScale = 1.5
)

// CheckTextScale rejects a text size outside the accepted range.
func CheckTextScale(scale float64) error {
	if scale < MinTextScale || scale > MaxTextScale {
		return fmt.Errorf("text scale must be between %v and %v, not %v", MinTextScale, MaxTextScale, scale)
	}
	return nil
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
	if err := cfg.Default().Validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	if cfg.Theme != "" && !slices.Contains(Themes, cfg.Theme) {
		return Config{}, fmt.Errorf("%s: app_theme must be %s, not %q", path, strings.Join(Themes, " or "), cfg.Theme)
	}
	if cfg.TextScale != 0 {
		if err := CheckTextScale(cfg.TextScale); err != nil {
			return Config{}, fmt.Errorf("%s: app_text_scale: %w", path, err)
		}
	}
	for i, p := range cfg.Projects {
		if p.Name == "" || p.Dir == "" {
			return Config{}, fmt.Errorf("%s: project %d needs a name and a directory", path, i+1)
		}
		cfg.Projects[i].Dir = ExpandHome(p.Dir)
		if p.PRWatch != "" && p.PRWatch != "webhook" && p.PRWatch != "poll" {
			return Config{}, fmt.Errorf("%s: project %s: pr_watch_method must be webhook or poll, not %q", path, p.Name, p.PRWatch)
		}
		if err := p.CleanupMode.validate(); err != nil {
			return Config{}, fmt.Errorf("%s: project %s: %w", path, p.Name, err)
		}
		if err := p.validateModel(); err != nil {
			return Config{}, fmt.Errorf("%s: project %s: %w", path, p.Name, err)
		}
		if err := p.validateQueue(); err != nil {
			return Config{}, fmt.Errorf("%s: project %s: %w", path, p.Name, err)
		}
		if _, err := p.PRPollEvery(); err != nil {
			return Config{}, fmt.Errorf("%s: project %s: %w", path, p.Name, err)
		}
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

// Save writes the config file, creating its folder. An existing file is edited, not
// rewritten: projects and settings that did not change keep their comments, their
// layout and their "~" paths.
func Save(path string, c Config) error {
	data, err := encode(path, c)
	if err != nil {
		return err
	}
	return atomicfile.Write(path, data, 0o600)
}

func encode(path string, c Config) ([]byte, error) {
	var doc yaml.Node
	if old, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(old, &doc)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		if err := doc.Encode(c); err != nil {
			return nil, fmt.Errorf("encoding config: %w", err)
		}
		return render(&doc)
	}
	root := doc.Content[0]
	syncScalar(root, "agent_command", c.Agent, nil)
	syncScalar(root, "data_dir", c.DataDir, ExpandHome)
	syncScalar(root, "app_theme", c.Theme, nil)
	syncScalar(root, "app_text_scale", formatScale(c.TextScale), nil)
	syncScalar(root, "keep_mac_awake", formatBool(c.KeepAwake), nil)
	if err := syncProjects(root, c.Projects); err != nil {
		return nil, err
	}
	return render(&doc)
}

func render(doc *yaml.Node) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("encoding config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("encoding config: %w", err)
	}
	return buf.Bytes(), nil
}

// field is the value node of key in a mapping node, or nil.
func field(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func dropField(m *yaml.Node, key string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content = slices.Delete(m.Content, i, i+2)
			return
		}
	}
}

// syncScalar makes the key hold want. A value that already means want, once norm has run on it, stays as written.
func syncScalar(m *yaml.Node, key, want string, norm func(string) string) {
	node := field(m, key)
	switch {
	case node == nil && want != "":
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, &yaml.Node{Kind: yaml.ScalarNode, Value: want})
	case node == nil:
	case want == "":
		dropField(m, key)
	default:
		have := node.Value
		if norm != nil {
			have = norm(have)
		}
		if have != want {
			node.Value, node.Tag, node.Style = want, "", 0
		}
	}
}

func formatScale(scale float64) string {
	if scale == 0 {
		return ""
	}
	return strconv.FormatFloat(scale, 'f', -1, 64)
}

func formatBool(b *bool) string {
	if b == nil {
		return ""
	}
	return strconv.FormatBool(*b)
}

// syncProjects makes the projects list hold want. A project whose entry already means the same keeps its text.
func syncProjects(root *yaml.Node, want []Project) error {
	seq := field(root, "projects")
	if seq == nil || seq.Kind != yaml.SequenceNode {
		seq = &yaml.Node{Kind: yaml.SequenceNode}
		if i := slices.IndexFunc(root.Content, func(n *yaml.Node) bool { return n.Value == "projects" }); i >= 0 && i%2 == 0 {
			root.Content[i+1] = seq
		} else {
			root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "projects"}, seq)
		}
	}
	if len(seq.Content) == 0 {
		seq.Style = 0
	}
	var next []*yaml.Node
	for _, p := range want {
		i := slices.IndexFunc(seq.Content, func(n *yaml.Node) bool { return sameProject(n, p) })
		if i < 0 {
			i = editInPlace(seq, p)
		}
		if i >= 0 {
			next = append(next, seq.Content[i])
			continue
		}
		var n yaml.Node
		if err := n.Encode(p); err != nil {
			return fmt.Errorf("encoding project %s: %w", p.Name, err)
		}
		next = append(next, &n)
	}
	seq.Content = next
	return nil
}

// editInPlace updates the entry of p when only the settings the panel writes changed, so the rest of the entry keeps
// its comments and layout. It returns the entry's index, or -1 when there is no such entry.
func editInPlace(seq *yaml.Node, p Project) int {
	i := slices.IndexFunc(seq.Content, func(n *yaml.Node) bool { f := field(n, "name"); return f != nil && f.Value == p.Name })
	if i < 0 {
		return -1
	}
	syncCleanup(seq.Content[i], p.CleanupMode)
	syncScalar(seq.Content[i], "browser_enabled", formatBool(p.Browser), nil)
	syncScalar(seq.Content[i], "digest_schedule", p.Digest, nil)
	if !sameProject(seq.Content[i], p) {
		return -1
	}
	return i
}

func syncCleanup(project *yaml.Node, want Cleanup) {
	node := field(project, "session_cleanup_mode")
	if node == nil || node.Kind != yaml.MappingNode {
		if want == (Cleanup{}) {
			return
		}
		dropField(project, "session_cleanup_mode")
		node = &yaml.Node{Kind: yaml.MappingNode}
		project.Content = append(project.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "session_cleanup_mode"}, node)
	}
	syncScalar(node, "merge", want.Merge, nil)
	syncScalar(node, "close", want.Close, nil)
	if len(node.Content) == 0 {
		dropField(project, "session_cleanup_mode")
	}
}

func sameProject(n *yaml.Node, p Project) bool {
	var have Project
	if n.Decode(&have) != nil {
		return false
	}
	have.Dir = ExpandHome(have.Dir)
	return reflect.DeepEqual(have, p)
}
