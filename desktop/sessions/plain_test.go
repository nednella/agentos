package sessions_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/sessions"
	ctl "github.com/nednella/agentos/internal/control"
)

func plainSession(h *apptest.Harness, t *testing.T) sessions.Session {
	t.Helper()
	s, err := h.Sessions().Create("my work", "", false, 0)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// worksIn tells the app, as the agent's hook does, that the session is working in dir.
func worksIn(h *apptest.Harness, t *testing.T, s sessions.Session, dir string) {
	t.Helper()
	cwd, _ := json.Marshal(map[string]string{"cwd": dir})
	h.Hook(t, s.ID, "Stop", string(cwd))
}

// ownBranch checks out my-fix in dir with one pushed commit.
func ownBranch(t *testing.T, dir string) {
	t.Helper()
	gitIn(t, dir, "checkout", "-b", "my-fix")
	if err := os.WriteFile(filepath.Join(dir, "fix.txt"), []byte("fix\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "fix.txt")
	gitIn(t, dir, "commit", "-m", "fix")
	gitIn(t, dir, "push", "-u", "origin", "my-fix")
}

func hasPR(h *apptest.Harness, id string) func() bool {
	return func() bool { got, _ := h.Session(id); return got.PR != nil }
}

func TestPRIsFoundWhenTheTurnEnds(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	ownBranch(t, h.Dir)
	s := plainSession(h, t)
	worksIn(h, t, s, h.Dir)
	eventually(t, "the branch", func() bool { got, _ := h.Session(s.ID); return got.Branch == "my-fix" })

	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))
	worksIn(h, t, s, h.Dir)
	eventually(t, "the PR the agent opened during the turn", hasPR(h, s.ID))
}

func TestPlainSessionFindsItsPR(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	ownBranch(t, h.Dir)
	s := plainSession(h, t)
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))

	worksIn(h, t, s, h.Dir)
	eventually(t, "the PR of the branch the session works on", hasPR(h, s.ID))
	if got, _ := h.Session(s.ID); got.Branch != "my-fix" || got.PR.State != "draft" {
		t.Errorf("session = %+v", got)
	}

	fresh := h.Restart(t)
	fresh.RefreshPRs()
	var again sessions.Session
	for _, v := range fresh.Sessions().List() {
		if v.ID == s.ID {
			again = v
		}
	}
	if again.Branch != "my-fix" || again.PR == nil {
		t.Errorf("after restart = %+v", again)
	}
}

func TestPlainSessionOnMainHasNoPR(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	s := plainSession(h, t)
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))

	worksIn(h, t, s, h.Dir)
	h.RefreshPRs()
	if got, _ := h.Session(s.ID); got.PR != nil || got.Branch != "" {
		t.Errorf("session on main = %+v", got)
	}
	if err := h.Cleanup(s.ID, false); err == nil {
		t.Error("cleaned up a session with no branch")
	}
}

func TestPlainSessionCleanupCommandRunsInTheProjectFolder(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{CleanupCommand: "git -C {dir} switch main && git branch -D {branch}"})
	repoFixture(t, h)
	ownBranch(t, h.Dir)
	s := plainSession(h, t)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	worksIn(h, t, s, h.Dir)
	eventually(t, "the PR", hasPR(h, s.ID))

	h.GH.SetPR(prJSON("MERGED", false, "[]", 0, 0))
	h.RefreshPRs()
	eventually(t, "the session to be cleaned up", func() bool { _, ok := h.Session(s.ID); return !ok })

	if got := gitIn(t, h.Dir, "branch", "--show-current"); got != "main" {
		t.Errorf("project folder is on %q", got)
	}
	if got := gitIn(t, h.Dir, "branch", "--list", "my-fix"); got != "" {
		t.Errorf("branch left: %q", got)
	}
	log := h.Cleanups()
	if len(log) != 1 || log[0].Status != "done" || log[0].Issue != 0 || log[0].PR != 12 {
		t.Fatalf("log = %+v", log)
	}
	if want := []string{"clean-up command for my-fix", "temp files", "session"}; !slices.Equal(log[0].Removed, want) {
		t.Errorf("removed = %q, want %q", log[0].Removed, want)
	}
}

func TestPlainSessionCleanupBlockedByChangesInTheProjectFolder(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	ownBranch(t, h.Dir)
	s := plainSession(h, t)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	worksIn(h, t, s, h.Dir)
	eventually(t, "the PR", hasPR(h, s.ID))
	if err := os.WriteFile(filepath.Join(h.Dir, "fix.txt"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	h.GH.SetPR(prJSON("MERGED", false, "[]", 0, 0))
	h.RefreshPRs()
	eventually(t, "blocked", func() bool { got, _ := h.Session(s.ID); return got.Cleanup == "blocked" })

	if got, _ := h.Session(s.ID); got.CleanupReason != "the worktree has uncommitted changes" {
		t.Errorf("reason = %q", got.CleanupReason)
	}
	if got := gitIn(t, h.Dir, "branch", "--show-current"); got != "my-fix" {
		t.Errorf("project folder is on %q although the clean-up was blocked", got)
	}
}

func TestPlainSessionCleanupRemovesItsWorktree(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	wt := filepath.Join(h.Dir, "trees", "my-fix")
	gitIn(t, h.Dir, "worktree", "add", "-b", "my-fix", wt)
	gitIn(t, wt, "push", "-u", "origin", "my-fix")
	s := plainSession(h, t)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	worksIn(h, t, s, wt)
	eventually(t, "the PR", hasPR(h, s.ID))

	h.GH.SetPR(prJSON("MERGED", false, "[]", 0, 0))
	h.RefreshPRs()
	eventually(t, "the session to be cleaned up", func() bool { _, ok := h.Session(s.ID); return !ok })

	if exists(wt) || gitIn(t, h.Dir, "branch", "--list", "my-fix") != "" {
		t.Errorf("worktree exists: %v, branches: %q", exists(wt), gitIn(t, h.Dir, "branch", "--list", "my-fix"))
	}
	if got := gitIn(t, h.Dir, "branch", "--show-current"); got != "main" {
		t.Errorf("project folder is on %q", got)
	}
}

func TestSessionReportsItsBranch(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	ownBranch(t, h.Dir)
	gitIn(t, h.Dir, "checkout", "main")
	s := plainSession(h, t)
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))

	if resp := h.Ask(t, ctl.Request{Cmd: "track", Session: s.ID}); resp.OK {
		t.Error("track without a branch was accepted")
	}
	if resp := h.Ask(t, ctl.Request{Cmd: "track", Opts: map[string]string{"branch": "my-fix"}}); resp.OK {
		t.Error("track outside a session was accepted")
	}
	resp := h.Ask(t, ctl.Request{Cmd: "track", Session: s.ID, Opts: map[string]string{"branch": "my-fix"}})
	if !resp.OK {
		t.Fatalf("track: %s", resp.Error)
	}
	eventually(t, "the PR of the reported branch", hasPR(h, s.ID))
	if got, _ := h.Session(s.ID); got.Branch != "my-fix" {
		t.Errorf("session = %+v", got)
	}
}

func TestIssueSessionWithoutBranchPatternFollowsItsFolder(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{NoBranch: true})
	repoFixture(t, h)
	ownBranch(t, h.Dir)
	gitIn(t, h.Dir, "checkout", "main")
	s, err := h.Sessions().Create("#5 work", "", false, 5)
	if err != nil {
		t.Fatal(err)
	}
	if s.Branch != "" {
		t.Errorf("fresh session names branch %q", s.Branch)
	}
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))

	gitIn(t, h.Dir, "checkout", "my-fix")
	worksIn(h, t, s, h.Dir)
	eventually(t, "the PR of the branch the session works on", hasPR(h, s.ID))
	if got, _ := h.Session(s.ID); got.Branch != "my-fix" {
		t.Errorf("session = %+v", got)
	}
}

func TestIssueSessionKnowsItsIssue(t *testing.T) {
	h := newHarness(t)
	s := issueSession(h, t, 9)
	if err := h.TypeInto(s.ID, "echo issue=$AGENTOS_ISSUE"); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("tmux", "-L", h.Socket, "send-keys", "-t", s.ID, "Enter").CombinedOutput(); err != nil {
		t.Fatalf("pressing Enter: %v: %s", err, out)
	}
	eventually(t, "the issue number in the session's environment", func() bool {
		return strings.Contains(h.Pane(t, s.ID), "issue=9")
	})
}
