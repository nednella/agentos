package projects_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/desktop/internal/warn"
	"github.com/nednella/agentos/desktop/issues"
	"github.com/nednella/agentos/desktop/projects"
	"github.com/nednella/agentos/internal/project"
)

func writeConfig(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func loadRegistry(t *testing.T, path string) *projects.Registry {
	t.Helper()
	cfg, err := project.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return projects.NewRegistry(path, cfg)
}

func TestRegistryKeepsHandEditOnSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, "projects:\n  - name: p\n    directory: "+dir+"\n")
	r := loadRegistry(t, path)

	writeConfig(t, path, "projects:\n  - name: p\n    directory: "+dir+"\n    queue_sections:\n      - name: Inbox\n        actions:\n          - {name: Plan, command: \"/plan {n}\", model: opus}\n")
	if err := r.SetDigestSchedule("p", "off"); err != nil {
		t.Fatal(err)
	}

	cfg, err := project.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Projects[0]
	if p.Digest != "off" || len(p.QueueSections) != 1 || p.QueueSections[0].Actions[0].Name != "Plan" {
		t.Errorf("saved project = %+v; want the hand edit and the setting", p)
	}
	if got := r.List()[0]; len(got.QueueSections) != 1 {
		t.Errorf("registry project = %+v; want the hand edit", got)
	}
}

func TestRegistryReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, "projects:\n  - name: p\n    directory: "+dir+"\n")
	r := loadRegistry(t, path)
	if changed, err := r.Reload(); changed || err != nil {
		t.Errorf("Reload of an unchanged file = %v, %v", changed, err)
	}

	writeConfig(t, path, "projects:\n  - name: renamed\n    directory: "+dir+"\n")
	if changed, err := r.Reload(); !changed || err != nil {
		t.Fatalf("Reload after an edit = %v, %v", changed, err)
	}
	if got := r.List(); len(got) != 1 || got[0].Name != "renamed" {
		t.Errorf("List = %+v, want the edited name", got)
	}
	if changed, err := r.Reload(); changed || err != nil {
		t.Errorf("second Reload = %v, %v", changed, err)
	}
}

func TestRegistryRefusesToSaveOverBadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(t.TempDir(), "config.yaml")
	writeConfig(t, path, "projects:\n  - name: p\n    directory: "+dir+"\n")
	r := loadRegistry(t, path)

	bad := "projects: [\n"
	writeConfig(t, path, bad)
	if _, err := r.Reload(); err == nil {
		t.Error("Reload of a broken file succeeded")
	}
	if got := r.List(); len(got) != 1 || got[0].Name != "p" {
		t.Errorf("List after a broken file = %+v, want the old config", got)
	}
	if err := r.SetDigestSchedule("p", "off"); err == nil {
		t.Error("saving over a broken file succeeded")
	}
	if data, _ := os.ReadFile(path); string(data) != bad {
		t.Errorf("file = %q, want it left as written", data)
	}
}

func TestAppFollowsConfigEdits(t *testing.T) {
	h := newHarness(t)
	data, err := os.ReadFile(h.Conf)
	if err != nil {
		t.Fatal(err)
	}
	writeConfig(t, h.Conf, strings.Replace(string(data), "name: Inbox", "name: Triage", 1))
	eventually(t, "the queue to show the renamed section", func() bool {
		list, err := h.Issues(false)
		return err == nil && slices.ContainsFunc(list, func(is issues.Issue) bool { return is.Section == "Triage" })
	})

	writeConfig(t, h.Conf, "projects: [\n")
	eventually(t, "a config warning", func() bool {
		return slices.ContainsFunc(h.Rec.Warnings(), func(w warn.Warning) bool { return w.Source == "config" && w.Message != "" })
	})
	writeConfig(t, h.Conf, string(data))
	eventually(t, "the config warning to clear", func() bool {
		w := h.Rec.Warnings()
		return len(w) > 0 && w[len(w)-1] == warn.Warning{Source: "config"}
	})
}
