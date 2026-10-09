package stats

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/project"
)

type fakeProjects []project.Project

func (f fakeProjects) List() []project.Project { return f }

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

	books := NewBooks(act, projects)
	build := func(days int) Ledger {
		t.Helper()
		led, err := books.Ledger(days, now)
		if err != nil {
			t.Fatal(err)
		}
		return led
	}

	t.Run("thirty days", func(t *testing.T) {
		led := build(30)
		if led.Days != 30 || led.Since != day(400).Format(time.DateOnly) {
			t.Errorf("days %d, since %s", led.Days, led.Since)
		}
		var names []string
		for _, r := range led.Projects {
			names = append(names, fmt.Sprintf("%s:%d", r.Project, r.Prompts))
		}
		if want := []string{"Alpha:3", "Beta:3", "Delta:3", "Gamma:0"}; !slices.Equal(names, want) {
			t.Errorf("projects = %v, want %v", names, want)
		}
		alpha := led.Projects[0]
		if alpha.Sessions != 2 || alpha.IssueSessions != 1 || alpha.WorkMs != 1500 {
			t.Errorf("alpha = %+v", alpha)
		}
		tot := led.Totals
		if tot.Project != "" || tot.Prompts != 9 || tot.Sessions != 3 || tot.IssueSessions != 2 || tot.WorkMs != 5500 {
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
}

func TestLedgerWithoutRecord(t *testing.T) {
	now := time.Date(2026, 10, 9, 15, 0, 0, 0, time.Local)
	books := NewBooks(NewActivity(t.TempDir()), fakeProjects{{Name: "Solo", Dir: "/s"}})
	led, err := books.Ledger(30, now)
	if err != nil {
		t.Fatal(err)
	}
	if led.Since != "" || len(led.Heat) != 364 || len(led.Projects) != 1 || led.Totals.Prompts != 0 {
		t.Errorf("since %q, heat %d, projects %d, totals %+v", led.Since, len(led.Heat), len(led.Projects), led.Totals)
	}
}
