package project

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestKey(t *testing.T) {
	tests := []struct{ name, want string }{
		{"api", "api"},
		{"My App", "my-app"},
		{"a.b:c/d", "a-b-c-d"},
		{"--x--", "x"},
		{"日本", "project"},
		{"", "project"},
	}
	for _, tt := range tests {
		if got := (Project{Name: tt.name}).Key(); got != tt.want {
			t.Errorf("Key(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	files := 0
	write := func(body string) string {
		files++
		path := filepath.Join(dir, fmt.Sprintf("config%d.yaml", files))
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	tests := []struct {
		name    string
		path    string
		want    Config
		wantErr bool
	}{
		{name: "missing file", path: filepath.Join(dir, "nope.yaml")},
		{
			name: "full",
			path: write("agent: bash\nprojects:\n  - {name: api, dir: /srv/api}\n"),
			want: Config{Agent: "bash", Projects: []Project{{Name: "api", Dir: "/srv/api"}}},
		},
		{name: "project without dir", path: write("projects:\n  - {name: api}\n"), wantErr: true},
		{
			name: "pull request watch",
			path: write("projects:\n  - {name: api, dir: /srv/api, pr_watch: poll, pr_poll: 1m}\n"),
			want: Config{Projects: []Project{{Name: "api", Dir: "/srv/api", PRWatch: "poll", PRPoll: "1m"}}},
		},
		{name: "unknown pr_watch", path: write("projects:\n  - {name: api, dir: /srv/api, pr_watch: push}\n"), wantErr: true},
		{name: "bad pr_poll", path: write("projects:\n  - {name: api, dir: /srv/api, pr_poll: often}\n"), wantErr: true},
		{
			name: "models",
			path: write("session_model: haiku\nsession_effort: low\nprojects:\n  - {name: api, dir: /srv/api, session_model: opus}\n"),
			want: Config{Model: "haiku", Effort: "low", Projects: []Project{{Name: "api", Dir: "/srv/api", Model: "opus"}}},
		},
		{
			name: "queue sections",
			path: write("projects:\n  - name: api\n    dir: /srv/api\n    session_prompt_send: manual\n    queue_sections:\n      - {name: Ready, labels: [ready], actions: [{name: Work, command: \"/work {n}\", model: opus, effort: high}]}\n"),
			want: Config{Projects: []Project{{Name: "api", Dir: "/srv/api", SessionPromptSend: "manual", QueueSections: []Section{{Name: "Ready", Labels: []string{"ready"}, Actions: []Action{{Name: "Work", Command: "/work {n}", Model: "opus", Effort: "high"}}}}}}},
		},
		{name: "section without a name", path: write("projects:\n  - {name: api, dir: /srv/api, queue_sections: [{labels: [x]}]}\n"), wantErr: true},
		{name: "two sections of a name", path: write("projects:\n  - {name: api, dir: /srv/api, queue_sections: [{name: A}, {name: A}]}\n"), wantErr: true},
		{name: "action without a name", path: write("projects:\n  - {name: api, dir: /srv/api, queue_sections: [{name: A, actions: [{command: x}]}]}\n"), wantErr: true},
		{name: "two actions of a name", path: write("projects:\n  - {name: api, dir: /srv/api, queue_sections: [{name: A, actions: [{name: X}, {name: X}]}]}\n"), wantErr: true},
		{name: "bad action effort", path: write("projects:\n  - {name: api, dir: /srv/api, queue_sections: [{name: A, actions: [{name: X, effort: huge}]}]}\n"), wantErr: true},
		{name: "unknown prompt send", path: write("projects:\n  - {name: api, dir: /srv/api, session_prompt_send: later}\n"), wantErr: true},
		{name: "bad effort", path: write("session_effort: huge\n"), wantErr: true},
		{name: "model that looks like a flag", path: write("projects:\n  - {name: api, dir: /srv/api, session_model: --x}\n"), wantErr: true},
		{name: "theme", path: write("theme: dark\n"), want: Config{Theme: "dark"}},
		{name: "unknown theme", path: write("theme: sepia\n"), wantErr: true},
		{name: "bad yaml", path: write("agent: [\n"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Load(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got.Agent != tt.want.Agent || len(got.Projects) != len(tt.want.Projects) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
			for i := range got.Projects {
				if !reflect.DeepEqual(got.Projects[i], tt.want.Projects[i]) {
					t.Errorf("project %d = %+v, want %+v", i, got.Projects[i], tt.want.Projects[i])
				}
			}
		})
	}
}

func TestLoadExpandsHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	path := filepath.Join(t.TempDir(), "c.yaml")
	if err := os.WriteFile(path, []byte("projects:\n  - {name: a, dir: ~/code/a}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "code/a"); cfg.Projects[0].Dir != want {
		t.Errorf("dir = %q, want %q", cfg.Projects[0].Dir, want)
	}
}

func TestResolve(t *testing.T) {
	root := t.TempDir()
	api, web, sub := filepath.Join(root, "api"), filepath.Join(root, "web"), filepath.Join(root, "api", "pkg")
	other := filepath.Join(root, "other")
	for _, d := range []string{api, web, sub, other} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	cfg := Config{Projects: []Project{{Name: "api", Dir: api}, {Name: "apipkg", Dir: sub}, {Name: "web", Dir: web}}}
	tests := []struct {
		name string
		cwd  string
		want Project
	}{
		{"inside a project", api, Project{Name: "api", Dir: api}},
		{"deepest project wins", sub, Project{Name: "apipkg", Dir: sub}},
		{"outside every project", other, Project{Name: "other", Dir: other}},
		{"sibling sharing a prefix", filepath.Join(root, "api2"), Project{Name: "api2", Dir: filepath.Join(root, "api2")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.Resolve(tt.cwd); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolve(%q) = %+v, want %+v", tt.cwd, got, tt.want)
			}
		})
	}
}

func TestLoadDataDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	path := filepath.Join(t.TempDir(), "c.yaml")
	if err := os.WriteFile(path, []byte("data_dir: ~/Notes/agentos\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil || cfg.DataDir != filepath.Join(home, "Notes/agentos") {
		t.Errorf("data_dir = %q, %v", cfg.DataDir, err)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "config.yaml")
	want := Config{Agent: "claude", Projects: []Project{
		{Name: "a", Dir: "/srv/a", NoteSessionCommand: "/plan {text}", QueueSections: []Section{{Name: "Go", Labels: []string{"go"}, Actions: []Action{{Name: "Work", Command: "/x {n}"}}}}},
		{Name: "b", Dir: "/srv/b"},
	}}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSaveKeepsCommentsAndTildePaths(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	original := `# my agentos setup
data_dir: ~/Notes/agentos # synced
agent: claude
projects:
  # the main one
  - name: api
    dir: ~/code/api   # work
    note_session_command: "/plan {text}"
  - {name: web, dir: ~/code/web}
`
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("adding a project", func(t *testing.T) {
		cfg.Projects = append(cfg.Projects, Project{Name: "new", Dir: "/srv/new"})
		if err := Save(path, cfg); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(path)
		for _, want := range []string{"# my agentos setup", "~/Notes/agentos # synced", "# the main one", "~/code/api", "# work", "{name: web, dir: ~/code/web}", "/srv/new"} {
			if !strings.Contains(string(got), want) {
				t.Errorf("the saved file lost %q:\n%s", want, got)
			}
		}
		if strings.Contains(string(got), home) {
			t.Errorf("a ~ path was written out in full:\n%s", got)
		}
		reloaded, err := Load(path)
		if err != nil || !reflect.DeepEqual(reloaded, Config{DataDir: cfg.DataDir, Agent: "claude", Projects: cfg.Projects}) {
			t.Errorf("reload = %+v, %v", reloaded, err)
		}
	})

	t.Run("removing a project", func(t *testing.T) {
		cfg.Projects = cfg.Projects[1:]
		if err := Save(path, cfg); err != nil {
			t.Fatal(err)
		}
		got, _ := os.ReadFile(path)
		if strings.Contains(string(got), "name: api") || !strings.Contains(string(got), "# my agentos setup") || !strings.Contains(string(got), "~/code/web") {
			t.Errorf("saved file:\n%s", got)
		}
		reloaded, err := Load(path)
		if err != nil || len(reloaded.Projects) != 2 || reloaded.Projects[0].Name != "web" || reloaded.Projects[1].Name != "new" {
			t.Errorf("reload = %+v, %v", reloaded, err)
		}
	})

	t.Run("removing every project", func(t *testing.T) {
		cfg.Projects = nil
		if err := Save(path, cfg); err != nil {
			t.Fatal(err)
		}
		reloaded, err := Load(path)
		if err != nil || len(reloaded.Projects) != 0 || reloaded.Agent != "claude" {
			t.Errorf("reload = %+v, %v", reloaded, err)
		}
	})
}

func TestLoadRejectsTheOldCleanupCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("projects:\n  - name: a\n    dir: /a\n    cleanup: make clean\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "remove_worktree") {
		t.Errorf("Load = %v, want an error naming remove_worktree", err)
	}
}

func TestLoadRejectsAnUnknownCleanupMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("projects:\n  - name: a\n    dir: /a\n    cleanup: {merge: now}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "auto or manual") {
		t.Errorf("Load = %v, want an error naming auto or manual", err)
	}
}
