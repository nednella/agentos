package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestMigrateLayout(t *testing.T) {
	root := t.TempDir()
	sync, local := filepath.Join(root, "sync"), filepath.Join(root, "local")

	// The old layout, by kind.
	put(t, filepath.Join(sync, "notes", "alpha.json"), `[{"id":"a","images":["/media/notes-media/alpha/x.png"]}]`)
	put(t, filepath.Join(sync, "notes-media", "alpha", "x.png"), "png")
	put(t, filepath.Join(sync, "stats", "alpha.jsonl"), "{\"kind\":\"finished\"}\n")
	put(t, filepath.Join(sync, "digest", "alpha.json"), `{"items":[]}`)
	put(t, filepath.Join(sync, "notes", "beta.json"), `[{"id":"old"}]`)
	put(t, filepath.Join(sync, "beta", "notes.json"), `[{"id":"new"}]`) // already in the new place
	put(t, filepath.Join(local, "evidence", "alpha", "1", "index.json"), `[{"url":"/media/evidence/alpha/1/e.png"}]`)
	put(t, filepath.Join(local, "evidence", "alpha", "1", "e.png"), "png")
	put(t, filepath.Join(local, "prs", "alpha.json"), `{"12":{"comments":3,"checks":false}}`)
	put(t, filepath.Join(local, "prs", "alpha.live.json"), `{"alpha/1":true}`)
	put(t, filepath.Join(local, "cleanups", "alpha.jsonl"), "{\"at\":1,\"status\":\"done\"}\n{\"at\":2,\"status\":\"blocked\"}\n")
	put(t, filepath.Join(local, "browser", "alpha", "Default", "Cookies"), "cookies")

	var logged []string
	logf := func(format string, args ...any) { logged = append(logged, strings.TrimSpace(format)) }
	migrateLayout(sync, local, logf)

	for path, want := range map[string]string{
		filepath.Join(sync, "alpha", "notes.json"):                     `[{"id":"a","images":["/media/alpha/notes-media/x.png"]}]`,
		filepath.Join(sync, "alpha", "notes-media", "x.png"):           "png",
		filepath.Join(sync, "alpha", "stats.jsonl"):                    "{\"kind\":\"finished\"}\n",
		filepath.Join(sync, "alpha", "digest.json"):                    `{"items":[]}`,
		filepath.Join(sync, "beta", "notes.json"):                      `[{"id":"new"}]`,
		filepath.Join(local, "alpha", "evidence", "1", "index.json"):   `[{"url":"/media/alpha/evidence/1/e.png"}]`,
		filepath.Join(local, "alpha", "evidence", "1", "e.png"):        "png",
		filepath.Join(local, "alpha", "browser", "Default", "Cookies"): "cookies",
	} {
		if got := read(t, path); got != want {
			t.Errorf("%s = %q, want %q", path, got, want)
		}
	}
	if got := read(t, filepath.Join(local, "alpha", "prs.json")); got != `{"acks":{"12":{"comments":3,"checks":false}},"live":{"alpha/1":true}}` {
		t.Errorf("prs.json = %s", got)
	}
	if got := read(t, filepath.Join(local, "alpha", "cleanups.json")); !strings.HasPrefix(got, `[{"at":1,`) || !strings.Contains(got, `"at":2`) {
		t.Errorf("cleanups.json = %s", got)
	}

	// The old copy of an existing file stays; everything else old is gone.
	if read(t, filepath.Join(sync, "notes", "beta.json")) != `[{"id":"old"}]` {
		t.Error("the old beta notes were lost")
	}
	for _, gone := range []string{
		filepath.Join(sync, "notes", "alpha.json"), filepath.Join(sync, "notes-media"), filepath.Join(sync, "stats"), filepath.Join(sync, "digest"),
		filepath.Join(local, "evidence"), filepath.Join(local, "prs"), filepath.Join(local, "cleanups"), filepath.Join(local, "browser"),
	} {
		if exists(gone) {
			t.Errorf("%s is still there", gone)
		}
	}
	if len(logged) == 0 || !strings.Contains(strings.Join(logged, "\n"), "kept") {
		t.Errorf("log = %q", logged)
	}

	// A second run changes nothing.
	logged = nil
	migrateLayout(sync, local, logf)
	if got := strings.Join(logged, "\n"); strings.Contains(got, "moved") {
		t.Errorf("the second run moved things: %s", got)
	}
}

func TestMigrateKeepsFilesInTheNewPlace(t *testing.T) {
	root := t.TempDir()
	put(t, filepath.Join(root, "notes", "notes.json"), `[]`) // project "notes", new layout
	migrateLayout(root, root, func(string, ...any) {})
	if read(t, filepath.Join(root, "notes", "notes.json")) != `[]` {
		t.Error("a project named like an old folder lost its notes")
	}
}
