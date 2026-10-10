package stats

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/activity"
)

const (
	mine  = "aaaaaaaa"
	other = "bbbbbbbb"
)

func loaded(t *testing.T, dir string) *Log {
	t.Helper()
	l := NewLog(dir, mine)
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
	if len(files) != 1 || filepath.Base(files[0]) != time.Now().Format(time.DateOnly)+"."+mine+".jsonl" {
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
	l := NewLog(dir, mine)
	l.Record("proj", ev(activity.Prompt, time.Now()))
	l.Load()
	l.Record("proj", ev(activity.Prompt, time.Now()))
	if got := l.Days("proj")[time.Now().Format(time.DateOnly)].Prompts; got != 2 {
		t.Errorf("prompts = %d, want 2", got)
	}
}

func TestLogOfNoProjectIsEmpty(t *testing.T) {
	if got := loaded(t, t.TempDir()).Days("none"); len(got) != 0 {
		t.Errorf("days = %v", got)
	}
}

func TestLogGzipsDaysBeforeYesterday(t *testing.T) {
	dir := t.TempDir()
	folder := filepath.Join(dir, "proj", "events")
	if err := os.MkdirAll(folder, 0o700); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	write := func(ago int, name string) {
		t.Helper()
		line := fmt.Sprintf(`{"at":%d,"kind":"prompt"}`+"\n", now.AddDate(0, 0, -ago).UnixMilli())
		if err := os.WriteFile(filepath.Join(folder, name), []byte(line), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	day := func(ago int) string { return now.AddDate(0, 0, -ago).Format(time.DateOnly) }
	write(3, day(3)+"."+mine+".jsonl")
	write(2, day(2)+"."+mine+".jsonl")
	write(1, day(1)+"."+mine+".jsonl")
	write(0, day(0)+"."+mine+".jsonl")

	count := func() int {
		t.Helper()
		n := 0
		for _, d := range loaded(t, dir).Days("proj") {
			n += d.Prompts
		}
		return n
	}
	if n := count(); n != 4 {
		t.Errorf("prompts = %d, want 4", n)
	}
	var names []string
	entries, _ := os.ReadDir(folder)
	for _, e := range entries {
		names = append(names, e.Name())
	}
	want := []string{day(3) + "." + mine + ".jsonl.gz", day(2) + "." + mine + ".jsonl.gz", day(1) + "." + mine + ".jsonl", day(0) + "." + mine + ".jsonl"}
	slices.Sort(want)
	if !slices.Equal(names, want) {
		t.Errorf("files = %v, want %v", names, want)
	}

	write(3, day(3)+"."+mine+".jsonl") // as if a gzip stopped before it removed the plain file
	if n := count(); n != 4 {
		t.Errorf("prompts with a day both plain and gzipped = %d, want 4", n)
	}
}

func writeEvents(t *testing.T, dir, name string, kind string, n int, at time.Time) {
	t.Helper()
	path := filepath.Join(dir, "proj", "events", name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	line := fmt.Sprintf(`{"at":%d,"kind":%q}`+"\n", at.UnixMilli(), kind)
	if err := os.WriteFile(path, []byte(strings.Repeat(line, n)), 0o600); err != nil {
		t.Fatal(err)
	}
}

func prompts(l *Log, day string) int { return l.Days("proj")[day].Prompts }

func TestLogCountsEveryMachinesFiles(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local)
	writeEvents(t, dir, "2026-10-01.jsonl", activity.Prompt, 1, at)
	writeEvents(t, dir, "2026-10-01."+mine+".jsonl", activity.Prompt, 2, at)
	writeEvents(t, dir, "2026-10-01."+other+".jsonl", activity.Prompt, 4, at)
	if got := prompts(loaded(t, dir), "2026-10-01"); got != 7 {
		t.Errorf("prompts = %d, want 7", got)
	}
}

func TestLogSkipsNamesThatAreNotEventFiles(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local)
	for _, name := range []string{"2026-10-01 2.jsonl", "2026-10-01." + other + " 2.jsonl", "2026-10-01.short.jsonl", "notes.jsonl", "2026-10-01.jsonl.bak"} {
		writeEvents(t, dir, name, activity.Prompt, 1, at)
	}
	l := loaded(t, dir)
	if got := l.Days("proj"); len(got) != 0 {
		t.Errorf("days = %+v, want none", got)
	}
	n := 0
	l.Since("proj", "2000-01-01", func(activity.Event) { n++ })
	if n != 0 {
		t.Errorf("Since read %d events from names that are not event files", n)
	}
}

func TestLogReadsFilesThatChangeAfterLoad(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local)
	l := loaded(t, dir)
	if got := prompts(l, "2026-10-01"); got != 0 {
		t.Fatalf("prompts = %d, want 0", got)
	}
	name := "2026-10-01." + other + ".jsonl"
	writeEvents(t, dir, name, activity.Prompt, 1, at)
	if got := prompts(l, "2026-10-01"); got != 1 {
		t.Errorf("prompts after a file synced in = %d, want 1", got)
	}
	writeEvents(t, dir, name, activity.Prompt, 3, at)
	if got := prompts(l, "2026-10-01"); got != 3 {
		t.Errorf("prompts after the file changed = %d, want 3", got)
	}
	if err := os.Remove(filepath.Join(dir, "proj", "events", name)); err != nil {
		t.Fatal(err)
	}
	if got := prompts(l, "2026-10-01"); got != 0 {
		t.Errorf("prompts after the file went = %d, want 0", got)
	}
}

func TestLogCountsAnIssueClosedByTwoMachinesOnce(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local)
	line := fmt.Sprintf(`{"at":%d,"kind":"pr_merged","issue":7,"ms":100}`+"\n", at.UnixMilli())
	for _, m := range []string{mine, other} {
		path := filepath.Join(dir, "proj", "events", "2026-10-01."+m+".jsonl")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	d := loaded(t, dir).Days("proj")["2026-10-01"]
	if d.PRsMerged != 2 || d.LeadMs != 200 || !slices.Equal(d.ClosedIssues, []int{7}) {
		t.Errorf("day = %+v", d)
	}
}

func TestLogGzipsOnlyItsOwnFiles(t *testing.T) {
	dir := t.TempDir()
	at := time.Now().AddDate(0, 0, -5)
	day := at.Format(time.DateOnly)
	names := []string{day + "." + mine + ".jsonl", day + "." + other + ".jsonl", day + ".jsonl"}
	for _, name := range names {
		writeEvents(t, dir, name, activity.Prompt, 1, at)
	}
	l := loaded(t, dir)
	entries, _ := os.ReadDir(filepath.Join(dir, "proj", "events"))
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	want := []string{day + "." + mine + ".jsonl.gz", day + "." + other + ".jsonl", day + ".jsonl"}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
	if n := prompts(l, day); n != 3 {
		t.Errorf("prompts = %d, want 3", n)
	}
}
