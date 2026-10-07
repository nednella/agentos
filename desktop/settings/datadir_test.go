package settings_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/internal/project"
)

// The test binary runs from no .app bundle, so every change ends on this.
const noRelaunch = "agentos runs from no .app bundle, so it cannot relaunch"

func write(t *testing.T, path, data string, perm os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), perm); err != nil {
		t.Fatal(err)
	}
}

func configDataDir(t *testing.T, h *apptest.Harness) string {
	t.Helper()
	cfg, err := project.Load(h.Conf)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.DataDir
}

func TestDataDirFixedByEnvironment(t *testing.T) {
	h := apptest.New(t)
	if got := h.Settings(); !got.DataFixed || got.DataDir != filepath.Join(h.State, "data") {
		t.Fatalf("settings = %+v", got)
	}
	h.Rec.SetPick(t.TempDir())
	if _, err := h.PickDataDir(); err == nil {
		t.Error("the picker opened while AGENTOS_DEV_DATA_DIR sets the folder")
	}
	if err := h.SetDataDir(t.TempDir(), false); err == nil || configDataDir(t, h) != "" {
		t.Errorf("SetDataDir = %v, data_dir %q", err, configDataDir(t, h))
	}
}

func TestPickDataDir(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{DataDirConfig: true})
	if got := h.Settings(); got.DataFixed || got.DataDir != filepath.Join(h.State, "data") {
		t.Fatalf("settings = %+v", got)
	}
	if got, err := h.PickDataDir(); err != nil || got.Dir != "" {
		t.Errorf("cancelled pick = %+v, %v", got, err)
	}

	empty := t.TempDir()
	write(t, filepath.Join(empty, ".DS_Store"), "", 0o600)
	h.Rec.SetPick(empty)
	if got, err := h.PickDataDir(); err != nil || got.Dir != empty || !got.Empty {
		t.Errorf("empty folder = %+v, %v", got, err)
	}

	full := t.TempDir()
	write(t, filepath.Join(full, "main", "notes.json"), "[]", 0o600)
	h.Rec.SetPick(full)
	if got, err := h.PickDataDir(); err != nil || got.Dir != full || got.Empty {
		t.Errorf("full folder = %+v, %v", got, err)
	}
	if configDataDir(t, h) != filepath.Join(h.State, "data") {
		t.Error("picking a folder changed the config")
	}
}

func TestSetDataDirCopiesTheData(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{DataDirConfig: true})
	old := filepath.Join(h.State, "data")
	files := map[string]string{
		"main/notes.json":            `[{"id":"n1"}]`,
		"main/notes-media/a.png":     "png",
		"main/evidence/tok/shot.png": "shot",
		"main/stats.jsonl":           "{}\n",
		"main/digest.json":           "{}",
		"other/notes.json":           "[]",
	}
	for name, data := range files {
		write(t, filepath.Join(old, name), data, 0o600)
	}
	write(t, filepath.Join(old, "main", "prs.json"), "{}", 0o600) // the local folder's, when both are one folder

	to := t.TempDir()
	if err := h.SetDataDir(to, true); err == nil || !strings.Contains(err.Error(), noRelaunch) {
		t.Fatalf("SetDataDir = %v", err)
	}
	for name, data := range files {
		got, err := os.ReadFile(filepath.Join(to, name))
		if err != nil || string(got) != data {
			t.Errorf("%s = %q, %v", name, got, err)
		}
		if info, err := os.Stat(filepath.Join(to, name)); err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("%s mode = %v, %v", name, info.Mode(), err)
		}
		if _, err := os.Stat(filepath.Join(old, name)); err != nil {
			t.Errorf("the old %s went: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(to, "main", "prs.json")); err == nil {
		t.Error("the local folder's prs.json was copied")
	}
	if got := configDataDir(t, h); got != to {
		t.Errorf("data_dir = %q", got)
	}
	if got := h.Restart(t).Settings().DataDir; got != to {
		t.Errorf("data folder after restart = %q", got)
	}
}

func TestSetDataDirUsesAFullFolderAsItIs(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{DataDirConfig: true})
	write(t, filepath.Join(h.State, "data", "main", "notes.json"), "mine", 0o600)
	to := t.TempDir()
	write(t, filepath.Join(to, "main", "notes.json"), "theirs", 0o600)

	if err := h.SetDataDir(to, true); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Fatalf("copying into a full folder = %v", err)
	}
	if got := configDataDir(t, h); got != filepath.Join(h.State, "data") {
		t.Errorf("a refused copy changed data_dir to %q", got)
	}

	if err := h.SetDataDir(to, false); err == nil || !strings.Contains(err.Error(), noRelaunch) {
		t.Fatalf("SetDataDir = %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(to, "main", "notes.json")); string(got) != "theirs" {
		t.Errorf("the folder's notes = %q", got)
	}
	if got := configDataDir(t, h); got != to {
		t.Errorf("data_dir = %q", got)
	}
}

func TestSetDataDirRejectsTheCurrentFolder(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{DataDirConfig: true})
	if err := h.SetDataDir(filepath.Join(h.State, "data")+"/", false); err == nil || !strings.Contains(err.Error(), "already") {
		t.Errorf("SetDataDir(current) = %v", err)
	}
}

func TestFailedCopyLeavesTheFolderEmpty(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{DataDirConfig: true})
	write(t, filepath.Join(h.State, "data", "main", "notes.json"), "[]", 0o600)
	write(t, filepath.Join(h.State, "data", "main", "stats.jsonl"), "{}\n", 0o000)
	to := t.TempDir()
	write(t, filepath.Join(to, ".DS_Store"), "", 0o600)

	if err := h.SetDataDir(to, true); err == nil || strings.Contains(err.Error(), noRelaunch) {
		t.Fatalf("SetDataDir = %v", err)
	}
	if entries, _ := os.ReadDir(to); len(entries) != 1 || entries[0].Name() != ".DS_Store" {
		t.Errorf("the folder kept %v", entries)
	}
	if got := configDataDir(t, h); got != filepath.Join(h.State, "data") {
		t.Errorf("a failed copy changed data_dir to %q", got)
	}
}
