package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nednella/agentos/internal/project"
)

func TestProjects(t *testing.T) {
	h := newHarness(t)
	root := t.TempDir()
	other, clash := filepath.Join(root, "a", "other"), filepath.Join(root, "b", "other")
	for _, d := range []string{other, clash} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	snap, err := h.app.AddProjectDir(other)
	if err != nil || snap.Project.Name != "other" || snap.Project.Dir != other {
		t.Fatalf("AddProjectDir = %+v, %v", snap.Project, err)
	}
	if again, err := h.app.AddProjectDir(other); err != nil || len(again.Projects) != 2 {
		t.Errorf("adding the same folder twice: %d projects, %v", len(again.Projects), err)
	}
	if snap, err = h.app.AddProjectDir(clash); err != nil || snap.Project.Name != "other-2" {
		t.Errorf("name clash gave %+v, %v", snap.Project, err)
	}
	if _, err := h.app.AddProjectDir(filepath.Join(root, "missing")); err == nil {
		t.Error("a folder that does not exist was accepted")
	}

	cfg, err := project.Load(h.conf)
	if err != nil || cfg.Agent != "bash" || len(cfg.Projects) != 3 || cfg.Projects[0].Commands["ready"] != "/ship {n}" {
		t.Fatalf("saved config = %+v, %v", cfg, err)
	}
	if got := readLastProject(h.state); got != "other-2" {
		t.Errorf("last project = %q", got)
	}

	t.Run("picker", func(t *testing.T) {
		before := h.app.Snapshot()
		if snap, err := h.app.AddProject(); err != nil || snap.Project.Name != before.Project.Name {
			t.Errorf("a cancelled picker changed the project: %+v, %v", snap.Project, err)
		}
		picked := filepath.Join(root, "c", "picked")
		if err := os.MkdirAll(picked, 0o700); err != nil {
			t.Fatal(err)
		}
		h.rec.mu.Lock()
		h.rec.pick = picked
		h.rec.mu.Unlock()
		if snap, err := h.app.AddProject(); err != nil || snap.Project.Name != "picked" {
			t.Errorf("picked project = %+v, %v", snap.Project, err)
		}
	})

	t.Run("counts", func(t *testing.T) {
		if _, err := h.app.SwitchProject("main"); err != nil {
			t.Fatal(err)
		}
		a, err := h.app.NewSession("a", "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := h.app.SwitchProject("other"); err != nil {
			t.Fatal(err)
		}
		b, err := h.app.NewSession("b", "")
		if err != nil {
			t.Fatal(err)
		}
		h.hook(t, a.ID, "Notification", `{"message":"needs you"}`)
		h.hook(t, b.ID, "PreToolUse", `{"tool_name":"Edit"}`)
		counts := func() string {
			var out []string
			list, _ := h.rec.last("projects").([]Project)
			for _, p := range list {
				if p.Sessions > 0 {
					out = append(out, fmt.Sprintf("%s:%d/%d/%d", p.Name, p.NeedsYou, p.Working, p.Sessions))
				}
			}
			return strings.Join(out, " ")
		}
		eventually(t, "counts across projects", func() bool { return counts() == "main:1/0/1 other:0/1/1" })
		var viaSnapshot []string
		for _, p := range h.app.Snapshot().Projects {
			if p.Sessions > 0 {
				viaSnapshot = append(viaSnapshot, fmt.Sprintf("%s:%d/%d/%d", p.Name, p.NeedsYou, p.Working, p.Sessions))
			}
		}
		if got := strings.Join(viaSnapshot, " "); got != "main:1/0/1 other:0/1/1" {
			t.Errorf("snapshot counts = %s", got)
		}
	})

	t.Run("last project at start", func(t *testing.T) {
		if _, err := h.app.SwitchProject("other-2"); err != nil {
			t.Fatal(err)
		}
		t.Setenv("AGENTOS_DIR", "/")
		cfg, err := loadConfig()
		if err != nil || cfg.project.Name != "other-2" {
			t.Errorf("from Finder: %+v, %v", cfg.project, err)
		}
		t.Setenv("AGENTOS_DIR", h.dir)
		if cfg, _ = loadConfig(); cfg.project.Name != "main" {
			t.Errorf("inside a known folder: %q", cfg.project.Name)
		}
		t.Setenv("AGENTOS_DIR", "/")
		if err := writeLastProject(h.state, "gone"); err != nil {
			t.Fatal(err)
		}
		if cfg, _ = loadConfig(); cfg.project.Name != "main" {
			t.Errorf("unknown last project gave %q, want the first one", cfg.project.Name)
		}
	})

	t.Run("remove", func(t *testing.T) {
		if _, err := h.app.SwitchProject("other"); err != nil {
			t.Fatal(err)
		}
		snap, err := h.app.RemoveProject("other")
		if err != nil || snap.Project.Name == "other" {
			t.Fatalf("RemoveProject = %+v, %v", snap.Project, err)
		}
		cfg, _ := project.Load(h.conf)
		for _, p := range cfg.Projects {
			if p.Name == "other" {
				t.Error("the removed project is still in the config")
			}
		}
		listed := false
		for _, p := range snap.Projects {
			listed = listed || (p.Name == "other" && p.Sessions == 1)
		}
		if !listed {
			t.Errorf("a project with a live session vanished: %+v", snap.Projects)
		}
		if _, err := h.app.RemoveProject("nope"); err == nil {
			t.Error("removing an unknown project succeeded")
		}
	})
}
