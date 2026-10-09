package stats

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestActivityIsRecordedPerDay(t *testing.T) {
	dir := t.TempDir()
	a := NewActivity(dir)
	day1 := time.Date(2026, 10, 8, 23, 59, 0, 0, time.Local)
	day2 := time.Date(2026, 10, 9, 0, 1, 0, 0, time.Local)
	const id = "proj/aaaaaaaa"
	a.Prompt(id, day1)
	a.Prompt(id, day2)
	a.Prompt(id, day2.UTC())
	a.Session(id, true, day2)
	a.Session(id, false, day2)
	a.Worked(id, 1500, day2)
	a.Worked(id, 500, day2)
	a.Worked(id, -5, day2)
	a.Worked(id, 0, day1)
	a.Prompt("proj/bbbbbbbb", day1)
	a.Prompt("other/aaaaaaaa", day1)

	got, err := a.Days("proj")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Day{
		"2026-10-08": {Prompts: 2},
		"2026-10-09": {Prompts: 2, Sessions: 2, IssueSessions: 1, WorkMs: 2000},
	}
	if len(got) != len(want) || got["2026-10-08"] != want["2026-10-08"] || got["2026-10-09"] != want["2026-10-09"] {
		t.Errorf("days = %+v, want %+v", got, want)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "proj", "activity.json"))
	if err != nil {
		t.Fatal(err)
	}
	wantFile := `{"2026-10-08":{"prompts":2,"sessions":0,"issueSessions":0,"workMs":0},"2026-10-09":{"prompts":2,"sessions":2,"issueSessions":1,"workMs":2000}}`
	if string(raw) != wantFile {
		t.Errorf("file = %s, want %s", raw, wantFile)
	}
	if _, err := os.Stat(filepath.Join(dir, "other", "activity.json")); err != nil {
		t.Errorf("other project's file: %v", err)
	}
}

func TestActivityOfNoProjectIsEmpty(t *testing.T) {
	got, err := NewActivity(t.TempDir()).Days("none")
	if err != nil || len(got) != 0 {
		t.Errorf("days = %v, %v", got, err)
	}
}

func TestActivityKeepsAFileItCannotRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proj", "activity.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := NewActivity(dir)
	a.Prompt("proj/aaaaaaaa", time.Now())
	if b, _ := os.ReadFile(path); string(b) != "{broken" {
		t.Errorf("file = %s", b)
	}
	if _, err := a.Days("proj"); err == nil {
		t.Error("a broken file read without an error")
	}
}
