package sessions_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/internal/warn"
	"github.com/nednella/agentos/desktop/sessions"
	"github.com/nednella/agentos/internal/project"
)

// pulls is what the repo's pull request list says of one branch.
func pulls(branch, updatedAt string) string {
	return "[" + pull(12, branch, "acme", updatedAt) + "]"
}

// pull is one entry of the repo's pull request list, from a branch in the repo of owner.
func pull(number int, branch, owner, updatedAt string) string {
	return fmt.Sprintf(`{"number":%d,"head":{"ref":%q,"repo":{"owner":{"login":%q}}},"updated_at":%q}`, number, branch, owner, updatedAt)
}

func TestEndedSessionKeepsItsPRAndIsCleanedUpOnMerge(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))
	h.RefreshPRs()
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := h.Session(s.ID); got.State != "ended" || got.PR == nil || got.PR.State != "draft" {
		t.Fatalf("ended row = %+v, want its draft PR", got)
	}

	h.GH.SetPR(prJSON("MERGED", false, "[]", 0, 0))
	h.RefreshPRs()
	eventually(t, "the ended row to be cleaned up", func() bool { _, ok := h.Session(s.ID); return !ok })
	if exists(wt) || branches(t, h.Dir) != "" {
		t.Errorf("worktree exists: %v, branches: %q", exists(wt), branches(t, h.Dir))
	}
	log := h.Cleanups()
	if len(log) != 1 || log[0].Status != "done" || log[0].PR != 12 {
		t.Fatalf("log = %+v", log)
	}
	if removed := strings.Join(log[0].Removed, "|"); !strings.Contains(removed, "session") || !strings.Contains(removed, "clean-up command for issue-7") {
		t.Errorf("removed = %q", log[0].Removed)
	}
}

func TestQueueIsReadAgainWhenAPRMergesOrCloses(t *testing.T) {
	h := newHarness(t)
	if _, err := h.Issues(false); err != nil {
		t.Fatal(err)
	}
	issueSession(h, t, 7)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()
	before := h.GH.IssueCalls()
	h.RefreshPRs()
	if h.GH.IssueCalls() != before {
		t.Fatal("an unchanged PR read the issues again")
	}
	h.GH.SetPR(prJSON("MERGED", false, "[]", 0, 0))
	h.RefreshPRs()
	if h.GH.IssueCalls() != before+1 {
		t.Errorf("issue list calls after the merge = %d, want %d", h.GH.IssueCalls(), before+1)
	}
	if h.Rec.Count("issues") == 0 {
		t.Error("no issues event followed the merge")
	}
	h.RefreshPRs()
	if h.GH.IssueCalls() != before+1 {
		t.Error("a PR that stayed merged read the issues again")
	}
}

func TestReviewWakesTheSession(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"/address-review {n}\"\n    pr_checks_command: \"/fix-checks {n}\"\n"})
	s := issueSession(h, t, 12)
	h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"go"}`)
	h.Hook(t, s.ID, "Stop", `{}`)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 2, 0))
	h.RefreshPRs()
	time.Sleep(500 * time.Millisecond)
	if strings.Contains(h.Pane(t, s.ID), "/address-review") {
		t.Fatal("the comments a PR had when first seen woke the session")
	}

	h.GH.SetPR(prJSON("OPEN", false, "[]", 2, 1))
	h.RefreshPRs()
	eventually(t, "the review prompt in the session", func() bool { return strings.Contains(h.Pane(t, s.ID), "/address-review 12") })
	if strings.Contains(h.Pane(t, s.ID), "APPLICATION_PROMPT") {
		t.Error("a slash command got the app's prefix, so Claude Code would not run it")
	}
	h.RefreshPRs()
	time.Sleep(500 * time.Millisecond)
	if n := strings.Count(h.Pane(t, s.ID), "/address-review 12"); n != 1 {
		t.Errorf("the prompt appears %d times, want once", n)
	}

	t.Run("a restart does not wake again", func(t *testing.T) {
		fresh := h.Restart(t)
		fresh.RefreshPRs()
		time.Sleep(500 * time.Millisecond)
		if n := strings.Count(h.Pane(t, s.ID), "/address-review 12"); n != 1 {
			t.Errorf("the prompt appears %d times after a restart, want once", n)
		}
	})
}

func TestDraftCommentsAreSeenButSendNoReviewPrompt(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"/address-review {n}\"\n    pr_checks_command: \"/fix-checks {n}\"\n"})
	s := issueSession(h, t, 12)
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))
	h.RefreshPRs()
	h.GH.SetPR(prJSON("OPEN", true, `[{"status":"COMPLETED","conclusion":"FAILURE"}]`, 1, 0))
	h.RefreshPRs()
	eventually(t, "the checks prompt on the draft", func() bool { return strings.Contains(h.Pane(t, s.ID), "/fix-checks 12") })
	if strings.Contains(h.Pane(t, s.ID), "/address-review") {
		t.Fatal("a comment on a draft woke the session")
	}

	h.GH.SetPR(prJSON("OPEN", false, "[]", 1, 0))
	h.RefreshPRs()
	time.Sleep(500 * time.Millisecond)
	if strings.Contains(h.Pane(t, s.ID), "/address-review") {
		t.Fatal("a comment made while the PR was a draft woke the session once it was ready")
	}
	h.GH.SetPR(prJSON("OPEN", false, "[]", 1, 1))
	h.RefreshPRs()
	eventually(t, "the review prompt once the PR is ready", func() bool { return strings.Contains(h.Pane(t, s.ID), "/address-review 12") })
}

func TestReviewDraftsWakesTheSessionOnADraft(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"/address-review {n}\"\n    pr_review_drafts: true\n"})
	s := issueSession(h, t, 12)
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))
	h.RefreshPRs()
	h.GH.SetPR(prJSON("OPEN", true, "[]", 1, 0))
	h.RefreshPRs()
	eventually(t, "the review prompt on the draft", func() bool { return strings.Contains(h.Pane(t, s.ID), "/address-review 12") })
}

func TestPlainReviewPromptIsMarkedFromTheApp(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"Read PR {n}\"\n"})
	s := issueSession(h, t, 12)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 2, 0))
	h.RefreshPRs()
	h.GH.SetPR(prJSON("OPEN", false, "[]", 2, 1))
	h.RefreshPRs()
	eventually(t, "the marked review prompt", func() bool { return strings.Contains(h.Pane(t, s.ID), "APPLICATION_PROMPT: Read PR 12") })
}

func TestFailingChecksWakeOncePerCommit(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"/address-review {n}\"\n    pr_checks_command: \"/fix-checks {n}\"\n"})
	s := issueSession(h, t, 12)
	failing := `[{"status":"COMPLETED","conclusion":"FAILURE"}]`
	h.GH.SetPR(apptest.WithHead(prJSON("OPEN", false, failing, 0, 0), "aaa"))
	h.RefreshPRs()
	eventually(t, "the checks prompt", func() bool { return strings.Contains(h.Pane(t, s.ID), "/fix-checks 12") })
	h.RefreshPRs()
	h.RefreshPRs()
	time.Sleep(500 * time.Millisecond)
	if n := strings.Count(h.Pane(t, s.ID), "/fix-checks 12"); n != 1 {
		t.Fatalf("the same failure woke the session %d times, want once", n)
	}
	h.GH.SetPR(apptest.WithHead(prJSON("OPEN", false, failing, 0, 0), "bbb"))
	h.RefreshPRs()
	eventually(t, "a second checks prompt for the new commit", func() bool {
		return strings.Count(h.Pane(t, s.ID), "/fix-checks 12") == 2
	})
}

func TestConflictWakesOncePerCommit(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_conflict_command: \"/rebase {n}\"\n"})
	s := issueSession(h, t, 12)
	h.GH.SetPR(apptest.WithMergeable(apptest.WithHead(prJSON("OPEN", false, "[]", 0, 0), "aaa"), "UNKNOWN"))
	h.RefreshPRs()
	time.Sleep(500 * time.Millisecond)
	if strings.Contains(h.Pane(t, s.ID), "/rebase") {
		t.Fatal("a PR whose mergeability GitHub has not worked out woke the session")
	}

	h.GH.SetPR(apptest.WithMergeable(apptest.WithHead(prJSON("OPEN", false, "[]", 0, 0), "aaa"), "CONFLICTING"))
	h.RefreshPRs()
	eventually(t, "the conflict prompt", func() bool { return strings.Contains(h.Pane(t, s.ID), "/rebase 12") })
	h.RefreshPRs()
	h.RefreshPRs()
	time.Sleep(500 * time.Millisecond)
	if n := strings.Count(h.Pane(t, s.ID), "/rebase 12"); n != 1 {
		t.Fatalf("the same conflict woke the session %d times, want once", n)
	}
	h.GH.SetPR(apptest.WithMergeable(apptest.WithHead(prJSON("OPEN", false, "[]", 0, 0), "bbb"), "CONFLICTING"))
	h.RefreshPRs()
	eventually(t, "a second conflict prompt for the new commit", func() bool {
		return strings.Count(h.Pane(t, s.ID), "/rebase 12") == 2
	})
}

func TestWakeWaitsUntilTheTurnEnds(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"/address-review {n}\"\n    pr_checks_command: \"/fix-checks {n}\"\n"})
	s := issueSession(h, t, 12)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()
	h.Hook(t, s.ID, "Notification", `{"message":"needs permission","notification_type":"permission_prompt"}`)
	eventually(t, "waiting", func() bool { return stateOf(h, s.ID) == "waiting" })

	h.GH.SetPR(prJSON("OPEN", false, "[]", 1, 0))
	h.RefreshPRs()
	time.Sleep(500 * time.Millisecond)
	if strings.Contains(h.Pane(t, s.ID), "/address-review") {
		t.Fatal("the prompt was typed into a session waiting on the user")
	}
	h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"yes"}`)
	eventually(t, "working", func() bool { return stateOf(h, s.ID) == "working" })
	time.Sleep(500 * time.Millisecond)
	if strings.Contains(h.Pane(t, s.ID), "/address-review") {
		t.Fatal("the prompt was typed into a session in the middle of its turn")
	}
	h.Hook(t, s.ID, "Stop", `{}`)
	eventually(t, "the held prompt once the turn ends", func() bool { return strings.Contains(h.Pane(t, s.ID), "/address-review 12") })
}

// withReviews puts the reviews in place of the none PRJSON printed.
func withReviews(prJSON string, reviews ...string) string {
	return strings.Replace(prJSON, `"reviews":{"nodes":[]}`, `"reviews":{"nodes":[`+strings.Join(reviews, ",")+`]}`, 1)
}

func reviewBy(login, state string) string {
	return fmt.Sprintf(`{"author":{"login":%q},"state":%q}`, login, state)
}

func TestOneReviewAndItsRepliesWakeTheSessionOnce(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"/address-review {n}\"\n"})
	s := issueSession(h, t, 12)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()
	h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"go"}`)
	eventually(t, "working", func() bool { return stateOf(h, s.ID) == "working" })

	review := reviewBy("bot", "CHANGES_REQUESTED")
	mine := reviewBy(apptest.GHUser, "COMMENTED")
	answer := reviewBy("bot", "COMMENTED")
	for _, reviews := range [][]string{{review}, {review, mine, mine}, {review, mine, mine, answer}, {review, mine, mine, answer, answer}} {
		h.GH.SetPR(withReviews(prJSON("OPEN", false, "[]", 0, 0), reviews...))
		h.RefreshPRs()
	}
	time.Sleep(500 * time.Millisecond)
	if strings.Contains(h.Pane(t, s.ID), "/address-review") {
		t.Fatal("the prompt was typed into a session in the middle of its turn")
	}
	h.Hook(t, s.ID, "Stop", `{}`)
	eventually(t, "the review prompt once the turn ends", func() bool { return strings.Contains(h.Pane(t, s.ID), "/address-review 12") })
	time.Sleep(500 * time.Millisecond)
	if n := strings.Count(h.Pane(t, s.ID), "/address-review 12"); n != 1 {
		t.Errorf("the prompt appears %d times, want once", n)
	}

	h.GH.SetPR(withReviews(prJSON("OPEN", false, "[]", 0, 0), review, mine, mine, answer, answer, mine))
	h.RefreshPRs()
	time.Sleep(500 * time.Millisecond)
	if n := strings.Count(h.Pane(t, s.ID), "/address-review 12"); n != 1 {
		t.Errorf("a reply by the gh user woke the idle session: the prompt appears %d times, want once", n)
	}
}

// waitingWithHeldReview is a session of issue 12 waiting on the user, with a review prompt held for it.
func waitingWithHeldReview(t *testing.T, h *apptest.Harness) sessions.Session {
	t.Helper()
	s := issueSession(h, t, 12)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()
	h.Hook(t, s.ID, "Notification", `{"message":"needs permission","notification_type":"permission_prompt"}`)
	eventually(t, "waiting", func() bool { return stateOf(h, s.ID) == "waiting" })
	h.GH.SetPR(prJSON("OPEN", false, "[]", 1, 0))
	h.RefreshPRs()
	return s
}

func TestEveryHeldPromptIsSent(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"/address-review {n}\"\n    pr_checks_command: \"/fix-checks {n}\"\n"})
	s := waitingWithHeldReview(t, h)
	h.GH.SetPR(prJSON("OPEN", false, `[{"status":"COMPLETED","conclusion":"FAILURE"}]`, 1, 0))
	h.RefreshPRs()
	time.Sleep(500 * time.Millisecond)
	if pane := h.Pane(t, s.ID); strings.Contains(pane, "/address-review") || strings.Contains(pane, "/fix-checks") {
		t.Fatal("a prompt was typed into a session waiting on the user")
	}
	h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"yes"}`)
	h.Hook(t, s.ID, "Stop", `{}`)
	eventually(t, "both held prompts", func() bool {
		pane := h.Pane(t, s.ID)
		return strings.Contains(pane, "/address-review 12") && strings.Contains(pane, "/fix-checks 12")
	})
	if pane := h.Pane(t, s.ID); strings.Index(pane, "/address-review 12") > strings.Index(pane, "/fix-checks 12") {
		t.Errorf("the held prompts came out of order:\n%s", pane)
	}
}

func TestEndingASessionDropsItsHeldPrompts(t *testing.T) {
	for _, tt := range []struct {
		name string
		end  func(h *apptest.Harness, id string) error
	}{
		{"killed", func(h *apptest.Harness, id string) error { return h.KillSession(id) }},
		{"cleaned up", func(h *apptest.Harness, id string) error { return h.Cleanup(id, true) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"/address-review {n}\"\n"})
			s := waitingWithHeldReview(t, h)
			if err := tt.end(h, s.ID); err != nil {
				t.Fatal(err)
			}
			time.Sleep(500 * time.Millisecond)
			for _, v := range h.Sessions().List() {
				if v.ID != s.ID {
					t.Errorf("a held prompt started session %+v", v)
				}
			}
		})
	}
}

func TestResumeTypesItsPromptWithoutSendingWhenSendIsManual(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"/address-review {n}\"\n    session_prompt_send: manual\n"})
	s := issueSession(h, t, 12)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 1))
	h.RefreshPRs()
	var fresh string
	eventually(t, "a new session for the issue", func() bool {
		for _, v := range h.Sessions().List() {
			if v.Issue == 12 && v.State != "ended" {
				fresh = v.ID
				return true
			}
		}
		return false
	})
	eventually(t, "the prompt typed", func() bool { return strings.Contains(h.Pane(t, fresh), "/address-review 12") })
	time.Sleep(600 * time.Millisecond)
	if pane := h.Pane(t, fresh); strings.Contains(pane, "No such file or directory") {
		t.Errorf("the prompt was sent:\n%s", pane)
	}
}

func TestReviewOnAnEndedSessionStartsANewOne(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"/address-review {n}\"\n"})
	s := issueSession(h, t, 12)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 1))
	h.RefreshPRs()
	var fresh string
	eventually(t, "a new session for the issue", func() bool {
		for _, v := range h.Sessions().List() {
			if v.Issue == 12 && v.State != "ended" {
				fresh = v.ID
				return true
			}
		}
		return false
	})
	eventually(t, "the ended row to go", func() bool { _, ok := h.Session(s.ID); return !ok })
	eventually(t, "the prompt in the new session", func() bool { return strings.Contains(h.Pane(t, fresh), "/address-review 12") })
	if got, _ := h.Session(fresh); got.Title != "#12 work" {
		t.Errorf("the new session is titled %q", got.Title)
	}
}

func TestResumedSessionIsCleanedUpOnMerge(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{CleanupCommand: removeWorktreeAndBranch, ProjectExtra: "    pr_review_command: \"/address-review {n}\"\n"})
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 1))
	h.RefreshPRs()
	var fresh string
	eventually(t, "a new session for the issue", func() bool {
		for _, v := range h.Sessions().List() {
			if v.Issue == 7 && v.State != "ended" {
				fresh = v.ID
				return true
			}
		}
		return false
	})
	eventually(t, "the ended row to go", func() bool { _, ok := h.Session(s.ID); return !ok })

	// The new session knows the PR was open, as the ended one did, so the merge cleans up without asking.
	h.GH.SetPR(prJSON("MERGED", false, "[]", 0, 1))
	h.RefreshPRs()
	eventually(t, "the new session to be cleaned up", func() bool { _, ok := h.Session(fresh); return !ok })
	if exists(wt) {
		t.Error("the worktree is still there")
	}
}

func TestReviewOnAnEndedSessionLeavesTheIssuesLiveSession(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_review_command: \"/address-review {n}\"\n"})
	h.GH.SetPRFor("issue-12", "[]")
	h.GH.SetPRFor("side", prJSON("OPEN", false, "[]", 0, 0))
	s := issueSession(h, t, 12)
	if err := h.Sessions().Track(s.ID, "side"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the PR of side", func() bool { got, _ := h.Session(s.ID); return got.PR != nil })
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	live, _, err := h.Sessions().CreateIssue("#12 work", "", 12, project.Action{}, nil, "", "")
	if err != nil {
		t.Fatal(err)
	}
	h.GH.SetPRFor("side", prJSON("OPEN", false, "[]", 0, 1))
	h.RefreshPRs()
	time.Sleep(300 * time.Millisecond)
	var ids []string
	for _, v := range h.Sessions().List() {
		if v.Issue == 12 {
			ids = append(ids, v.ID)
		}
	}
	if !slices.Equal(ids, []string{live.ID, s.ID}) && !slices.Equal(ids, []string{s.ID, live.ID}) {
		t.Errorf("sessions of #12 = %v, want the live %s and the ended %s", ids, live.ID, s.ID)
	}
}

func TestPollWatcherFetchesOnlyWhatMoved(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_watch_method: poll\n    pr_poll_interval: 1s\n"})
	h.GH.SetPulls(`W/"one"`, pulls("issue-12", "2026-10-01T12:00:00Z"))
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))
	s := issueSession(h, t, 12)
	eventually(t, "the PR found by the poll", func() bool { got, _ := h.Session(s.ID); return got.PR != nil })
	fetched := h.GH.Calls("api graphql")
	eventually(t, "a conditional request", func() bool { return h.GH.Calls("api -i") >= 3 })
	if h.GH.Calls("api graphql") != fetched {
		t.Errorf("an unchanged list (304) still fetched the PR: %d PR lookups, had %d", h.GH.Calls("api graphql"), fetched)
	}

	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.GH.SetPulls(`W/"two"`, pulls("issue-12", "2026-10-02T12:00:00Z"))
	eventually(t, "the moved PR to be fetched", func() bool { got, _ := h.Session(s.ID); return got.PR != nil && got.PR.State == "open" })
	if h.GH.Forwards() != 0 {
		t.Error("pr_watch: poll still tried the webhook")
	}
}

func TestEndedSessionWithoutAnOpenPRIsNotWatchedOrFetched(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_watch_method: poll\n    pr_poll_interval: 1s\n"})
	h.GH.SetPulls(`W/"one"`, pulls("other", "2026-10-01T12:00:00Z"))
	s := issueSession(h, t, 12)
	eventually(t, "the poll to run", func() bool { return h.GH.Calls("api -i") >= 1 })
	h.RefreshPRs()
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	h.RefreshPRs()
	polled, fetched := h.GH.Calls("api -i"), h.GH.Calls("api graphql")
	time.Sleep(2500 * time.Millisecond)
	h.RefreshPRs()
	if got := h.GH.Calls("api -i"); got != polled {
		t.Errorf("the watcher polled %d more times for an ended session", got-polled)
	}
	if got := h.GH.Calls("api graphql"); got != fetched {
		t.Errorf("the one-minute check fetched %d more times for an ended session with no PR", got-fetched)
	}
}

func TestStoppingTheAppStopsItsWatchers(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_watch_method: poll\n    pr_poll_interval: 1s\n"})
	h.GH.SetPulls(`W/"one"`, pulls("issue-12", "2026-10-01T12:00:00Z"))
	issueSession(h, t, 12)
	eventually(t, "the poll to run", func() bool { return h.GH.Calls("api -i") >= 2 })
	h.App.Stop()
	polled := h.GH.Calls("api -i")
	time.Sleep(2500 * time.Millisecond)
	if got := h.GH.Calls("api -i"); got != polled {
		t.Errorf("the poll ran %d more times after the app stopped", got-polled)
	}
}

func TestPollWatcherKeepsLookingWhileChecksRun(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_watch_method: poll\n    pr_poll_interval: 1s\n"})
	h.GH.SetPulls(`W/"one"`, pulls("issue-12", "2026-10-01T12:00:00Z"))
	h.GH.SetPR(prJSON("OPEN", false, `[{"status":"IN_PROGRESS"}]`, 0, 0))
	s := issueSession(h, t, 12)
	eventually(t, "pending checks", func() bool { got, _ := h.Session(s.ID); return got.PR != nil && got.PR.Checks == "pending" })
	h.GH.SetPR(prJSON("OPEN", false, `[{"status":"COMPLETED","conclusion":"SUCCESS"}]`, 0, 0))
	eventually(t, "passing checks without the list moving", func() bool { got, _ := h.Session(s.ID); return got.PR.Checks == "passing" })
}

func TestWebhookWatcherRefreshesThePRAnEventNames(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_poll_interval: 1s\n"})
	h.GH.WebhookWorks()
	s := issueSession(h, t, 12)
	eventually(t, "the forward to start", func() bool { return h.GH.Forwards() == 1 })
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))
	h.GH.Event(`{"action":"opened","pull_request":{"number":12}}`)
	eventually(t, "the PR after its opened event", func() bool { got, _ := h.Session(s.ID); return got.PR != nil })

	h.GH.SetPR(prJSON("OPEN", true, `[{"status":"COMPLETED","conclusion":"FAILURE"}]`, 0, 0))
	h.GH.Event(`{"action":"completed","check_suite":{"pull_requests":[{"number":12}]}}`)
	eventually(t, "failing checks after the check_suite event", func() bool { got, _ := h.Session(s.ID); return got.PR.Checks == "failing" })

	h.GH.SetPR(prJSON("OPEN", true, `[{"status":"COMPLETED","conclusion":"FAILURE"}]`, 1, 0))
	h.GH.Event(`{"action":"created","issue":{"number":12,"pull_request":{"url":"x"}},"comment":{"body":"hi"}}`)
	eventually(t, "the comment after the issue_comment event", func() bool { got, _ := h.Session(s.ID); return got.PR.Comments == 1 })
	h.GH.Event(`{"action":"created","issue":{"number":99},"comment":{"body":"not a PR"}}`)
	listed := h.GH.Calls("api -i")
	time.Sleep(2500 * time.Millisecond)
	if h.GH.Calls("api -i") != listed {
		t.Error("the webhook watcher polled the list")
	}
}

func TestWebhookAskedForButDownIsWarnedAbout(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_watch_method: webhook\n    pr_poll_interval: 1s\n"})
	issueSession(h, t, 12)
	eventually(t, "a warning that the stream is down", func() bool {
		for _, w := range h.Rec.Warnings() {
			if w.Source == "pull requests" && strings.Contains(w.Message, "gh webhook forward for acme/widgets stopped") {
				return true
			}
		}
		return false
	})
	listed := h.GH.Calls("api -i")
	time.Sleep(2500 * time.Millisecond)
	if h.GH.Calls("api -i") != listed {
		t.Error("pr_watch: webhook fell back to polling")
	}
}

func TestWebhookThatDoesNotWorkLeavesThePoll(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_poll_interval: 1s\n"})
	h.GH.SetPulls(`W/"one"`, pulls("issue-12", "2026-10-01T12:00:00Z"))
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))
	s := issueSession(h, t, 12)
	eventually(t, "the PR found by the poll", func() bool { got, _ := h.Session(s.ID); return got.PR != nil })
	if h.GH.Forwards() != 1 {
		t.Errorf("gh webhook forward was tried %d times, want once", h.GH.Forwards())
	}
	for _, w := range h.Rec.Warnings() {
		if w == (warn.Warning{Source: "pull requests", Message: w.Message}) && w.Message != "" {
			t.Errorf("a webhook that was never asked for warned: %q", w.Message)
		}
	}
}

func TestNoWakeWithoutACommand(t *testing.T) {
	h := newHarness(t)
	s := issueSession(h, t, 12)
	failing := `[{"status":"COMPLETED","conclusion":"FAILURE"}]`
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()
	h.GH.SetPR(apptest.WithHead(prJSON("OPEN", false, failing, 1, 1), "aaa"))
	h.RefreshPRs()
	eventually(t, "the row flagged", func() bool { got, _ := h.Session(s.ID); return got.PRAttention != "" })
	time.Sleep(500 * time.Millisecond)
	if pane := h.Pane(t, s.ID); strings.Contains(pane, "PR") || strings.Contains(pane, "12") {
		t.Errorf("a project with no wake settings typed into the session:\n%s", pane)
	}
}

// branchTrees adds a worktree on each of br-1 to br-n.
func branchTrees(t *testing.T, h *apptest.Harness, n int) []string {
	t.Helper()
	repoFixture(t, h)
	var trees []string
	for i := 1; i <= n; i++ {
		trees = append(trees, branchWorktree(t, h, fmt.Sprintf("br-%d", i)))
	}
	return trees
}

func TestBranchesTheListHoldsNoPROfAreNotLookedUp(t *testing.T) {
	h := newHarness(t)
	trees := branchTrees(t, h, 5)
	h.GH.SetPulls(`W/"one"`, "["+pull(21, "br-4", "stranger", "2026-10-01T12:00:00Z")+"]")
	h.GH.SetPRFor("br-4", numberedPR(21, "OPEN"))
	s := issueSession(h, t, 7)
	for range 2 {
		for i, wt := range trees {
			worksIn(h, t, s, wt)
			eventually(t, "the session on its branch", onBranch(h, s.ID, fmt.Sprintf("br-%d", i+1)))
		}
	}
	h.RefreshPRs()
	if n := h.GH.Calls("api graphql"); n != 0 {
		t.Errorf("%d PR lookups for branches with no PR of ours, want none", n)
	}

	h.GH.SetPulls(`W/"two"`, "["+pull(20, "br-5", "acme", "2026-10-02T12:00:00Z")+","+pull(21, "br-4", "stranger", "2026-10-01T12:00:00Z")+"]")
	h.GH.SetPRFor("br-5", numberedPR(20, "OPEN"))
	h.RefreshPRs()
	h.RefreshPRs()
	if got, _ := h.Session(s.ID); got.PR == nil || got.PR.Number != 20 {
		t.Errorf("session = %+v, want the PR of br-5 and not the stranger's", got)
	}
	if n, other := h.GH.Lookups("br-5"), h.GH.Calls("api graphql")-h.GH.Lookups("br-5"); n != 2 || other != 0 {
		t.Errorf("br-5 looked up %d times in two reconciles, want 2; other branches %d times, want none", n, other)
	}
}

func TestBranchMissingFromAFullListIsLookedUpOnce(t *testing.T) {
	h := newHarness(t)
	trees := branchTrees(t, h, 2)
	var page []string
	for i := range 100 {
		page = append(page, pull(100+i, fmt.Sprintf("old-%d", i), "acme", "2026-01-01T12:00:00Z"))
	}
	h.GH.SetPulls(`W/"full"`, "["+strings.Join(page, ",")+"]")
	s := issueSession(h, t, 7)
	for i, wt := range trees {
		worksIn(h, t, s, wt)
		eventually(t, "the session on its branch", onBranch(h, s.ID, fmt.Sprintf("br-%d", i+1)))
	}
	h.RefreshPRs()
	h.RefreshPRs()
	if a, b := h.GH.Lookups("br-1"), h.GH.Lookups("br-2"); a != 1 || b != 1 {
		t.Errorf("br-1 looked up %d times, br-2 %d times, want once each", a, b)
	}
}

func TestBranchWithNoPRIsDroppedOnceItsWorktreeIsGone(t *testing.T) {
	h := newHarness(t)
	trees := branchTrees(t, h, 2)
	s := issueSession(h, t, 7)
	for i, wt := range trees {
		worksIn(h, t, s, wt)
		eventually(t, "the session on its branch", onBranch(h, s.ID, fmt.Sprintf("br-%d", i+1)))
	}
	gitIn(t, h.Dir, "worktree", "remove", trees[0])
	h.RefreshPRs()
	gone, kept := h.GH.Lookups("br-1"), h.GH.Lookups("br-2")
	h.RefreshPRs()
	if h.GH.Lookups("br-1") != gone {
		t.Error("a branch with no PR and no worktree is still looked up")
	}
	if h.GH.Lookups("br-2") == kept {
		t.Error("the branch that still has its worktree is no longer looked up")
	}
}
