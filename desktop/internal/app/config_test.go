package app

import (
	"os"
	"path/filepath"
	"testing"
)

// useConfig points the app at a throwaway config file listing the given projects.
func useConfig(t *testing.T, body string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTOS_CONFIG", path)
}

func TestLoadConfigPicksTheProject(t *testing.T) {
	root := t.TempDir()
	api, web, deep, other := filepath.Join(root, "api"), filepath.Join(root, "web"), filepath.Join(root, "api", "pkg"), filepath.Join(root, "other")
	for _, d := range []string{api, web, deep, other} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	useConfig(t, "projects:\n  - {name: api, dir: "+api+"}\n  - {name: web, dir: "+web+"}\n")

	tests := []struct {
		name, dir, want string
	}{
		{"a folder inside a project", deep, "api"},
		{"a project's folder", web, "web"},
		{"a folder in no project", other, "other"},
		{"Finder starts in the root, so the first project", "/", "api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AGENTOS_DIR", tt.dir)
			cfg, err := Load()
			if err != nil || cfg.Project.Name != tt.want {
				t.Errorf("project = %q, %v; want %q", cfg.Project.Name, err, tt.want)
			}
		})
	}

	t.Run("Finder with no projects falls back to the home folder", func(t *testing.T) {
		useConfig(t, "agent: claude\n")
		t.Setenv("AGENTOS_DIR", "/")
		cfg, err := Load()
		if err != nil || cfg.Project.Dir != home {
			t.Errorf("project = %+v, %v; want the home folder %s", cfg.Project, err, home)
		}
	})
}

func TestLoadConfigRejectsABrokenFile(t *testing.T) {
	useConfig(t, "projects:\n  - {name: api}\n")
	if _, err := Load(); err == nil {
		t.Error("a project without a folder was accepted")
	}
}
