package projects_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewProject(t *testing.T) {
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@example.com")
	h := newHarness(t)
	parent := t.TempDir()
	h.Rec.SetPick(parent)

	snap, err := h.NewProject(" fresh ")
	dir := filepath.Join(parent, "fresh")
	if err != nil || snap.Project.Name != "fresh" || snap.Project.Dir != dir {
		t.Fatalf("NewProject = %+v, %v", snap.Project, err)
	}
	readme, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil || !strings.Contains(string(readme), "<b>fresh</b>") {
		t.Errorf("README = %q, %v", readme, err)
	}
	if !exists(filepath.Join(dir, ".git")) {
		t.Error("no git repository")
	}
	if h.GH.Calls("repo create fresh --public --source . --push") != 0 {
		t.Errorf("NewProject made the GitHub repo: gh calls = %q", h.GH.CallLog())
	}
	if _, err := h.CreateRepo("fresh"); err != nil || h.GH.Calls("repo create fresh --public --source . --push") != 1 {
		t.Errorf("CreateRepo = %v; gh calls = %q", err, h.GH.CallLog())
	}

	t.Run("rejects a bad name before asking for a folder", func(t *testing.T) {
		for _, name := range []string{"", "..", "a/b", "two words"} {
			if _, err := h.NewProject(name); err == nil {
				t.Errorf("%q was accepted", name)
			}
		}
	})

	t.Run("a cancelled picker changes nothing", func(t *testing.T) {
		h.Rec.SetPick("")
		before := h.Snapshot()
		if snap, err := h.NewProject("never"); err != nil || snap.Project.Name != before.Project.Name {
			t.Errorf("NewProject = %+v, %v", snap.Project, err)
		}
	})

	t.Run("an existing folder is left alone", func(t *testing.T) {
		h.Rec.SetPick(parent)
		_, err := h.NewProject("fresh")
		if err == nil || err.Error() != dir+" already exists" {
			t.Errorf("err = %v", err)
		}
		if h.GH.Calls("repo view fresh") != 1 {
			t.Errorf("the repo was looked up for a folder that exists: gh calls = %q", h.GH.CallLog())
		}
		if !exists(filepath.Join(dir, "README.md")) {
			t.Error("the existing folder lost its README")
		}
	})

	t.Run("an existing GitHub repo creates nothing", func(t *testing.T) {
		h.GH.ExistingRepos = []string{"online"}
		t.Cleanup(func() { h.GH.ExistingRepos = nil })
		_, err := h.NewProject("online")
		if err == nil || err.Error() != "a GitHub repo called online already exists" {
			t.Errorf("err = %v", err)
		}
		if exists(filepath.Join(parent, "online")) {
			t.Error("the folder was created")
		}
		if h.GH.Calls("repo create online --public --source . --push") != 0 {
			t.Errorf("gh calls = %q", h.GH.CallLog())
		}
	})

	t.Run("a repo that is not found does not block", func(t *testing.T) {
		if snap, err := h.NewProject("unseen"); err != nil || snap.Project.Name != "unseen" {
			t.Errorf("NewProject = %+v, %v", snap.Project, err)
		}
	})

	t.Run("a failed repo lookup does not block", func(t *testing.T) {
		h.GH.ViewErr = errors.New("gh: not logged in")
		t.Cleanup(func() { h.GH.ViewErr = nil })
		if snap, err := h.NewProject("offline"); err != nil || snap.Project.Name != "offline" {
			t.Fatalf("NewProject = %+v, %v", snap.Project, err)
		}
		h.GH.CreateRepoErr = errors.New("not logged in")
		t.Cleanup(func() { h.GH.CreateRepoErr = nil })
		if _, err := h.CreateRepo("offline"); err == nil || !strings.Contains(err.Error(), "not logged in") {
			t.Errorf("CreateRepo err = %v", err)
		}
	})

	t.Run("a GitHub failure keeps the project and can be retried", func(t *testing.T) {
		h.GH.CreateRepoErr = errors.New("name already exists")
		if snap, err := h.NewProject("taken"); err != nil || snap.Project.Name != "taken" {
			t.Fatalf("NewProject = %+v, %v", snap.Project, err)
		}
		if _, err := h.CreateRepo("taken"); err == nil || !strings.Contains(err.Error(), "name already exists") {
			t.Errorf("err = %v", err)
		}
		if !exists(filepath.Join(parent, "taken", "README.md")) {
			t.Error("the local folder was removed")
		}
		h.GH.CreateRepoErr = nil
		if snap, err := h.CreateRepo("taken"); err != nil || snap.Project.Name != "taken" {
			t.Errorf("retry = %+v, %v", snap.Project, err)
		}
		if h.GH.Calls("repo create taken --public --source . --push") != 2 {
			t.Errorf("gh calls = %q", h.GH.CallLog())
		}
	})

	t.Run("an unknown project has no repo to create", func(t *testing.T) {
		if _, err := h.CreateRepo("missing"); err == nil {
			t.Error("CreateRepo of an unknown project succeeded")
		}
	})
}
