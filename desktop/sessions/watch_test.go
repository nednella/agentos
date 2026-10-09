package sessions_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/internal/warn"
)

// pulls is what the repo's pull request list says of one branch.
func pulls(branch, updatedAt string) string {
	return fmt.Sprintf(`[{"number":12,"head":{"ref":%q},"updated_at":%q}]`, branch, updatedAt)
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

func TestWakeWaitsUntilTheSessionIsNotWaiting(t *testing.T) {
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
	eventually(t, "the held prompt once the session works again", func() bool { return strings.Contains(h.Pane(t, s.ID), "/address-review 12") })
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

func TestPollWatcherFetchesOnlyWhatMoved(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_watch_method: poll\n    pr_poll_interval: 1s\n"})
	h.GH.SetPulls(`W/"one"`, pulls("issue-12", "2026-10-01T12:00:00Z"))
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))
	s := issueSession(h, t, 12)
	eventually(t, "the PR found by the poll", func() bool { got, _ := h.Session(s.ID); return got.PR != nil })
	fetched := h.GH.Calls("pr list")
	eventually(t, "a conditional request", func() bool { return h.GH.Calls("api -i") >= 3 })
	if h.GH.Calls("pr list") != fetched {
		t.Errorf("an unchanged list (304) still fetched the PR: %d pr list calls, had %d", h.GH.Calls("pr list"), fetched)
	}

	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.GH.SetPulls(`W/"two"`, pulls("issue-12", "2026-10-02T12:00:00Z"))
	eventually(t, "the moved PR to be fetched", func() bool { got, _ := h.Session(s.ID); return got.PR != nil && got.PR.State == "open" })
	if h.GH.Forwards() != 0 {
		t.Error("pr_watch: poll still tried the webhook")
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
	h := newHarness(t)
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
	if h.GH.Calls("api -i") != 0 {
		t.Error("the webhook watcher polled the list")
	}
}

func TestWebhookAskedForButDownIsWarnedAbout(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    pr_watch_method: webhook\n"})
	issueSession(h, t, 12)
	eventually(t, "a warning that the stream is down", func() bool {
		for _, w := range h.Rec.Warnings() {
			if w.Source == "pull requests" && strings.Contains(w.Message, "gh webhook forward for acme/widgets stopped") {
				return true
			}
		}
		return false
	})
	if h.GH.Calls("api -i") != 0 {
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
