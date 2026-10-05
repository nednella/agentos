package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func prJSON(state string, draft bool, rollup string, comments, reviews int) string {
	repeat := func(n int) string {
		return "[" + strings.TrimSuffix(strings.Repeat(`{"id":"x"},`, n), ",") + "]"
	}
	return fmt.Sprintf(`[{"number":12,"url":"https://github.com/acme/widgets/pull/12","state":%q,"isDraft":%v,"statusCheckRollup":%s,"comments":%s,"reviews":%s,"updatedAt":"2026-10-01T12:00:00Z"}]`,
		state, draft, rollup, repeat(comments), repeat(reviews))
}

func TestParsePR(t *testing.T) {
	tests := []struct {
		name         string
		data         string
		state, check string
		comments     int
	}{
		{"draft", prJSON("OPEN", true, "[]", 0, 0), "draft", "none", 0},
		{"open", prJSON("OPEN", false, "[]", 1, 2), "open", "none", 3},
		{"merged", prJSON("MERGED", false, "[]", 0, 0), "merged", "none", 0},
		{"closed", prJSON("CLOSED", false, "[]", 0, 0), "closed", "none", 0},
		{"passing", prJSON("OPEN", false, `[{"status":"COMPLETED","conclusion":"SUCCESS"},{"status":"COMPLETED","conclusion":"SKIPPED"},{"state":"SUCCESS"}]`, 0, 0), "open", "passing", 0},
		{"pending", prJSON("OPEN", false, `[{"status":"COMPLETED","conclusion":"SUCCESS"},{"status":"IN_PROGRESS","conclusion":""}]`, 0, 0), "open", "pending", 0},
		{"failing wins", prJSON("OPEN", false, `[{"status":"IN_PROGRESS"},{"status":"COMPLETED","conclusion":"FAILURE"},{"status":"COMPLETED","conclusion":"SUCCESS"}]`, 0, 0), "open", "failing", 0},
		{"status context error", prJSON("OPEN", false, `[{"state":"ERROR"}]`, 0, 0), "open", "failing", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr, err := parsePR([]byte(tt.data))
			if err != nil || pr == nil {
				t.Fatalf("parsePR = %v, %v", pr, err)
			}
			if pr.State != tt.state || pr.Checks != tt.check || pr.Comments != tt.comments || pr.Number != 12 || pr.UpdatedAt == 0 {
				t.Errorf("pr = %+v", pr)
			}
		})
	}
	if pr, err := parsePR([]byte("[]")); pr != nil || err != nil {
		t.Errorf("no PR gave %v, %v", pr, err)
	}
	if _, err := parsePR([]byte("nope")); err == nil {
		t.Error("bad json accepted")
	}
}

func TestParseWorktrees(t *testing.T) {
	w := parseWorktrees("worktree /p\nHEAD abc\nbranch refs/heads/main\n\nworktree /p/trees/issue-7\nHEAD def\nbranch refs/heads/issue-7\n\nworktree /p/detached\nHEAD 123\ndetached\n")
	if w.main != "/p" || w.byBranch["issue-7"] != "/p/trees/issue-7" || w.byBranch["main"] != "/p" || len(w.byBranch) != 2 {
		t.Errorf("worktrees = %+v", w)
	}
}

func (h *harness) issueSession(t *testing.T, issue int) Session {
	t.Helper()
	s, err := h.app.sessions.Create(fmt.Sprintf("#%d work", issue), "", issue)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestPRTracking(t *testing.T) {
	h := newHarness(t)
	s := h.issueSession(t, 12)
	if s.Branch != "issue-12" || s.PR != nil || s.PRAttention != "" || s.Cleanup != "" {
		t.Errorf("fresh session = %+v", s)
	}
	prAlerts := func() int {
		h.rec.mu.Lock()
		defer h.rec.mu.Unlock()
		n := 0
		for _, e := range h.rec.events {
			if m, ok := e.payload.(map[string]string); ok && e.name == "attention" && m["state"] == "pr" && m["id"] == s.ID {
				n++
			}
		}
		return n
	}

	h.gh.setPR(prJSON("OPEN", true, `[{"status":"IN_PROGRESS"}]`, 1, 0))
	h.app.RefreshPRs()
	got, _ := h.session(s.ID)
	if got.PR == nil || got.PR.State != "draft" || got.PR.Checks != "pending" || got.PR.Comments != 1 || got.PRAttention != "" {
		t.Fatalf("first sight = %+v", got)
	}

	h.gh.setPR(prJSON("OPEN", false, "[]", 2, 1))
	h.app.RefreshPRs()
	h.app.RefreshPRs()
	if got, _ = h.session(s.ID); got.PRAttention != "comments" || got.PR.State != "open" {
		t.Fatalf("new comments = %+v", got)
	}
	if prAlerts() != 1 {
		t.Errorf("pr attention events = %d, want 1", prAlerts())
	}
	if last := h.rec.lastSessions(); len(last) == 0 || last[0].PRAttention != "comments" {
		t.Error("the sessions event did not carry the attention")
	}

	if err := h.app.AckPR(s.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ = h.session(s.ID); got.PRAttention != "" {
		t.Errorf("after ack = %+v", got)
	}

	h.gh.setPR(prJSON("OPEN", false, `[{"status":"COMPLETED","conclusion":"FAILURE"}]`, 3, 0))
	h.app.RefreshPRs()
	if got, _ = h.session(s.ID); got.PRAttention != "checks" || prAlerts() != 2 {
		t.Errorf("failing checks: attention %q, events %d", got.PRAttention, prAlerts())
	}
	if err := h.app.AckPR(s.ID); err != nil {
		t.Fatal(err)
	}

	t.Run("a restart does not alert again", func(t *testing.T) {
		cfg, err := loadConfig()
		if err != nil {
			t.Fatal(err)
		}
		fresh := newApp(cfg, host{emit: h.rec.emit, clipboard: h.rec.clip, pickDir: h.rec.picker}, h.gh.run)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := fresh.start(ctx); err != nil {
			t.Fatal(err)
		}
		before := prAlerts()
		fresh.RefreshPRs()
		var again Session
		for _, v := range fresh.sessions.List() {
			if v.ID == s.ID {
				again = v
			}
		}
		if again.PR == nil || again.PRAttention != "" || prAlerts() != before {
			t.Errorf("after restart = %+v, events %d -> %d", again, before, prAlerts())
		}
		h.gh.setPR(prJSON("OPEN", false, `[{"status":"COMPLETED","conclusion":"FAILURE"}]`, 4, 0))
		fresh.RefreshPRs()
		for _, v := range fresh.sessions.List() {
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

// repoFixture makes h.dir a git repo with a bare origin, and returns the origin's path.
func repoFixture(t *testing.T, h *harness) string {
	t.Helper()
	origin := filepath.Join(t.TempDir(), "origin.git")
	gitIn(t, filepath.Dir(origin), "init", "--bare", origin)
	gitIn(t, h.dir, "init")
	if err := os.WriteFile(filepath.Join(h.dir, "README"), []byte("hi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	gitIn(t, h.dir, "add", "README")
	gitIn(t, h.dir, "commit", "-m", "first")
	gitIn(t, h.dir, "remote", "add", "origin", origin)
	gitIn(t, h.dir, "push", "-u", "origin", "main")
	return origin
}

// issueWorktree adds a worktree on issue-7 with one pushed commit.
func issueWorktree(t *testing.T, h *harness) string {
	t.Helper()
	wt := filepath.Join(h.dir, "trees", "issue-7")
	gitIn(t, h.dir, "worktree", "add", "-b", "issue-7", wt)
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
	s := h.issueSession(t, 7)
	h.hook(t, s.ID, "UserPromptSubmit", `{"prompt":"go"}`)

	h.openThenMerge()
	eventually(t, "the session to be cleaned up", func() bool { _, ok := h.session(s.ID); return !ok })

	if exists(wt) || branches(t, h.dir) != "" {
		t.Errorf("worktree exists: %v, branches: %q", exists(wt), branches(t, h.dir))
	}
	log := h.app.Cleanups()
	if len(log) != 1 || log[0].Status != "done" || log[0].Issue != 7 || log[0].PR != 12 || log[0].SessionTitle != "#7 work" {
		t.Fatalf("log = %+v", log)
	}
	if want := []string{"worktree trees/issue-7", "branch issue-7", "temp files", "session"}; !slices.Equal(log[0].Removed, want) {
		t.Errorf("removed = %q, want %q", log[0].Removed, want)
	}
	if rec := h.rec.last("cleanups"); rec == nil {
		t.Error("no cleanups event")
	}
	if out, _ := exec.Command("tmux", "-L", h.socket, "list-sessions").CombinedOutput(); strings.Contains(string(out), s.ID) {
		t.Errorf("tmux session still there: %s", out)
	}
	if exists(filepath.Join(h.dir, "README")) == false {
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
			s := h.issueSession(t, 7)

			h.openThenMerge()
			eventually(t, "blocked", func() bool { got, _ := h.session(s.ID); return got.Cleanup == "blocked" })

			got, _ := h.session(s.ID)
			if got.CleanupReason != tt.reason {
				t.Errorf("reason = %q, want %q", got.CleanupReason, tt.reason)
			}
			if !exists(wt) || branches(t, h.dir) == "" {
				t.Error("something was removed although the clean-up was blocked")
			}
			log := h.app.Cleanups()
			if len(log) != 1 || log[0].Status != "blocked" || log[0].Reason != tt.reason || len(log[0].Removed) != 0 {
				t.Errorf("log = %+v", log)
			}
			h.app.RefreshPRs()
			h.app.RefreshPRs()
			if len(h.app.Cleanups()) != 1 {
				t.Error("a blocked clean-up was tried again by itself")
			}

			if err := h.app.Cleanup(s.ID, true); err != nil {
				t.Fatal(err)
			}
			if _, ok := h.session(s.ID); ok || exists(wt) || branches(t, h.dir) != "" {
				t.Errorf("forced clean-up left things: wt %v branches %q", exists(wt), branches(t, h.dir))
			}
			if log := h.app.Cleanups(); len(log) != 2 || log[0].Status != "done" {
				t.Errorf("log after force = %+v", log)
			}
		})
	}
}

func TestCleanupClosedAsksFirst(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := h.issueSession(t, 7)

	h.gh.setPR(prJSON("CLOSED", false, "[]", 0, 0))
	h.app.RefreshPRs()
	h.app.RefreshPRs()
	if got, _ := h.session(s.ID); got.Cleanup != "ask" || !exists(wt) || len(h.app.Cleanups()) != 0 {
		t.Fatalf("closed PR: cleanup %q, worktree %v", got.Cleanup, exists(wt))
	}
	if err := h.app.Cleanup(s.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, ok := h.session(s.ID); ok || exists(wt) || branches(t, h.dir) != "" {
		t.Error("the confirmed clean-up left things behind")
	}
}

func TestCleanupMergedWithRemoteBranchGone(t *testing.T) {
	h := newHarness(t)
	origin := repoFixture(t, h)
	wt := issueWorktree(t, h)
	gitIn(t, origin, "branch", "-D", "issue-7")
	gitIn(t, h.dir, "fetch", "--prune")
	s := h.issueSession(t, 7)

	h.openThenMerge()
	eventually(t, "clean-up of a merged branch GitHub deleted", func() bool { _, ok := h.session(s.ID); return !ok })
	if exists(wt) || branches(t, h.dir) != "" {
		t.Error("left things behind")
	}
}

func TestCleanupCustomCommand(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	h.app.sessions.registry.mu.Lock()
	h.app.sessions.registry.cfg.Projects[0].Cleanup = "git worktree remove --force {worktree} && touch cleaned-{branch}"
	h.app.sessions.registry.mu.Unlock()
	s := h.issueSession(t, 7)

	h.openThenMerge()
	eventually(t, "clean-up", func() bool { _, ok := h.session(s.ID); return !ok })
	if exists(wt) || !exists(filepath.Join(h.dir, "cleaned-issue-7")) {
		t.Errorf("custom command: worktree exists %v, marker exists %v", exists(wt), exists(filepath.Join(h.dir, "cleaned-issue-7")))
	}
	if branches(t, h.dir) != "" {
		t.Error("the branch was kept")
	}
}

func TestCleanupLeavesBranchOfMainTree(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	gitIn(t, h.dir, "checkout", "-b", "issue-7")
	s := h.issueSession(t, 7)

	h.openThenMerge()
	eventually(t, "clean-up", func() bool { _, ok := h.session(s.ID); return !ok })
	if branches(t, h.dir) == "" || !exists(filepath.Join(h.dir, "README")) {
		t.Error("the main working tree's branch was removed")
	}
	log := h.app.Cleanups()
	if len(log) != 1 || log[0].Status != "done" || !strings.Contains(strings.Join(log[0].Removed, "|"), "kept") {
		t.Errorf("log = %+v", log)
	}
}

func TestSessionWithoutIssueIsNeverCleanedUp(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	s, err := h.app.NewSession("plain", "")
	if err != nil {
		t.Fatal(err)
	}
	h.openThenMerge()
	time.Sleep(400 * time.Millisecond)
	if _, ok := h.session(s.ID); !ok {
		t.Error("a session without an issue was cleaned up")
	}
	if err := h.app.Cleanup(s.ID, true); err == nil {
		t.Error("Cleanup accepted a session without an issue")
	}
	if got, _ := h.session(s.ID); got.Branch != "" || got.PR != nil {
		t.Errorf("session = %+v", got)
	}
}

func (f *fakeGH) setPR(out string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pr = out
}
func (h *harness) openThenMerge() {
	h.gh.setPR(prJSON("OPEN", false, "[]", 0, 0))
	h.app.RefreshPRs()
	h.gh.setPR(prJSON("MERGED", false, "[]", 0, 0))
	h.app.RefreshPRs()
}

func TestMergedBeforeTheSessionAsksFirst(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := h.issueSession(t, 7)

	h.gh.setPR(prJSON("MERGED", false, "[]", 0, 0))
	h.app.RefreshPRs()
	h.app.RefreshPRs()
	time.Sleep(300 * time.Millisecond)
	if got, _ := h.session(s.ID); got.Cleanup != "ask" || !exists(wt) || branches(t, h.dir) == "" || len(h.app.Cleanups()) != 0 {
		t.Fatalf("an old merged PR: cleanup %q, worktree %v", got.Cleanup, exists(wt))
	}
	if err := h.app.Cleanup(s.ID, true); err != nil {
		t.Fatal(err)
	}
	if _, ok := h.session(s.ID); ok || exists(wt) || branches(t, h.dir) != "" {
		t.Error("the confirmed clean-up left things behind")
	}
}

func TestCleanupAfterRestartBetweenOpenAndMerged(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := h.issueSession(t, 7)
	h.gh.setPR(prJSON("OPEN", false, "[]", 0, 0))
	h.app.RefreshPRs()

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	fresh := newApp(cfg, host{emit: h.rec.emit, clipboard: h.rec.clip, pickDir: h.rec.picker}, h.gh.run)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := fresh.start(ctx); err != nil {
		t.Fatal(err)
	}
	h.gh.setPR(prJSON("MERGED", false, "[]", 0, 0))
	fresh.RefreshPRs()
	eventually(t, "the restarted app to clean up", func() bool {
		for _, v := range fresh.sessions.List() {
			if v.ID == s.ID {
				return false
			}
		}
		return true
	})
	if exists(wt) || branches(t, h.dir) != "" {
		t.Error("the clean-up left things behind")
	}
}

func TestLiveFlagGoesWithTheSession(t *testing.T) {
	h := newHarness(t)
	s := h.issueSession(t, 7)
	h.gh.setPR(prJSON("OPEN", true, "[]", 0, 0))
	h.app.RefreshPRs()
	live := func() bool {
		h.app.life.mu.Lock()
		defer h.app.life.mu.Unlock()
		return h.app.life.liveFile("main")[s.ID]
	}
	if !live() {
		t.Fatal("a draft PR was not counted as live")
	}
	if err := h.app.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	if live() {
		t.Error("the live flag outlived its session")
	}
}
