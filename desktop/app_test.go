package main

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
			cfg, err := loadConfig()
			if err != nil || cfg.project.Name != tt.want {
				t.Errorf("project = %q, %v; want %q", cfg.project.Name, err, tt.want)
			}
		})
	}

	t.Run("Finder with no projects falls back to the home folder", func(t *testing.T) {
		useConfig(t, "agent: claude\n")
		t.Setenv("AGENTOS_DIR", "/")
		cfg, err := loadConfig()
		if err != nil || cfg.project.Dir != home {
			t.Errorf("project = %+v, %v; want the home folder %s", cfg.project, err, home)
		}
	})
}

func TestLoadConfigRejectsABrokenFile(t *testing.T) {
	useConfig(t, "projects:\n  - {name: api}\n")
	if _, err := loadConfig(); err == nil {
		t.Error("a project without a folder was accepted")
	}
}

func TestSnapshotListsTheProjectsOnce(t *testing.T) {
	root := t.TempDir()
	api := filepath.Join(root, "api")
	if err := os.MkdirAll(api, 0o700); err != nil {
		t.Fatal(err)
	}
	useConfig(t, "projects:\n  - {name: api, dir: "+api+"}\n")

	t.Run("a configured project is current", func(t *testing.T) {
		t.Setenv("AGENTOS_DIR", api)
		cfg, err := loadConfig()
		if err != nil {
			t.Fatal(err)
		}
		snap := newApp(cfg, host{emit: func(string, any) {}}).Snapshot()
		if snap.Project.Name != "api" || len(snap.Projects) != 1 || snap.Version != "dev" {
			t.Errorf("snapshot = %+v", snap)
		}
	})

	t.Run("a folder in no project is listed too", func(t *testing.T) {
		scratch := filepath.Join(root, "scratch")
		if err := os.MkdirAll(scratch, 0o700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("AGENTOS_DIR", scratch)
		cfg, err := loadConfig()
		if err != nil {
			t.Fatal(err)
		}
		snap := newApp(cfg, host{emit: func(string, any) {}}).Snapshot()
		if snap.Project.Name != "scratch" || len(snap.Projects) != 2 || snap.Projects[1].Name != "scratch" {
			t.Errorf("snapshot = %+v", snap)
		}
	})
}
