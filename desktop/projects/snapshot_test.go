package projects_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nednella/agentos/desktop/internal/app"
	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/projects"
)

func useConfig(t *testing.T, body string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTOS_CONFIG", path)
}

func snapshot(cfg app.Config) projects.Snapshot {
	a := app.New(cfg, app.Host{Emit: func(string, any) {}, Clipboard: func(string) {}}, apptest.NoGH)
	for _, svc := range a.Services() {
		if s, ok := svc.(*projects.Service); ok {
			return s.Snapshot()
		}
	}
	return projects.Snapshot{}
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
		cfg, err := app.Load()
		if err != nil {
			t.Fatal(err)
		}
		snap := snapshot(cfg)
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
		cfg, err := app.Load()
		if err != nil {
			t.Fatal(err)
		}
		snap := snapshot(cfg)
		if snap.Project.Name != "scratch" || len(snap.Projects) != 2 || snap.Projects[1].Name != "scratch" {
			t.Errorf("snapshot = %+v", snap)
		}
	})
}
