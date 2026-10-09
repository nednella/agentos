package update_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/update"
)

const changelog = `# Changelog

## [0.14.0](https://github.com/nednella/agentos/compare/v0.13.0...v0.14.0) (2026-10-07)


### Features

* **desktop:** let agents close their browser window ([a796cf6](https://github.com/nednella/agentos/commit/a796cf6)), closes [#218](https://github.com/nednella/agentos/issues/218)


### Bug Fixes

* widen settings panel so every row shows ([29a0500](https://github.com/nednella/agentos/commit/29a0500))
* keep the Mac awake while sessions work ([#88](https://github.com/nednella/agentos/issues/88)) ([0313ac2](https://github.com/nednella/agentos/commit/0313ac2))

## 0.1.0 (2026-10-05)


### Features

* **ui:** first window ([1111111](https://github.com/nednella/agentos/commit/1111111))
`

func TestParseChangelogReadsEachReleaseAndItsChanges(t *testing.T) {
	want := []update.Release{
		{Version: "0.14.0", Date: "2026-10-07", Sections: []update.Section{
			{Title: "Features", Changes: []update.Change{{Scope: "desktop", Text: "let agents close their browser window", Issue: 218, URL: "https://github.com/nednella/agentos/issues/218"}}},
			{Title: "Bug Fixes", Changes: []update.Change{
				{Text: "widen settings panel so every row shows"},
				{Text: "keep the Mac awake while sessions work", Issue: 88, URL: "https://github.com/nednella/agentos/issues/88"},
			}},
		}},
		{Version: "0.1.0", Date: "2026-10-05", Sections: []update.Section{
			{Title: "Features", Changes: []update.Change{{Scope: "ui", Text: "first window"}}},
		}},
	}
	if got := update.ParseChangelog(changelog); !reflect.DeepEqual(got, want) {
		t.Errorf("ParseChangelog =\n%+v\nwant\n%+v", got, want)
	}
}

func versions(t *testing.T, u *update.Updater) []string {
	t.Helper()
	notes, err := u.PatchNotes()
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, r := range notes {
		out = append(out, r.Version)
	}
	return out
}

func TestPatchNotesShowTheReleasesSinceTheLastRunOnce(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{UpdateTick: time.Hour, Version: "0.14.0"})
	if err := os.WriteFile(filepath.Join(h.State, "seen-version"), []byte("0.12.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := versions(t, h.App.Updates()); !reflect.DeepEqual(got, []string{"0.14.0", "0.13.0"}) {
		t.Errorf("first launch after the update = %v", got)
	}
	if got := versions(t, h.App.Updates()); got != nil {
		t.Errorf("second launch = %v, want none", got)
	}
}

func TestPatchNotesOnTheFirstRunShowOnlyTheRunningRelease(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{UpdateTick: time.Hour, Version: "0.14.0"})
	if got := versions(t, h.App.Updates()); !reflect.DeepEqual(got, []string{"0.14.0"}) {
		t.Errorf("first run = %v", got)
	}
}

func TestADevBuildHasNoPatchNotes(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{UpdateTick: time.Hour})
	if got := versions(t, h.App.Updates()); got != nil {
		t.Errorf("dev build = %v, want none", got)
	}
}
