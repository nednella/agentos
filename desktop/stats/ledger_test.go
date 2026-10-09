package stats

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/project"
)

type fakeProjects []project.Project

func (f fakeProjects) List() []project.Project { return f }

type fakeRepos map[string]string // by folder

func (f fakeRepos) Repo(_ context.Context, dir string) string { return f[dir] }

// fakeGH answers every search with its count, and fails for the repos in fail.
type fakeGH struct {
	mu    sync.Mutex
	calls []string
	fail  map[string]string
}

func (f *fakeGH) run(_ context.Context, dir, name string, args ...string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, dir+": "+name+" "+strings.Join(args, " "))
	for repo, msg := range f.fail {
		if strings.Contains(args[len(args)-1], "repo:"+repo+" ") {
			return nil, errors.New(msg)
		}
	}
	return []byte(`{"data":{"merged":{"issueCount":4},"closedPRs":{"issueCount":1},"issuesClosed":{"issueCount":5},"issuesOpen":{"issueCount":2}}}`), nil
}

func TestLedger(t *testing.T) {
	now := time.Date(2026, 10, 9, 15, 0, 0, 0, time.Local)
	day := func(ago int) time.Time { return now.AddDate(0, 0, -ago) }
	act := NewActivity(t.TempDir())
	projects := fakeProjects{{Name: "Alpha", Dir: "/a"}, {Name: "Beta", Dir: "/b"}, {Name: "Gamma", Dir: "/g"}, {Name: "Delta", Dir: "/d"}}
	record := func(p string, ago, prompts int, sessions int, issue bool, workMs int64) {
		id := strings.ToLower(p) + "/aaaaaaaa"
		for range prompts {
			act.Prompt(id, day(ago))
		}
		for range sessions {
			act.Session(id, issue, day(ago))
		}
		act.Worked(id, workMs, day(ago))
	}
	record("Alpha", 0, 2, 1, true, 1000)
	record("Alpha", 10, 1, 1, false, 500)
	record("Alpha", 100, 5, 2, true, 2000)
	record("Beta", 3, 3, 1, true, 4000)
	record("Beta", 400, 9, 3, false, 100)
	record("Delta", 3, 3, 0, false, 0)

	repos := fakeRepos{"/a": "o/alpha", "/b": "o/beta", "/g": "o/gamma"}
	gh := &fakeGH{fail: map[string]string{"o/gamma": "gh api: exit status 1: boom\nsecond line"}}
	books := NewBooks(act, projects, repos, gh.run)
	build := func(days int) Ledger {
		t.Helper()
		led, err := books.Ledger(context.Background(), days, now)
		if err != nil {
			t.Fatal(err)
		}
		return led
	}

	t.Run("thirty days", func(t *testing.T) {
		led := build(30)
		if led.Days != 30 || led.Since != day(400).Format(time.DateOnly) || led.GitHub != "asking GitHub about o/gamma: gh api: exit status 1: boom" {
			t.Errorf("days %d, since %s, github %q", led.Days, led.Since, led.GitHub)
		}
		var names []string
		for _, r := range led.Projects {
			names = append(names, fmt.Sprintf("%s:%d", r.Project, r.Prompts))
		}
		if want := []string{"Alpha:3", "Beta:3", "Delta:3", "Gamma:0"}; !slices.Equal(names, want) {
			t.Errorf("projects = %v, want %v", names, want)
		}
		alpha := led.Projects[0]
		if alpha.Sessions != 2 || alpha.IssueSessions != 1 || alpha.WorkMs != 1500 || *alpha.PRsMerged != 4 || *alpha.PRsClosed != 1 || *alpha.IssuesClosed != 5 || *alpha.IssuesOpen != 2 {
			t.Errorf("alpha = %+v", alpha)
		}
		for _, r := range led.Projects[2:] {
			if r.PRsMerged != nil || r.PRsClosed != nil || r.IssuesClosed != nil || r.IssuesOpen != nil {
				t.Errorf("%s has GitHub numbers: %+v", r.Project, r)
			}
		}
		tot := led.Totals
		if tot.Project != "" || tot.Prompts != 9 || tot.Sessions != 3 || tot.IssueSessions != 2 || tot.WorkMs != 5500 || *tot.PRsMerged != 8 || *tot.IssuesOpen != 4 {
			t.Errorf("totals = %+v", tot)
		}
	})

	t.Run("year and lifetime", func(t *testing.T) {
		if year := build(365).Totals; year.Prompts != 14 || year.Sessions != 5 {
			t.Errorf("year totals = %+v", year)
		}
		life := build(0)
		if life.Days != 0 || life.Totals.Prompts != 23 || life.Totals.Sessions != 8 || life.Totals.WorkMs != 7600 {
			t.Errorf("lifetime = %+v", life)
		}
	})

	t.Run("heat", func(t *testing.T) {
		led := build(30)
		if len(led.Heat) != 364 || led.Heat[363].Day != "2026-10-09" || led.Heat[0].Day != day(363).Format(time.DateOnly) {
			t.Fatalf("heat = %d days, %v .. %v", len(led.Heat), led.Heat[0], led.Heat[len(led.Heat)-1])
		}
		counts := map[string]int{}
		sum := 0
		for _, d := range led.Heat {
			counts[d.Day] = d.Count
			sum += d.Count
		}
		if counts["2026-10-09"] != 2 || counts[day(3).Format(time.DateOnly)] != 6 || counts[day(100).Format(time.DateOnly)] != 5 || sum != 14 {
			t.Errorf("counts = %v, sum %d", counts, sum)
		}
	})

	t.Run("the query", func(t *testing.T) {
		gh.mu.Lock()
		gh.calls = nil
		gh.mu.Unlock()
		build(30)
		from := day(29).Format(time.DateOnly)
		want := `/a: gh api graphql -f query={ merged: search(query:"repo:o/alpha is:pr is:merged author:@me merged:>=` + from + `", type:ISSUE, first:0){issueCount} ` +
			`closedPRs: search(query:"repo:o/alpha is:pr is:closed is:unmerged author:@me closed:>=` + from + `", type:ISSUE, first:0){issueCount} ` +
			`issuesClosed: search(query:"repo:o/alpha is:issue is:closed closed:>=` + from + `", type:ISSUE, first:0){issueCount} ` +
			`issuesOpen: search(query:"repo:o/alpha is:issue is:open", type:ISSUE, first:0){issueCount} }`
		if !slices.Contains(gh.calls, want) || len(gh.calls) != 3 {
			t.Errorf("calls = %q, want one to be %q", gh.calls, want)
		}
		gh.mu.Lock()
		gh.calls = nil
		gh.mu.Unlock()
		build(0)
		for _, c := range gh.calls {
			if strings.Contains(c, ">=") {
				t.Errorf("lifetime call has a date: %s", c)
			}
		}
	})
}

func TestLedgerWithoutRecordOrRepo(t *testing.T) {
	now := time.Date(2026, 10, 9, 15, 0, 0, 0, time.Local)
	gh := &fakeGH{}
	books := NewBooks(NewActivity(t.TempDir()), fakeProjects{{Name: "Solo", Dir: "/s"}}, fakeRepos{}, gh.run)
	led, err := books.Ledger(context.Background(), 30, now)
	if err != nil {
		t.Fatal(err)
	}
	if led.Since != "" || led.GitHub != "" || len(led.Heat) != 364 || len(led.Projects) != 1 || led.Totals.PRsMerged != nil || len(gh.calls) != 0 {
		t.Errorf("since %q, github %q, projects %d, calls %q", led.Since, led.GitHub, len(led.Projects), gh.calls)
	}
}

func TestLedgerReportsABrokenAnswer(t *testing.T) {
	now := time.Date(2026, 10, 9, 15, 0, 0, 0, time.Local)
	bad := func(context.Context, string, string, ...string) ([]byte, error) { return []byte("not json"), nil }
	books := NewBooks(NewActivity(t.TempDir()), fakeProjects{{Name: "Solo", Dir: "/s"}}, fakeRepos{"/s": "o/s"}, bad)
	led, err := books.Ledger(context.Background(), 30, now)
	if err != nil {
		t.Fatal(err)
	}
	if led.GitHub == "" || led.Projects[0].PRsMerged != nil {
		t.Errorf("ledger = %+v", led)
	}
}
