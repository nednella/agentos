package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/session"
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

func (h *harness) session(id string) (Session, bool) {
	for _, s := range h.app.sessions.List() {
		if s.ID == id {
			return s, true
		}
	}
	return Session{}, false
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
		fresh := newApp(cfg, host{emit: h.rec.emit, clipboard: h.rec.clip, openURL: func(string) {}, pickDir: h.rec.picker}, h.gh.run)
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

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
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
	if _, err := h.app.evidence.AddText(s.ID, "tested by hand", "note", "agent"); err != nil {
		t.Fatal(err)
	}

	h.openThenMerge()
	eventually(t, "the session to be cleaned up", func() bool { _, ok := h.session(s.ID); return !ok })

	if exists(wt) || branches(t, h.dir) != "" {
		t.Errorf("worktree exists: %v, branches: %q", exists(wt), branches(t, h.dir))
	}
	log := h.app.Cleanups()
	if len(log) != 1 || log[0].Status != "done" || log[0].Issue != 7 || log[0].PR != 12 || log[0].SessionTitle != "#7 work" {
		t.Fatalf("log = %+v", log)
	}
	if want := []string{"worktree trees/issue-7", "branch issue-7", "temp files", "evidence", "session"}; !slices.Equal(log[0].Removed, want) {
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

func TestSessionsCarryTheGuard(t *testing.T) {
	h := newHarness(t)
	s, err := h.app.NewSession("g", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.app.TermWrite(s.ID, "x"); err == nil {
		t.Fatal("writing to a terminal that is not open succeeded")
	}
	if err := h.app.TypeInto(s.ID, `echo "guard=$AGENTOS_GUARD"`); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("tmux", "-L", h.socket, "send-keys", "-t", s.ID, "Enter").Run(); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the guard patterns in the session's environment", func() bool {
		return strings.Contains(h.pane(t, s.ID), `guard=["gh\\s+pr\\s+merge"`)
	})
}

func TestWaitsAreRecorded(t *testing.T) {
	h := newHarness(t)
	s := h.issueSession(t, 3)
	h.hook(t, s.ID, "PreToolUse", `{"tool_name":"Bash","tool_input":{"command":"upscope test api foo/bar.test.ts"}}`)
	h.hook(t, s.ID, "Notification", `{"message":"Claude needs your permission to use Bash","notification_type":"permission_prompt"}`)
	eventually(t, "open wait", func() bool {
		st, _ := h.app.Stats(7)
		return st.Total == 1 && st.Recent[0].WaitedMs == 0
	})
	time.Sleep(20 * time.Millisecond)
	h.hook(t, s.ID, "PreToolUse", `{"tool_name":"Edit","tool_input":{"file_path":"a.ts"}}`)
	h.hook(t, s.ID, "Notification", `{"message":"waiting for your input","notification_type":"idle_prompt"}`)
	h.hook(t, s.ID, "Notification", `{"message":"Claude needs your permission to use Edit","notification_type":"permission_prompt"}`)
	h.hook(t, s.ID, "PostToolUse", `{"tool_name":"Edit"}`)
	h.hook(t, s.ID, "Notification", `{"message":"pick one","notification_type":"elicitation_dialog"}`)
	h.hook(t, s.ID, "UserPromptSubmit", `{"prompt":"b"}`)
	h.hook(t, s.ID, "Stop", `{}`)
	eventually(t, "finished wait", func() bool {
		st, _ := h.app.Stats(7)
		return st.Total == 4
	})
	if err := h.app.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	st, err := h.app.Stats(7)
	if err != nil {
		t.Fatal(err)
	}
	var causes []string
	for _, c := range st.ByCause {
		causes = append(causes, fmt.Sprintf("%s/%s/%d", c.Kind, c.Label, c.Count))
	}
	want := []string{"finished/Turn finished/1", "permission/Bash: upscope test api/1", "permission/Edit/1", "question/Question/1"}
	slices.Sort(causes)
	if !slices.Equal(causes, want) {
		t.Errorf("causes = %q, want %q", causes, want)
	}
	if st.Recent[len(st.Recent)-1].Issue != 3 || st.Recent[0].SessionTitle != "#3 work" || st.MedianWaitMs <= 0 || len(st.ByDay) != 7 || st.ByDay[6].Count != 4 {
		t.Errorf("stats = %+v", st)
	}
	for _, w := range st.Recent {
		if w.WaitedMs <= 0 {
			t.Errorf("wait left open after the session ended: %+v", w)
		}
	}
	if h.rec.count("stats") < 4 {
		t.Errorf("stats events = %d", h.rec.count("stats"))
	}
	if _, err := os.Stat(filepath.Join(h.state, "data", "stats", "main.jsonl")); err != nil {
		t.Errorf("stats file: %v", err)
	}
}

func TestStatsSummary(t *testing.T) {
	w := newWaits(t.TempDir())
	now := time.Date(2026, 10, 2, 15, 0, 0, 0, time.Local)
	at := func(daysAgo int) int64 { return now.AddDate(0, 0, -daysAgo).UnixMilli() }
	for i, wait := range []Wait{
		{Kind: "permission", Label: "Bash: yarn test", StartedAt: at(0), WaitedMs: 1000},
		{Kind: "permission", Label: "Bash: yarn test", StartedAt: at(0), WaitedMs: 3000},
		{Kind: "finished", Label: "Turn finished", StartedAt: at(2), WaitedMs: 2000},
		{Kind: "permission", Label: "Edit", StartedAt: at(2) + 1, WaitedMs: 10000},
		{Kind: "question", Label: "Question", StartedAt: at(30), WaitedMs: 500},
	} {
		wait.SessionTitle = fmt.Sprint("s", i)
		if err := w.append("p", wait); err != nil {
			t.Fatal(err)
		}
	}
	st, err := w.Stats("p", 7, now)
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != 4 || st.TotalWaitMs != 16000 || st.MedianWaitMs != 2500 {
		t.Errorf("total %d, wait %d, median %d", st.Total, st.TotalWaitMs, st.MedianWaitMs)
	}
	if st.ByCause[0].Label != "Bash: yarn test" || st.ByCause[0].Count != 2 || st.ByCause[0].TotalWaitMs != 4000 || len(st.ByCause) != 3 {
		t.Errorf("byCause = %+v", st.ByCause)
	}
	var days []string
	for _, d := range st.ByDay {
		days = append(days, fmt.Sprintf("%s=%d", d.Day, d.Count))
	}
	if len(days) != 7 || days[0] != "2026-09-26=0" || days[4] != "2026-09-30=2" || days[6] != "2026-10-02=2" {
		t.Errorf("byDay = %v", days)
	}
	if st.Recent[0].StartedAt < st.Recent[len(st.Recent)-1].StartedAt || len(st.Recent) != 4 {
		t.Errorf("recent order = %+v", st.Recent)
	}
	for i := range 60 {
		_ = w.append("q", Wait{Kind: "finished", Label: "Turn finished", StartedAt: at(0) + int64(i), WaitedMs: 1})
	}
	if st, _ = w.Stats("q", 1, now); st.Total != 60 || len(st.Recent) != 50 || st.Recent[0].StartedAt != at(0)+59 {
		t.Errorf("recent cap: total %d, kept %d", st.Total, len(st.Recent))
	}
	if st, _ = w.Stats("none", 3, now); st.Total != 0 || len(st.ByDay) != 3 || st.ByCause == nil || st.Recent == nil {
		t.Errorf("empty stats = %+v", st)
	}
}

func TestCauseOf(t *testing.T) {
	tests := []struct {
		name        string
		rec         session.Record
		kind, label string
	}{
		{"bash command", session.Record{State: session.Waiting, Notify: "permission_prompt", Tool: "Bash", Command: "yarn test api  --watch"}, "permission", "Bash: yarn test api"},
		{"short bash command", session.Record{State: session.Waiting, Notify: "permission_prompt", Tool: "Bash", Command: "ls"}, "permission", "Bash: ls"},
		{"bash without command", session.Record{State: session.Waiting, Notify: "permission_prompt", Tool: "Bash"}, "permission", "Bash"},
		{"other tool", session.Record{State: session.Waiting, Notify: "permission_prompt", Tool: "Edit", Command: "ignored"}, "permission", "Edit"},
		{"unknown tool", session.Record{State: session.Waiting, Notify: "permission_prompt"}, "permission", "Permission"},
		{"untyped permission", session.Record{State: session.Waiting, Detail: "Claude needs your permission to use Edit", Tool: "Edit"}, "permission", "Edit"},
		{"elicitation", session.Record{State: session.Waiting, Notify: "elicitation_dialog", Tool: "Edit"}, "question", "Question"},
		{"finished", session.Record{State: session.Finished, Tool: "Edit"}, "finished", "Turn finished"},
	}
	for _, tt := range tests {
		if kind, label := causeOf(tt.rec); kind != tt.kind || label != tt.label {
			t.Errorf("%s: got %s/%q, want %s/%q", tt.name, kind, label, tt.kind, tt.label)
		}
	}
}

func TestNotesV4(t *testing.T) {
	h := newHarness(t)
	notesDir := filepath.Join(h.state, "data", "notes")
	if err := os.MkdirAll(notesDir, 0o700); err != nil {
		t.Fatal(err)
	}
	old := `[{"id":"a","text":"old done","createdAt":1,"updatedAt":1,"done":true,"issue":0,"issueUrl":""},{"id":"b","text":"old open","createdAt":2,"updatedAt":2,"done":false,"issue":0,"issueUrl":""}]`
	if err := os.WriteFile(filepath.Join(notesDir, "main.json"), []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	notes := h.app.Snapshot().Notes
	if len(notes) != 2 || notes[0].ID != "b" || notes[1].ID != "a" || !notes[1].Archived || notes[0].Archived || notes[0].Images == nil {
		t.Fatalf("migrated notes = %+v", notes)
	}

	fresh, _ := h.app.AddNote("fresh")
	if n, err := h.app.SetNotePinned("a", true); err != nil || !n.Pinned {
		t.Fatalf("SetNotePinned = %+v, %v", n, err)
	}
	if _, err := h.app.SetNoteArchived("a", false); err != nil {
		t.Fatal(err)
	}
	order := func() string {
		var ids []string
		for _, n := range h.app.Snapshot().Notes {
			ids = append(ids, n.ID)
		}
		return strings.Join(ids, ",")
	}
	if got, want := order(), "a,"+fresh.ID+",b"; got != want {
		t.Errorf("pinned first: %s, want %s", got, want)
	}
	if _, err := h.app.SetNoteDone("b", true); err != nil {
		t.Fatal(err)
	}
	if got, want := order(), "a,"+fresh.ID+",b"; got != want {
		t.Errorf("archived last: %s, want %s", got, want)
	}
	if _, err := h.app.SetNoteArchived("a", true); err != nil {
		t.Fatal(err)
	}
	if got := order(); !strings.HasPrefix(got, fresh.ID) {
		t.Errorf("an archived pinned note stays last: %s", got)
	}
}

var onePixelPNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")

func TestNoteImages(t *testing.T) {
	h := newHarness(t)
	n, _ := h.app.AddNote("with picture")
	b64 := base64.StdEncoding.EncodeToString(onePixelPNG)

	n, err := h.app.AddNoteImage(n.ID, b64, "image/png")
	if err != nil || len(n.Images) != 1 || !strings.HasPrefix(n.Images[0], "/media/notes-media/main/") || !strings.HasSuffix(n.Images[0], ".png") {
		t.Fatalf("AddNoteImage = %+v, %v", n, err)
	}
	url := n.Images[0]
	bad := []struct{ name, data, mime string }{
		{"svg", b64, "image/svg+xml"},
		{"html as png", base64.StdEncoding.EncodeToString([]byte("<html></html>")), "image/png"},
		{"not base64", "%%%", "image/png"},
		{"over 10 MB", base64.StdEncoding.EncodeToString(append(slices.Clone(onePixelPNG), make([]byte, maxImageSize)...)), "image/png"},
	}
	for _, tt := range bad {
		if _, err := h.app.AddNoteImage(n.ID, tt.data, tt.mime); err == nil {
			t.Errorf("%s was accepted", tt.name)
		}
	}
	if _, err := h.app.AddNoteImage("nope", b64, "image/png"); err == nil {
		t.Error("an image for an unknown note was accepted")
	}
	if files, _ := os.ReadDir(filepath.Join(h.state, "data", mediaFolder, "main")); len(files) != 1 {
		t.Errorf("%d files stored, want 1", len(files))
	}

	srv := httptest.NewServer(h.app.mediaHandler())
	defer srv.Close()
	get := func(path string) (int, string, []byte) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body := make([]byte, 4096)
		k, _ := resp.Body.Read(body)
		return resp.StatusCode, resp.Header.Get("Content-Type"), body[:k]
	}
	if code, ctype, body := get(url); code != 200 || ctype != "image/png" || string(body) != string(onePixelPNG) {
		t.Errorf("GET %s = %d %s", url, code, ctype)
	}
	for _, p := range []string{
		"/media/notes/main.json",
		"/media/notes-media/../notes/main.json",
		"/media/notes-media/main/../../notes/main.json",
		"/media/notes-media/main/..%2f..%2fnotes%2fmain.json",
		"/media/../../../etc/passwd",
		"/media/notes-media/main/",
		"/media/notes-media/main/missing.png",
		"/other",
	} {
		if code, _, _ := get(p); code != 404 {
			t.Errorf("GET %s = %d, want 404", p, code)
		}
	}

	h.gh.create = "https://github.com/acme/widgets/issues/9\n"
	if _, err := h.app.Issues(false); err != nil {
		t.Fatal(err)
	}
	if _, err := h.app.NoteToIssue(n.ID); err != nil {
		t.Fatal(err)
	}
	var create string
	for _, c := range h.gh.callLog() {
		if strings.HasPrefix(c, "issue create") {
			create = c
		}
	}
	if !strings.Contains(create, "(1 screenshot is attached to the note in agentos.)") {
		t.Errorf("gh call = %q", create)
	}

	if _, err := h.app.RemoveNoteImage(n.ID, "/media/notes-media/main/other.png"); err == nil {
		t.Error("removed a picture the note does not have")
	}
	n, err = h.app.RemoveNoteImage(n.ID, url)
	if err != nil || len(n.Images) != 0 || exists(h.app.notes.mediaPath(url)) {
		t.Errorf("RemoveNoteImage = %+v, %v", n, err)
	}
	n, _ = h.app.AddNoteImage(n.ID, b64, "image/png")
	file := h.app.notes.mediaPath(n.Images[0])
	if err := h.app.DeleteNote(n.ID); err != nil || exists(file) {
		t.Errorf("deleting the note left its picture: %v", err)
	}
}

func TestMediaServedInHTTPMode(t *testing.T) {
	h := newHarness(t)
	n, _ := h.app.AddNote("x")
	n, err := h.app.AddNoteImage(n.ID, base64.StdEncoding.EncodeToString(onePixelPNG), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle(mediaPrefix, h.app.mediaHandler())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", n.Images[0], nil))
	if rec.Code != 200 || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("status %d, headers %v", rec.Code, rec.Header())
	}
}

func (f *fakeGH) setPR(out string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pr = out
}

func (f *fakeGH) callLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

// openThenMerge reports the PR open, then merged: the way a session sees its own PR get merged.
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
	fresh := newApp(cfg, host{emit: h.rec.emit, clipboard: h.rec.clip, openURL: func(string) {}, pickDir: h.rec.picker}, h.gh.run)
	fresh.digestFirst = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := fresh.start(ctx); err != nil {
		t.Fatal(err)
	}
	h.gh.setPR(prJSON("MERGED", false, "[]", 0, 0))
	fresh.RefreshPRs()
	eventually(t, "the restarted app to clean up", func() bool { return !fresh.sessions.Has(s.ID) })
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
