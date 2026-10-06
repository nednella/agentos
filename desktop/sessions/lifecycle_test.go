package sessions_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/sessions"
	ctl "github.com/nednella/agentos/internal/control"
)

func issueSession(h *apptest.Harness, t *testing.T, issue int) sessions.Session {
	t.Helper()
	s, err := h.Sessions().Create(fmt.Sprintf("#%d work", issue), "", false, issue)
	if err != nil {
		t.Fatal(err)
	}
	// Text that reaches bash before its prompt shows twice: the terminal echoes it, then readline draws it.
	eventually(t, "the shell's prompt", func() bool { return strings.HasSuffix(strings.TrimSpace(h.Pane(t, s.ID)), "$") })
	return s
}

func TestPRTracking(t *testing.T) {
	h := newHarness(t)
	s := issueSession(h, t, 12)
	if s.Branch != "issue-12" || s.PR != nil || s.PRAttention != "" || s.Cleanup != "" {
		t.Errorf("fresh session = %+v", s)
	}
	prAlerts := func() int { return h.Rec.CountAttentionFor(s.ID, "pr") }

	h.GH.SetPR(prJSON("OPEN", true, `[{"status":"IN_PROGRESS"}]`, 1, 0))
	h.RefreshPRs()
	got, _ := h.Session(s.ID)
	if got.PR == nil || got.PR.State != "draft" || got.PR.Checks != "pending" || got.PR.Comments != 1 || got.PRAttention != "" {
		t.Fatalf("first sight = %+v", got)
	}

	h.GH.SetPR(prJSON("OPEN", false, "[]", 2, 1))
	h.RefreshPRs()
	h.RefreshPRs()
	if got, _ = h.Session(s.ID); got.PRAttention != "comments" || got.PR.State != "open" {
		t.Fatalf("new comments = %+v", got)
	}
	if prAlerts() != 1 {
		t.Errorf("pr attention events = %d, want 1", prAlerts())
	}
	if last := h.Rec.LastSessions(); len(last) == 0 || last[0].PRAttention != "comments" {
		t.Error("the sessions event did not carry the attention")
	}

	if err := h.AckPR(s.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ = h.Session(s.ID); got.PRAttention != "" {
		t.Errorf("after ack = %+v", got)
	}

	h.GH.SetPR(prJSON("OPEN", false, `[{"status":"COMPLETED","conclusion":"FAILURE"}]`, 3, 0))
	h.RefreshPRs()
	if got, _ = h.Session(s.ID); got.PRAttention != "checks" || prAlerts() != 2 {
		t.Errorf("failing checks: attention %q, events %d", got.PRAttention, prAlerts())
	}
	if err := h.AckPR(s.ID); err != nil {
		t.Fatal(err)
	}

	t.Run("a restart does not alert again", func(t *testing.T) {
		fresh := h.Restart(t)
		fresh.RefreshPRs()
		var again sessions.Session
		for _, v := range fresh.Sessions().List() {
			if v.ID == s.ID {
				again = v
			}
		}
		if again.PR == nil || again.PRAttention != "" || fresh.Rec.CountAttentionFor(s.ID, "pr") != 0 {
			t.Errorf("after restart = %+v, events %d", again, fresh.Rec.CountAttentionFor(s.ID, "pr"))
		}
		h.GH.SetPR(prJSON("OPEN", false, `[{"status":"COMPLETED","conclusion":"FAILURE"}]`, 4, 0))
		fresh.RefreshPRs()
		for _, v := range fresh.Sessions().List() {
			if v.ID == s.ID {
				again = v
			}
		}
		if again.PRAttention != "comments" {
			t.Errorf("a new comment after restart: %q", again.PRAttention)
		}
	})
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "-c", "init.defaultBranch=main"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v in %s: %v: %s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

// repoFixture makes h.Dir a git repo with a bare origin, and returns the origin's path.
func repoFixture(t *testing.T, h *apptest.Harness) string {
	t.Helper()
	origin := filepath.Join(t.TempDir(), "origin.git")
	gitIn(t, filepath.Dir(origin), "init", "--bare", origin)
	gitIn(t, h.Dir, "init")
	if err := os.WriteFile(filepath.Join(h.Dir, "README"), []byte("hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, h.Dir, "add", "README")
	gitIn(t, h.Dir, "commit", "-m", "first")
	gitIn(t, h.Dir, "remote", "add", "origin", origin)
	gitIn(t, h.Dir, "push", "-u", "origin", "main")
	return origin
}

// issueWorktree adds a worktree on issue-7 with one pushed commit.
func issueWorktree(t *testing.T, h *apptest.Harness) string {
	t.Helper()
	wt := filepath.Join(h.Dir, "trees", "issue-7")
	gitIn(t, h.Dir, "worktree", "add", "-b", "issue-7", wt)
	if err := os.WriteFile(filepath.Join(wt, "work.txt"), []byte("work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, wt, "add", "work.txt")
	gitIn(t, wt, "commit", "-m", "work")
	gitIn(t, wt, "push", "-u", "origin", "issue-7")
	return wt
}

func branches(t *testing.T, dir string) string {
	return gitIn(t, dir, "branch", "--list", "issue-7")
}

func TestCleanupMerged(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)
	h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"go"}`)
	if resp := h.Ask(t, ctl.Request{Cmd: "show", Session: s.ID, Opts: map[string]string{"text": "tested by hand"}}); !resp.OK {
		t.Fatalf("show = %+v", resp)
	}

	openThenMerge(h)
	eventually(t, "the session to be cleaned up", func() bool { _, ok := h.Session(s.ID); return !ok })

	if exists(wt) || branches(t, h.Dir) != "" {
		t.Errorf("worktree exists: %v, branches: %q", exists(wt), branches(t, h.Dir))
	}
	log := h.Cleanups()
	if len(log) != 1 || log[0].Status != "done" || log[0].Issue != 7 || log[0].PR != 12 || !log[0].Merged || log[0].SessionTitle != "#7 work" {
		t.Fatalf("log = %+v", log)
	}
	if want := []string{"clean-up command for issue-7", "temp files", "evidence", "session"}; !slices.Equal(log[0].Removed, want) {
		t.Errorf("removed = %q, want %q", log[0].Removed, want)
	}
	if got := h.Rec.ProjectOf("cleanups"); got != "main" || len(h.Rec.LastCleanups()) != 1 {
		t.Errorf("cleanups event: project %q, %d entries", got, len(h.Rec.LastCleanups()))
	}
	if out, _ := exec.Command("tmux", "-L", h.Socket, "list-sessions").CombinedOutput(); strings.Contains(string(out), s.ID) {
		t.Errorf("tmux session still there: %s", out)
	}
	if exists(filepath.Join(h.Dir, "README")) == false {
		t.Error("the project folder itself was touched")
	}
}

func TestCleanupBlocked(t *testing.T) {
	tests := []struct {
		name   string
		spoil  func(t *testing.T, wt string)
		reason string
	}{
		{"uncommitted change", func(t *testing.T, wt string) {
			if err := os.WriteFile(filepath.Join(wt, "work.txt"), []byte("changed\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "the worktree has uncommitted changes"},
		{"untracked file", func(t *testing.T, wt string) {
			if err := os.WriteFile(filepath.Join(wt, "new.txt"), []byte("x\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		}, "the worktree has uncommitted changes"},
		{"unpushed commit", func(t *testing.T, wt string) {
			if err := os.WriteFile(filepath.Join(wt, "more.txt"), []byte("x\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			gitIn(t, wt, "add", "more.txt")
			gitIn(t, wt, "commit", "-m", "unpushed")
		}, "the branch has a commit that is not on origin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			repoFixture(t, h)
			wt := issueWorktree(t, h)
			tt.spoil(t, wt)
			s := issueSession(h, t, 7)

			openThenMerge(h)
			eventually(t, "blocked", func() bool { got, _ := h.Session(s.ID); return got.Cleanup == "blocked" })

			got, _ := h.Session(s.ID)
			if got.CleanupReason != tt.reason {
				t.Errorf("reason = %q, want %q", got.CleanupReason, tt.reason)
			}
			if !exists(wt) || branches(t, h.Dir) == "" {
				t.Error("something was removed although the clean-up was blocked")
			}
			log := h.Cleanups()
			if len(log) != 1 || log[0].Status != "blocked" || log[0].Reason != tt.reason || len(log[0].Removed) != 0 {
				t.Errorf("log = %+v", log)
			}
			h.RefreshPRs()
			h.RefreshPRs()
			if len(h.Cleanups()) != 1 {
				t.Error("a blocked clean-up was tried again by itself")
			}

			if err := h.Cleanup(s.ID, true); err != nil {
				t.Fatal(err)
			}
			if _, ok := h.Session(s.ID); ok || exists(wt) || branches(t, h.Dir) != "" {
				t.Errorf("forced clean-up left things: wt %v branches %q", exists(wt), branches(t, h.Dir))
			}
			if log := h.Cleanups(); len(log) != 2 || log[0].Status != "done" {
				t.Errorf("log after force = %+v", log)
			}
		})
	}
}

func TestCleanupClosedAsksFirst(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)

	h.GH.SetPR(prJSON("CLOSED", false, "[]", 0, 0))
	h.RefreshPRs()
	h.RefreshPRs()
	if got, _ := h.Session(s.ID); got.Cleanup != "ask" || !exists(wt) || len(h.Cleanups()) != 0 {
		t.Fatalf("closed PR: cleanup %q, worktree %v", got.Cleanup, exists(wt))
	}
	if err := h.Cleanup(s.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, ok := h.Session(s.ID); ok || exists(wt) || branches(t, h.Dir) != "" {
		t.Error("the confirmed clean-up left things behind")
	}
}

func TestCleanupMergedWithRemoteBranchGone(t *testing.T) {
	h := newHarness(t)
	origin := repoFixture(t, h)
	wt := issueWorktree(t, h)
	gitIn(t, origin, "branch", "-D", "issue-7")
	gitIn(t, h.Dir, "fetch", "--prune")
	s := issueSession(h, t, 7)

	openThenMergeAt(h, gitIn(t, h.Dir, "rev-parse", "issue-7"))
	eventually(t, "clean-up of a merged branch GitHub deleted", func() bool { _, ok := h.Session(s.ID); return !ok })
	if exists(wt) || branches(t, h.Dir) != "" {
		t.Error("left things behind")
	}
}

func TestCleanupMergedBlockedByCommitsAfterTheMergedHead(t *testing.T) {
	tests := []struct {
		name   string
		head   func(t *testing.T, h *apptest.Harness, wt string) string
		reason string
	}{
		{"a commit after the last push", func(t *testing.T, h *apptest.Harness, wt string) string {
			head := gitIn(t, wt, "rev-parse", "HEAD")
			if err := os.WriteFile(filepath.Join(wt, "later.txt"), []byte("x\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			gitIn(t, wt, "add", "later.txt")
			gitIn(t, wt, "commit", "-m", "after the merge")
			return head
		}, "the branch has commits that are not in the merged pull request"},
		{"a head git does not know", func(t *testing.T, h *apptest.Harness, wt string) string {
			return strings.Repeat("0", 40)
		}, "git could not compare the branch with the merged pull request"},
		{"no head from GitHub", func(t *testing.T, h *apptest.Harness, wt string) string { return "" },
			"GitHub did not say which commit the merged pull request ended on"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			origin := repoFixture(t, h)
			wt := issueWorktree(t, h)
			head := tt.head(t, h, wt)
			gitIn(t, origin, "branch", "-D", "issue-7")
			gitIn(t, h.Dir, "fetch", "--prune")
			s := issueSession(h, t, 7)

			openThenMergeAt(h, head)
			eventually(t, "blocked", func() bool { got, _ := h.Session(s.ID); return got.Cleanup == "blocked" })
			got, _ := h.Session(s.ID)
			if !strings.HasPrefix(got.CleanupReason, tt.reason) {
				t.Errorf("reason = %q, want %q", got.CleanupReason, tt.reason)
			}
			if !exists(wt) || branches(t, h.Dir) == "" {
				t.Error("something was removed although the branch holds work the merged PR lacks")
			}
		})
	}
}

func TestCleanupCustomCommand(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{CleanupCommand: "git worktree remove --force {worktree} && git branch -D {branch} && touch {dir}/cleaned-{branch}"})
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)

	openThenMerge(h)
	eventually(t, "clean-up", func() bool { _, ok := h.Session(s.ID); return !ok })
	if exists(wt) || !exists(filepath.Join(h.Dir, "cleaned-issue-7")) {
		t.Errorf("custom command: worktree exists %v, marker exists %v", exists(wt), exists(filepath.Join(h.Dir, "cleaned-issue-7")))
	}
	if branches(t, h.Dir) != "" {
		t.Error("the branch was kept")
	}
}

func TestCleanupCommandMayWorkInTheMainTree(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{CleanupCommand: "git checkout main && git branch -D {branch}"})
	repoFixture(t, h)
	gitIn(t, h.Dir, "checkout", "-b", "issue-7")
	gitIn(t, h.Dir, "push", "-u", "origin", "issue-7")
	s := issueSession(h, t, 7)

	openThenMerge(h)
	eventually(t, "clean-up", func() bool { _, ok := h.Session(s.ID); return !ok })
	if branches(t, h.Dir) != "" || gitIn(t, h.Dir, "branch", "--show-current") != "main" {
		t.Errorf("branches %q, folder on %q", branches(t, h.Dir), gitIn(t, h.Dir, "branch", "--show-current"))
	}
}

func TestCleanupWithoutCommandRemovesOnlyTheSession(t *testing.T) {
	h := apptest.New(t)
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)

	openThenMerge(h)
	eventually(t, "the session to be cleaned up", func() bool { _, ok := h.Session(s.ID); return !ok })
	if !exists(wt) || branches(t, h.Dir) == "" {
		t.Errorf("worktree exists: %v, branches: %q", exists(wt), branches(t, h.Dir))
	}
	log := h.Cleanups()
	if len(log) != 1 || log[0].Status != "done" || !slices.Equal(log[0].Removed, []string{"session"}) {
		t.Errorf("log = %+v", log)
	}
}

func TestCleanupCommandFailureStopsBeforeTheSession(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{CleanupCommand: "false"})
	repoFixture(t, h)
	issueWorktree(t, h)
	s := issueSession(h, t, 7)

	openThenMerge(h)
	eventually(t, "blocked", func() bool { got, _ := h.Session(s.ID); return got.Cleanup == "blocked" })
	if got, _ := h.Session(s.ID); !strings.Contains(got.CleanupReason, "the clean-up command failed for issue-7") {
		t.Errorf("reason = %q", got.CleanupReason)
	}
}

func TestCleanupCommandFailureWithNothingLeftCleansUpTheSession(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{CleanupCommand: "git worktree remove --force {worktree} && git branch -D {branch}; exit 3"})
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)

	openThenMerge(h)
	eventually(t, "clean-up", func() bool { _, ok := h.Session(s.ID); return !ok })
	if exists(wt) || branches(t, h.Dir) != "" {
		t.Errorf("worktree exists %v, branches %q", exists(wt), branches(t, h.Dir))
	}
	log := h.Cleanups()
	if len(log) != 1 || log[0].Status != "done" || !strings.Contains(strings.Join(log[0].Removed, "|"), "failed with nothing left: sh -c: exit status 3") {
		t.Errorf("log = %+v", log)
	}
}

func TestCleanupCommandFailureWithABranchLeftStillBlocks(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{CleanupCommand: "git worktree remove --force {worktree}; exit 3"})
	repoFixture(t, h)
	issueWorktree(t, h)
	s := issueSession(h, t, 7)

	openThenMerge(h)
	eventually(t, "blocked", func() bool { got, _ := h.Session(s.ID); return got.Cleanup == "blocked" })
	if got, _ := h.Session(s.ID); !strings.Contains(got.CleanupReason, "the clean-up command failed for issue-7") {
		t.Errorf("reason = %q", got.CleanupReason)
	}
}

func TestSessionWithoutIssueIsNeverCleanedUp(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	s, err := h.NewSession("plain", "")
	if err != nil {
		t.Fatal(err)
	}
	openThenMerge(h)
	time.Sleep(400 * time.Millisecond)
	if _, ok := h.Session(s.ID); !ok {
		t.Error("a session without an issue was cleaned up")
	}
	if err := h.Cleanup(s.ID, true); err == nil {
		t.Error("Cleanup accepted a session without an issue")
	}
	if got, _ := h.Session(s.ID); got.Branch != "" || got.PR != nil {
		t.Errorf("session = %+v", got)
	}
}

func openThenMerge(h *apptest.Harness) {
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()
	h.GH.SetPR(prJSON("MERGED", false, "[]", 0, 0))
	h.RefreshPRs()
}

// openThenMergeAt is openThenMerge for a merged PR that says which commit its branch ended on.
func openThenMergeAt(h *apptest.Harness, head string) {
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()
	h.GH.SetPR(apptest.WithHead(prJSON("MERGED", false, "[]", 0, 0), head))
	h.RefreshPRs()
}

func TestMergedBeforeTheSessionAsksFirst(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)

	h.GH.SetPR(prJSON("MERGED", false, "[]", 0, 0))
	h.RefreshPRs()
	h.RefreshPRs()
	time.Sleep(300 * time.Millisecond)
	if got, _ := h.Session(s.ID); got.Cleanup != "ask" || !exists(wt) || branches(t, h.Dir) == "" || len(h.Cleanups()) != 0 {
		t.Fatalf("an old merged PR: cleanup %q, worktree %v", got.Cleanup, exists(wt))
	}
	if err := h.Cleanup(s.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, ok := h.Session(s.ID); ok || exists(wt) || branches(t, h.Dir) != "" {
		t.Error("the confirmed clean-up left things behind")
	}
}

func TestCleanupAfterRestartBetweenOpenAndMerged(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()

	fresh := h.Restart(t)
	h.GH.SetPR(prJSON("MERGED", false, "[]", 0, 0))
	fresh.RefreshPRs()
	eventually(t, "the restarted app to clean up", func() bool {
		for _, v := range fresh.Sessions().List() {
			if v.ID == s.ID {
				return false
			}
		}
		return true
	})
	if exists(wt) || branches(t, h.Dir) != "" {
		t.Error("the clean-up left things behind")
	}
}

func TestLiveFlagGoesWithTheSession(t *testing.T) {
	h := newHarness(t)
	s := issueSession(h, t, 7)
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))
	h.RefreshPRs()
	live := func() bool {
		data, _ := os.ReadFile(filepath.Join(h.State, "data", "main", "prs.json"))
		var f struct{ Live map[string]bool }
		_ = json.Unmarshal(data, &f)
		return f.Live[s.ID]
	}
	if !live() {
		t.Fatal("a draft PR was not counted as live")
	}
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	if !live() {
		t.Error("the live flag went with the session's end, although its row still tracks the PR")
	}
	if err := h.DismissSession(s.ID); err != nil {
		t.Fatal(err)
	}
	if live() {
		t.Error("the live flag outlived its row")
	}
}

func TestPROpenedAttention(t *testing.T) {
	t.Run("a PR seen while the session is idle", func(t *testing.T) {
		h := newHarness(t)
		s := issueSession(h, t, 12)
		h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))
		h.RefreshPRs()
		h.RefreshPRs()
		if got := h.Rec.CountAttentionFor(s.ID, "opened"); got != 1 {
			t.Errorf("opened attention events = %d, want 1", got)
		}
		fresh := h.Restart(t)
		fresh.RefreshPRs()
		if got := fresh.Rec.CountAttentionFor(s.ID, "opened"); got != 0 {
			t.Errorf("opened attention events after restart = %d, want 0", got)
		}
	})

	t.Run("a PR seen while the session works waits for its reply", func(t *testing.T) {
		h := newHarness(t)
		s := issueSession(h, t, 12)
		h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"go"}`)
		eventually(t, "working", func() bool {
			got, _ := h.Session(s.ID)
			return got.State == "working"
		})
		h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))
		h.RefreshPRs()
		if got := h.Rec.CountAttentionFor(s.ID, "opened"); got != 0 {
			t.Errorf("opened attention events while working = %d, want 0", got)
		}
		h.Hook(t, s.ID, "Stop", `{}`)
		eventually(t, "idle", func() bool {
			got, _ := h.Session(s.ID)
			return got.State == "idle"
		})
		if opened, replied := h.Rec.CountAttentionFor(s.ID, "opened"), h.Rec.CountAttentionFor(s.ID, "replied"); opened != 1 || replied != 0 {
			t.Errorf("after the reply: opened %d, replied %d; want 1 and 0", opened, replied)
		}
		h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"more"}`)
		h.Hook(t, s.ID, "Stop", `{}`)
		eventually(t, "replied again", func() bool { return h.Rec.CountAttentionFor(s.ID, "replied") == 1 })
		if got := h.Rec.CountAttentionFor(s.ID, "opened"); got != 1 {
			t.Errorf("opened attention events after a second reply = %d, want 1", got)
		}
	})
}

func openThenClose(h *apptest.Harness) {
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()
	h.GH.SetPR(prJSON("CLOSED", false, "[]", 0, 0))
	h.RefreshPRs()
}

func TestCleanupMergeManualAsks(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{CleanupCommand: removeWorktreeAndBranch, ProjectExtra: "    session_cleanup_mode: {merge: manual}\n"})
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)

	openThenMerge(h)
	h.RefreshPRs()
	time.Sleep(300 * time.Millisecond)
	if got, _ := h.Session(s.ID); got.Cleanup != "ask" || !exists(wt) || len(h.Cleanups()) != 0 {
		t.Fatalf("manual merge: cleanup %q, worktree %v", got.Cleanup, exists(wt))
	}
}

func TestCleanupCloseAuto(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{CleanupCommand: removeWorktreeAndBranch, ProjectExtra: "    session_cleanup_mode: {close: auto}\n"})
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)

	openThenClose(h)
	eventually(t, "the session to be cleaned up", func() bool { _, ok := h.Session(s.ID); return !ok })
	if exists(wt) || branches(t, h.Dir) != "" {
		t.Errorf("worktree exists: %v, branches: %q", exists(wt), branches(t, h.Dir))
	}
}

func TestCleanupCloseAutoKeepsSafetyChecks(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{CleanupCommand: removeWorktreeAndBranch, ProjectExtra: "    session_cleanup_mode: {close: auto}\n"})
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)
	if err := os.WriteFile(filepath.Join(wt, "unsaved"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	openThenClose(h)
	eventually(t, "the clean-up to block", func() bool { got, _ := h.Session(s.ID); return got.Cleanup == "blocked" })
	if !exists(wt) {
		t.Error("an automatic clean-up removed a worktree with unsaved changes")
	}
}

func TestCleanupCloseAutoAsksForAPRNeverSeenOpen(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{CleanupCommand: removeWorktreeAndBranch, ProjectExtra: "    session_cleanup_mode: {close: auto}\n"})
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)

	h.GH.SetPR(prJSON("CLOSED", false, "[]", 0, 0))
	h.RefreshPRs()
	h.RefreshPRs()
	time.Sleep(300 * time.Millisecond)
	if got, _ := h.Session(s.ID); got.Cleanup != "ask" || !exists(wt) {
		t.Fatalf("an old closed PR: cleanup %q, worktree %v", got.Cleanup, exists(wt))
	}
}
