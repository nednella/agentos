package stats

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/activity"
)

func loaded(t *testing.T, dir string) *Log {
	t.Helper()
	l := NewLog(dir)
	l.Load()
	return l
}

func ev(kind string, at time.Time) activity.Event {
	return activity.Event{At: at.UnixMilli(), Kind: kind}
}

func TestLogRollsUpEventsByDay(t *testing.T) {
	dir := t.TempDir()
	l := loaded(t, dir)
	day1 := time.Date(2026, 10, 8, 23, 59, 0, 0, time.Local)
	day2 := time.Date(2026, 10, 9, 0, 1, 0, 0, time.Local)
	l.Record("proj", ev(activity.Prompt, day1))
	l.Record("proj", ev(activity.Prompt, day2))
	l.Record("proj", activity.Event{At: day2.UnixMilli(), Kind: activity.SessionStart, Issue: 4})
	l.Record("proj", ev(activity.SessionStart, day2))
	l.Record("proj", activity.Event{At: day2.UnixMilli(), Kind: activity.SessionEnd, Ms: 600})
	l.Record("proj", activity.Event{At: day2.UnixMilli(), Kind: activity.Worked, Ms: 1500})
	l.Record("proj", ev(activity.PROpened, day2))
	l.Record("proj", activity.Event{At: day2.UnixMilli(), Kind: activity.PRMerged, Issue: 7, Ms: 100})
	l.Record("proj", activity.Event{At: day2.UnixMilli(), Kind: activity.PRMerged, Issue: 7, Ms: 300})
	l.Record("proj", ev(activity.PRClosed, day2))
	l.Record("proj", ev(activity.IssueFiled, day2))
	l.Record("proj", ev("a kind from a later version", day2))
	l.Record("other", ev(activity.Prompt, day1))

	want := map[string]Day{
		"2026-10-08": {Prompts: 1},
		"2026-10-09": {Prompts: 1, Sessions: 2, IssueSessions: 1, Ended: 1, SessionMs: 600, WorkMs: 1500,
			PRsOpened: 1, PRsMerged: 2, PRsClosed: 1, LeadMs: 400, IssuesFiled: 1, ClosedIssues: []int{7}},
	}
	check := func(l *Log) {
		t.Helper()
		got := l.Days("proj")
		if len(got) != len(want) {
			t.Fatalf("days = %+v", got)
		}
		for day, w := range want {
			g := got[day]
			if !reflect.DeepEqual(g, w) {
				t.Errorf("%s = %+v, want %+v", day, g, w)
			}
		}
	}
	check(l)
	check(loaded(t, dir))

	files, _ := filepath.Glob(filepath.Join(dir, "proj", "events", "*.jsonl"))
	if len(files) != 1 || filepath.Base(files[0]) != time.Now().Format(time.DateOnly)+".jsonl" {
		t.Errorf("files = %v: events go to the file of the day they are written", files)
	}
}

func TestLogSkipsLinesThatDoNotParse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "proj", "events", "2026-10-01.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	prompt := fmt.Sprintf(`{"at":%d,"kind":"prompt"}`, time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local).UnixMilli())
	body := prompt + "\nnot json\n" + prompt + "\n" + prompt[:10]
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := loaded(t, dir).Days("proj")["2026-10-01"].Prompts; got != 2 {
		t.Errorf("prompts = %d, want 2", got)
	}
}

func TestLogCountsAnEventRecordedBeforeLoadOnce(t *testing.T) {
	dir := t.TempDir()
	l := NewLog(dir)
	l.Record("proj", ev(activity.Prompt, time.Now()))
	l.Load()
	l.Record("proj", ev(activity.Prompt, time.Now()))
	if got := l.Days("proj")[time.Now().Format(time.DateOnly)].Prompts; got != 2 {
		t.Errorf("prompts = %d, want 2", got)
	}
}

func TestLogReadsActivityBeforeTheFirstEvent(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "proj"), 0o700); err != nil {
		t.Fatal(err)
	}
	old := `{"2026-10-07":{"prompts":3,"sessions":1,"issueSessions":1,"workMs":50},"2026-10-08":{"prompts":4}}`
	if err := os.WriteFile(filepath.Join(dir, "proj", "activity.json"), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	l := NewLog(dir)
	l.Record("proj", ev(activity.Prompt, time.Date(2026, 10, 8, 9, 0, 0, 0, time.Local)))
	l.Load()
	got := l.Days("proj")
	if d := got["2026-10-07"]; d.Prompts != 3 || d.Sessions != 1 || d.IssueSessions != 1 || d.WorkMs != 50 {
		t.Errorf("a day before the log = %+v", d)
	}
	if d := got["2026-10-08"]; d.Prompts != 1 {
		t.Errorf("the log's first day = %+v: activity.json counted on top of the log", d)
	}
}

func TestLogOfNoProjectIsEmpty(t *testing.T) {
	if got := loaded(t, t.TempDir()).Days("none"); len(got) != 0 {
		t.Errorf("days = %v", got)
	}
}
