package issues_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/issues"
	"github.com/nednella/agentos/internal/project"
)

func TestIssues(t *testing.T) {
	h := newHarness(t)

	got, err := h.Issues(false)
	if err != nil {
		t.Fatal(err)
	}
	var lanes []string
	for _, is := range got {
		lanes = append(lanes, fmt.Sprintf("%d:%s:%s", is.Number, is.Lane, is.Type))
	}
	want := []string{"7:ready:bug", "8:plan:feature", "9:you:", "10:idea:chore", "11:inbox:", "12:idea:"}
	if !slices.Equal(lanes, want) {
		t.Errorf("issues = %v, want %v", lanes, want)
	}
	first := got[0]
	if first.Author != "ned" || !slices.Equal(first.Assignees, []string{"ned", "amy"}) || !slices.Equal(first.Labels, []string{"ready", "type:bug"}) ||
		first.CreatedAt != time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC).UnixMilli() || first.UpdatedAt <= first.CreatedAt {
		t.Errorf("issue 7 = %+v", first)
	}
	if raw, _ := json.Marshal(got[1]); !strings.Contains(string(raw), `"assignees":[]`) || !strings.Contains(string(raw), `"labels":["needs-plan","type:feature"]`) || !strings.Contains(string(raw), `"author":""`) {
		t.Errorf("issue 8 json = %s", raw)
	}
	if repo := h.Snapshot().Project.Repo; repo != "acme/widgets" {
		t.Errorf("repo = %q", repo)
	}
	if _, err := h.Issues(false); err != nil || h.GH.IssueCalls() != 1 {
		t.Errorf("cached call: err=%v, gh issue list ran %d times", err, h.GH.IssueCalls())
	}
	if _, err := h.Issues(true); err != nil || h.GH.IssueCalls() != 2 {
		t.Errorf("refresh: err=%v, gh issue list ran %d times", err, h.GH.IssueCalls())
	}

	s, err := h.StartIssue(7)
	if err != nil {
		t.Fatal(err)
	}
	if s.Issue != 7 || s.Title != "#7 Fix the thing" {
		t.Errorf("session = %+v", s)
	}
	eventually(t, "/ship typed", func() bool { return strings.Contains(h.Pane(t, s.ID), "/ship 7") })
	again, err := h.StartIssue(7)
	if err != nil || again.ID != s.ID {
		t.Errorf("second start = %+v, %v", again, err)
	}
	got, _ = h.Issues(false)
	if got[0].SessionID != s.ID || got[1].SessionID != "" {
		t.Errorf("session ids = %q %q", got[0].SessionID, got[1].SessionID)
	}
	if _, err := h.StartIssue(999); err == nil {
		t.Error("starting an unknown issue succeeded")
	}
	plan, err := h.StartIssue(8)
	if err != nil {
		t.Fatal(err)
	}
	issuesEvent := func(number int) string {
		list, _ := h.Rec.Last("issues").([]issues.Issue)
		for _, is := range list {
			if is.Number == number {
				return is.SessionID
			}
		}
		return "?"
	}
	eventually(t, "issues event with the session", func() bool { return issuesEvent(7) == s.ID })
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "issues event after the session ends", func() bool { return issuesEvent(7) == "" })
	eventually(t, "/investigate typed", func() bool { return strings.Contains(h.Pane(t, plan.ID), "/investigate 8") })
}

func TestIssuesWithoutRepo(t *testing.T) {
	gh := &apptest.FakeGH{Repo: errors.New("not a repo")}
	is := issues.New(gh.Run, nil, nil)
	list, err := is.List(context.Background(), project.Project{Dir: "/x"}, false)
	if err != nil || list == nil || len(list) != 0 {
		t.Errorf("List = %v, %v", list, err)
	}
	if repo := is.Repo(context.Background(), "/x"); repo != "" {
		t.Errorf("repo = %q", repo)
	}
}

func TestInboxAndIdeaCommands(t *testing.T) {
	h := newHarness(t)
	s, err := h.StartIssue(11) // no labels: the inbox lane
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "/investigate typed", func() bool { return strings.Contains(h.Pane(t, s.ID), "/investigate 11") })
	idea, err := h.StartIssue(10) // idea lane: no command by default
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(600 * time.Millisecond)
	if pane := h.Pane(t, idea.ID); strings.Contains(pane, "/") && strings.Contains(pane, "investigate") {
		t.Errorf("an idea got a command:\n%s", pane)
	}
}
