package projects_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/internal/project"
)

func TestSetUpProject(t *testing.T) {
	h := newHarness(t)
	if h.Snapshot().Project.NeedsSetup {
		t.Error("a project with queue sections is offered setting up")
	}
	dir := filepath.Join(t.TempDir(), "fresh")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	snap, err := h.AddProjectDir(dir)
	if err != nil || !snap.Project.NeedsSetup || snap.Project.SetupDismissed {
		t.Fatalf("a new project = %+v, %v", snap.Project, err)
	}

	if snap, err = h.DismissSetup(); err != nil || !snap.Project.SetupDismissed || !snap.Project.NeedsSetup {
		t.Errorf("after Not now: %+v, %v", snap.Project, err)
	}

	created, err := h.SetUpProject()
	if err != nil || created.Title != "agentos setup" {
		t.Fatalf("SetUpProject = %+v, %v", created, err)
	}
	if snap := h.Snapshot(); snap.Project.NeedsSetup {
		t.Errorf("a set-up project is still offered setting up: %+v", snap.Project)
	}
	cfg, err := project.Load(h.Conf)
	if err != nil {
		t.Fatal(err)
	}
	fresh := cfg.Projects[len(cfg.Projects)-1]
	if fresh.Name != "fresh" || len(fresh.QueueSections) != 1 || fresh.QueueSections[0].Name != "Inbox" || fresh.Branch != "issue-{n}" {
		t.Errorf("saved project = %+v", fresh)
	}
	if main := cfg.Projects[0]; main.OnReview != "" || len(main.QueueSections) != 3 {
		t.Errorf("setting up one project changed another: %+v", main)
	}
}

func TestSetUpAddsAProjectMissingFromTheConfig(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{NoProject: true})
	if snap := h.Snapshot(); snap.Project.Dir != h.Dir || !snap.Project.NeedsSetup {
		t.Fatalf("the folder the app started in = %+v", snap.Project)
	}
	if _, err := h.SetUpProject(); err != nil {
		t.Fatal(err)
	}
	cfg, err := project.Load(h.Conf)
	if err != nil || len(cfg.Projects) != 1 || cfg.Projects[0].Dir != h.Dir || cfg.Projects[0].NeedsSetup() {
		t.Errorf("saved config = %+v, %v", cfg, err)
	}
}

func TestSetUpLeavesAnotherMachinesProjectAlone(t *testing.T) {
	for _, tt := range []struct {
		name string
		act  func(h *apptest.Harness) error
	}{
		{"set up", func(h *apptest.Harness) error { _, err := h.SetUpProject(); return err }},
		{"not now", func(h *apptest.Harness) error { _, err := h.DismissSetup(); return err }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := apptest.NewWith(t, apptest.Options{NoProject: true, AbsentProject: "api"})
			if err := tt.act(h); err != nil {
				t.Fatal(err)
			}
			cfg, err := project.Load(h.Conf)
			if err != nil || len(cfg.Projects) != 2 {
				t.Fatalf("saved config = %+v, %v", cfg, err)
			}
			other, here := cfg.Projects[0], cfg.Projects[1]
			if other.Dir != "/nonexistent/api" || !other.NeedsSetup() || other.SetupDismissed || other.Branch != "" {
				t.Errorf("the other machine's project changed: %+v", other)
			}
			if here.Name != "api-2" || here.Dir != h.Dir {
				t.Errorf("the added project = %+v", here)
			}
			if tt.name == "set up" && here.NeedsSetup() || tt.name == "not now" && !here.SetupDismissed {
				t.Errorf("the added project was not edited: %+v", here)
			}
			if tt.name == "not now" && !h.Snapshot().Project.SetupDismissed {
				t.Error("the offer is still shown after Not now")
			}
		})
	}
}

func TestSetUpProjectKeepsTheConfigOfAProjectWithSections(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"\"\n"})
	before, err := os.ReadFile(h.Conf)
	if err != nil {
		t.Fatal(err)
	}
	created, err := h.SetUpProject()
	if err != nil || created.Title != "agentos setup" {
		t.Fatalf("SetUpProject = %+v, %v", created, err)
	}
	after, err := os.ReadFile(h.Conf)
	if err != nil || string(after) != string(before) {
		t.Errorf("the config changed:\n%s\nwas:\n%s", after, before)
	}
}
