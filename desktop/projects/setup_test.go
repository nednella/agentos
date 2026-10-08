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
	if err != nil || created.Title != "Set up for agentos" {
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
