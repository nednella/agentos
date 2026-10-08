package sessions_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/internal/warn"
	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/session"
)

// fakeTmux puts a tmux on PATH that fails the calls a test asks it to and passes the rest to the real one,
// on whatever private socket the call names.
type fakeTmux struct{ dir string }

func newFakeTmux(t *testing.T) fakeTmux {
	t.Helper()
	real, err := exec.LookPath("tmux")
	if err != nil {
		t.Skip("tmux is not installed")
	}
	dir := t.TempDir()
	script := fmt.Sprintf(`#!/bin/sh
case " $* " in
  *" list-sessions "*)
    if [ -e %[1]s/fail-list ]; then echo "error connecting to /tmp/x (Permission denied)" >&2; exit 1; fi
    if [ -e %[1]s/fail-slow-list ]; then out=$(%[2]s "$@"); rc=$?; sleep 0.4; printf '%%s\n' "$out"; exit $rc; fi ;;
  *"@agentos-issue"*) if [ -e %[1]s/fail-issue ]; then echo "no space for the option" >&2; exit 1; fi ;;
esac
exec %[2]s "$@"
`, dir, real)
	if err := os.WriteFile(filepath.Join(dir, "tmux"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return fakeTmux{dir}
}

// fail makes the named call ("list" or "issue") fail, or work again. "slow-list" makes list-sessions
// answer late with what tmux held when it was asked.
func (f fakeTmux) fail(t *testing.T, call string, on bool) {
	t.Helper()
	path := filepath.Join(f.dir, "fail-"+call)
	if on {
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
}

func stateOf(h *apptest.Harness, id string) string {
	got, ok := h.Session(id)
	if !ok {
		return "missing"
	}
	return string(got.State)
}

func tmuxRun(t *testing.T, h *apptest.Harness, args ...string) {
	t.Helper()
	if out, err := exec.Command("tmux", append([]string{"-L", h.Socket}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("tmux %v: %v: %s", args, err, out)
	}
}

func TestAFailedTmuxListingEndsNothing(t *testing.T) {
	h := newHarness(t)
	fake := newFakeTmux(t)
	s, err := h.NewSession("keep running", "")
	if err != nil {
		t.Fatal(err)
	}
	h.Hook(t, s.ID, "PreToolUse", `{"tool_name":"Edit"}`)
	eventually(t, "working", func() bool { return stateOf(h, s.ID) == "working" })

	fake.fail(t, "list", true)
	h.Sessions().Refresh()
	h.Sessions().Refresh()
	if got := stateOf(h, s.ID); got != "working" {
		t.Errorf("a failed listing changed the session to %s", got)
	}
	warnings := h.Rec.Warnings()
	if len(warnings) != 1 || warnings[0].Source != "tmux" || !strings.Contains(warnings[0].Message, "Permission denied") {
		t.Fatalf("warnings after two failures = %+v", warnings)
	}

	fake.fail(t, "list", false)
	h.Sessions().Refresh()
	if warnings = h.Rec.Warnings(); len(warnings) != 2 || warnings[1] != (warn.Warning{Source: "tmux"}) {
		t.Errorf("recovery was not announced: %+v", warnings)
	}
	if got := stateOf(h, s.ID); got != "working" {
		t.Errorf("after recovery the session is %s", got)
	}
}

func TestRefreshRevivesASessionTmuxStillLists(t *testing.T) {
	h := newHarness(t)
	s, err := h.NewSession("comes back", "")
	if err != nil {
		t.Fatal(err)
	}
	tmuxRun(t, h, "kill-session", "-t", s.ID)
	h.Sessions().Refresh()
	if got := stateOf(h, s.ID); got != "ended" {
		t.Fatalf("a killed session is %s", got)
	}
	tmuxRun(t, h, "new-session", "-d", "-s", s.ID, "sleep", "60")
	h.Sessions().Refresh()
	if got := stateOf(h, s.ID); got != "idle" {
		t.Errorf("a session tmux lists again is %s, want idle", got)
	}
	if exists(filepath.Join(h.State, "sessions", s.ID+".json")) {
		t.Error("the saved ended record outlived the revival")
	}
}

func TestRefreshDoesNotEndASessionBeingCreated(t *testing.T) {
	h := newHarness(t)
	fake := newFakeTmux(t)
	fake.fail(t, "slow-list", true)
	var wg sync.WaitGroup
	wg.Go(h.Sessions().Refresh) // reads the list now, applies it 400 ms later
	time.Sleep(150 * time.Millisecond)
	s, err := h.NewSession("created meanwhile", "")
	if err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	if ended := h.Rec.Ended(); len(ended) != 0 {
		t.Errorf("a list read before the creation ended %v", ended)
	}
	if got := stateOf(h, s.ID); got != "idle" {
		t.Errorf("the new session is %s", got)
	}
}

func TestDismissedSessionStaysGoneAfterALateHook(t *testing.T) {
	h := newHarness(t)
	s, err := h.NewSession("done", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	if err := h.DismissSession(s.ID); err != nil {
		t.Fatal(err)
	}
	h.Hook(t, s.ID, "SessionEnd", `{}`) // Claude's hook, running after the dismissal
	if exists(filepath.Join(h.State, "sessions", s.ID+".json")) {
		t.Error("the late hook wrote the state file of a dismissed session")
	}
	if got := h.Restart(t).Sessions().List(); len(got) != 0 {
		t.Errorf("a dismissed session came back after a restart: %+v", got)
	}
}

func TestRemovingAProjectWithLiveIssueSessionsIsRefused(t *testing.T) {
	h := newHarness(t)
	s := issueSession(h, t, 12)
	if _, err := h.RemoveProject("main"); err == nil || !strings.Contains(err.Error(), "session 1 for #12") {
		t.Fatalf("RemoveProject = %v, want an error naming the session", err)
	}
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.RemoveProject("main"); err != nil {
		t.Errorf("RemoveProject without a live issue session = %v", err)
	}
}

func TestRemovingAProjectEndsItsSessions(t *testing.T) {
	h := newHarness(t)
	s, err := h.Sessions().Create("work", "", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.RemoveProject("main"); err != nil {
		t.Fatal(err)
	}
	for _, v := range h.Sessions().List() {
		if v.ID == s.ID && v.State != session.Ended {
			t.Errorf("session %s survived forgetting its project: %+v", s.ID, v)
		}
	}
}

func TestNewSessionDoesNotInheritAnOlderOnesLiveFlag(t *testing.T) {
	h := newHarness(t)
	long := time.Now().Add(-time.Hour)
	if err := bus.WriteState(h.State, session.Record{Session: "main/1", State: session.Ended, Event: "SessionEnd", At: long, Title: "old", Issue: 7, EndedAt: long.UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	prs := filepath.Join(h.State, "data", "main", "prs.json")
	if err := os.MkdirAll(filepath.Dir(prs), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prs, []byte(`{"acks":{},"live":{"main/1":true}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	h.Sessions().Refresh()
	if err := h.DismissSession("main/1"); err != nil {
		t.Fatal(err)
	}

	s := issueSession(h, t, 7)
	if s.ID == "main/1" {
		t.Fatal("the new session took the id of the old one")
	}
	h.GH.SetPR(prJSON("MERGED", false, "[]", 0, 0))
	h.RefreshPRs()
	h.RefreshPRs()
	if got, _ := h.Session(s.ID); got.Cleanup != "ask" {
		t.Errorf("a PR first seen merged gave cleanup %q: the new session inherited the old one's live flag", got.Cleanup)
	}
}

func TestSessionIsKilledWhenItsIssueCannotBeRecorded(t *testing.T) {
	h := newHarness(t)
	fake := newFakeTmux(t)
	fake.fail(t, "issue", true)
	if _, err := h.Sessions().Create("#7 work", "", false, 7); err == nil {
		t.Fatal("Create succeeded although the issue could not be recorded")
	}
	if got := h.Sessions().List(); len(got) != 0 {
		t.Errorf("the failed session is listed: %+v", got)
	}
	out, _ := exec.Command("tmux", "-L", h.Socket, "list-sessions").CombinedOutput()
	if strings.Contains(string(out), "main/") {
		t.Errorf("the failed session still runs in tmux:\n%s", out)
	}
}

func TestEventsNameTheirProject(t *testing.T) {
	h := newHarness(t)
	if _, err := h.NewSession("x", ""); err != nil {
		t.Fatal(err)
	}
	if got := h.Rec.ProjectOf("sessions"); got != "main" {
		t.Errorf("sessions event project = %q", got)
	}
	if got := h.Snapshot().Project.Key; got != "main" {
		t.Errorf("snapshot project key = %q", got)
	}
}

func TestCleanupStopsWhenTheAppCloses(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{CleanupCommand: "sleep 30; : {worktree} {branch}"})
	repoFixture(t, h)
	issueWorktree(t, h)
	s := issueSession(h, t, 7)
	openThenMerge(h)
	eventually(t, "the clean-up to start", func() bool { got, _ := h.Session(s.ID); return got.Cleanup == "pending" })

	start := time.Now()
	h.App.Stop()
	if took := time.Since(start); took > 8*time.Second {
		t.Errorf("Stop took %v: it waited for the clean-up's command", took)
	}
	log := h.Cleanups()
	if len(log) != 1 || log[0].Status != "blocked" || log[0].Reason != "app closed during clean-up" {
		t.Fatalf("log = %+v", log)
	}
}

func TestPollFailuresAreWarnedAboutOnce(t *testing.T) {
	h := newHarness(t) // its folder is no git repo, so listing the worktrees fails too
	issueSession(h, t, 7)
	h.GH.PRErr = errors.New("gh pr list: exit status 1: HTTP 502 from GitHub\nmore detail")
	h.RefreshPRs()
	h.RefreshPRs()

	bySource := map[string][]string{}
	for _, w := range h.Rec.Warnings() {
		bySource[w.Source] = append(bySource[w.Source], w.Message)
	}
	// The message leaves out the branch: lookups of several branches that fail alike make one warning.
	if got := bySource["pull requests"]; len(got) != 1 || !strings.Contains(got[0], "HTTP 502") || strings.Contains(got[0], "more detail") || strings.Contains(got[0], "issue-7") {
		t.Errorf("pull request warnings = %q", got)
	}
	if got := bySource["worktrees"]; len(got) != 1 || !strings.Contains(got[0], "not a git repository") {
		t.Errorf("worktree warnings = %q", got)
	}

	h.GH.PRErr = nil
	h.RefreshPRs()
	got := bySource["pull requests"]
	for _, w := range h.Rec.Warnings() {
		if w.Source == "pull requests" {
			got = append(got[:0:0], w.Message)
		}
	}
	if len(got) != 1 || got[0] != "" {
		t.Errorf("the last pull request warning = %q, want the recovery", got)
	}
}
