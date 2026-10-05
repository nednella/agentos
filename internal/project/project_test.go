package project

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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
		{Name: "a", Dir: "/srv/a", Commands: map[string]string{"ready": "/x {n}"}, Lanes: map[string]string{"go": "ready"}},
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
