package issues_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/internal/warn"
	"github.com/nednella/agentos/desktop/issues"
	"github.com/nednella/agentos/internal/project"
)

func TestIssues(t *testing.T) {
	h := newHarness(t)

	got, err := h.Issues(false)
	if err != nil {
		t.Fatal(err)
	}
	var placed []string
	for _, is := range got {
		placed = append(placed, fmt.Sprintf("%d:%s:%s:%s", is.Number, is.Section, strings.Join(is.Actions, "+"), is.Type))
	}
	want := []string{"7:Ready:Work+Plan:bug", "8:Plan:Investigate:feature", "11:Inbox:Investigate:", "9::Start:", "10::Start:chore", "12::Start:"}
	if !slices.Equal(placed, want) {
		t.Errorf("issues = %v, want %v", placed, want)
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
	// The agent is bash: a sent command fails, a typed one just sits on the prompt.
	eventually(t, "/ship sent", func() bool { return strings.Contains(h.Pane(t, s.ID), "/ship: No such file or directory") })
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
		list := h.Rec.LastIssues()
		for _, is := range list {
			if is.Number == number {
				return is.SessionID
			}
		}
		return "?"
	}
	eventually(t, "issues event with the session", func() bool { return issuesEvent(7) == s.ID })
	if got := h.Rec.ProjectOf("issues"); got != "main" {
		t.Errorf("issues event project = %q", got)
	}
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "issues event after the session ends", func() bool { return issuesEvent(7) == "" })
	eventually(t, "/investigate typed", func() bool { return strings.Contains(h.Pane(t, plan.ID), "/investigate 8") })
}

func TestIssuesWithoutRepo(t *testing.T) {
	gh := &apptest.FakeGH{Repo: errors.New("gh repo view: exit status 1: not a git repository")}
	is := issues.New(gh.Run, nil, func(string, any) {})
	list, err := is.List(context.Background(), project.Project{Dir: "/x"}, false)
	if err != nil || list == nil || len(list) != 0 {
		t.Errorf("List = %v, %v", list, err)
	}
	if repo := is.Repo(context.Background(), "/x"); repo != "" {
		t.Errorf("repo = %q", repo)
	}
}

func TestIssuesDisabled(t *testing.T) {
	gh := &apptest.FakeGH{IssuesErr: errors.New("gh issue list: exit status 1: the 'acme/widgets' repository has disabled issues")}
	is := issues.New(gh.Run, nil, func(string, any) {})
	if _, err := is.List(context.Background(), project.Project{Dir: "/x"}, false); err != issues.ErrIssuesDisabled {
		t.Errorf("List error = %v, want ErrIssuesDisabled", err)
	}
}

func TestStartTypesTheChosenActionsCommand(t *testing.T) {
	h := newHarness(t)
	inbox, err := h.StartIssue(11)
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "/investigate typed", func() bool { return strings.Contains(h.Pane(t, inbox.ID), "/investigate 11") })
	plan, err := h.StartIssueWith(7, "Plan")
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "/plan typed", func() bool { return strings.Contains(h.Pane(t, plan.ID), "/plan 7") })
	if _, err := h.StartIssueWith(8, "Plan"); err == nil || !strings.Contains(err.Error(), `no action "Plan"`) {
		t.Errorf("an action of another section: %v", err)
	}
}

func TestIssueOfNoSectionTypesTheStartLine(t *testing.T) {
	h := newHarness(t)
	s, err := h.StartIssue(10)
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "start line typed", func() bool { return strings.Contains(h.Pane(t, s.ID), "Work on issue #10: Maybe") })
}

func TestManualPromptSendTypesWithoutSending(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    session_prompt_send: manual\n"})
	s, err := h.StartIssue(7)
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "/ship typed", func() bool { return strings.Contains(h.Pane(t, s.ID), "/ship 7") })
	time.Sleep(600 * time.Millisecond)
	if pane := h.Pane(t, s.ID); strings.Contains(pane, "No such file or directory") {
		t.Errorf("the command was sent:\n%s", pane)
	}
}

func TestRepoFailureIsWarnedAboutAndNotRemembered(t *testing.T) {
	gh := &apptest.FakeGH{Repo: errors.New("gh repo view: exit status 1: error connecting to api.github.com\nsecond line")}
	var warnings []warn.Warning
	is := issues.New(gh.Run, nil, func(event string, payload any) {
		if w, ok := payload.(warn.Warning); ok && event == "warnings" {
			warnings = append(warnings, w)
		}
	})
	ctx := context.Background()
	if repo := is.Repo(ctx, "/x"); repo != "" {
		t.Fatalf("repo = %q", repo)
	}
	if len(warnings) != 1 || warnings[0].Source != "github" || !strings.Contains(warnings[0].Message, "error connecting") || strings.Contains(warnings[0].Message, "second line") {
		t.Fatalf("warnings = %+v", warnings)
	}
	if is.Repo(ctx, "/x") != "" || len(warnings) != 1 {
		t.Errorf("an immediate retry ran gh again: warnings %+v", warnings)
	}

	gh.Repo = nil
	if is.CachedRepo("/x") != "" {
		t.Error("the failed lookup was cached as no repo")
	}
	if _, err := is.List(ctx, project.Project{Dir: "/x"}, true); err != nil {
		t.Fatal(err)
	}
	if repo := is.Repo(ctx, "/x"); repo != "acme/widgets" {
		t.Errorf("after gh worked again, repo = %q", repo)
	}
	if len(warnings) != 2 || warnings[1] != (warn.Warning{Source: "github"}) {
		t.Errorf("recovery was not announced: %+v", warnings)
	}
}

// A refresh asks gh for the repo again; the one it knows must not vanish from the project list meanwhile.
func TestRefreshKeepsTheKnownRepo(t *testing.T) {
	gh := &apptest.FakeGH{}
	var repoCalls atomic.Int32
	asked, answer := make(chan struct{}), make(chan struct{})
	runner := func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		if name == "gh" && args[0] == "repo" && repoCalls.Add(1) == 2 {
			close(asked)
			<-answer
		}
		return gh.Run(ctx, dir, name, args...)
	}
	is := issues.New(runner, nil, func(string, any) {})
	ctx, proj := context.Background(), project.Project{Dir: "/x"}
	if _, err := is.List(ctx, proj, false); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := is.List(ctx, proj, true)
		done <- err
	}()
	<-asked
	if repo := is.CachedRepo("/x"); repo != "acme/widgets" {
		t.Errorf("while gh was asked again, cached repo = %q", repo)
	}
	close(answer)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if repoCalls.Load() != 2 || gh.IssueCalls() != 2 {
		t.Errorf("gh repo view ran %d times, gh issue list %d times; want 2 and 2", repoCalls.Load(), gh.IssueCalls())
	}

	gh.Repo = errors.New("gh repo view: exit status 1: error connecting to api.github.com")
	if _, err := is.List(ctx, proj, true); err != nil {
		t.Fatal(err)
	}
	if repo := is.CachedRepo("/x"); repo != "acme/widgets" {
		t.Errorf("after a failed lookup, cached repo = %q", repo)
	}
	if repo := is.Repo(ctx, "/x"); repo != "acme/widgets" {
		t.Errorf("after a failed lookup, repo = %q", repo)
	}
}

func TestIssueDetail(t *testing.T) {
	h := newHarness(t)

	got, err := h.IssueDetail(7)
	if err != nil {
		t.Fatal(err)
	}
	if got.Number != 7 || got.BodyHTML != "<h2>Description</h2>\n<p>Fix it.</p>" || len(got.Comments) != 1 {
		t.Fatalf("detail = %+v", got)
	}
	if c := got.Comments[0]; c.Author != "amy" || c.BodyHTML != "<p>Agreed.</p>" || c.CreatedAt != time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC).UnixMilli() {
		t.Errorf("comment = %+v", c)
	}
	calls := h.GH.CallLog()
	if len(calls) < 2 || !strings.HasSuffix(calls[len(calls)-2], "issues/7 -H Accept: application/vnd.github.html+json") || !strings.Contains(calls[len(calls)-1], "issues/7/comments") {
		t.Errorf("gh calls = %v", calls)
	}

	empty, err := h.IssueDetail(11)
	if err != nil {
		t.Fatal(err)
	}
	if raw, _ := json.Marshal(empty); string(raw) != `{"number":11,"bodyHTML":"","comments":[]}` {
		t.Errorf("empty issue json = %s", raw)
	}
	if calls := h.GH.CallLog(); strings.Contains(calls[len(calls)-1], "/comments") {
		t.Error("the comments of an issue without any were read")
	}

	if _, err := h.IssueDetail(999); err == nil || !strings.Contains(err.Error(), "#999") {
		t.Errorf("unknown issue: err = %v", err)
	}
}
