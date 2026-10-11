package sessions_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/internal/activity"
	"github.com/nednella/agentos/internal/prompts"
)

func claudeArgs(t *testing.T, dir, id string) []string {
	t.Helper()
	var args []string
	eventually(t, "claude started", func() bool {
		data, err := os.ReadFile(filepath.Join(dir, strings.ReplaceAll(id, "/", "_")+".args"))
		args = strings.Split(strings.TrimSpace(string(data)), "\n")
		return err == nil && len(data) > 0
	})
	return args
}

func TestNewChatStartsClaudeInPlanMode(t *testing.T) {
	dir := apptest.ClaudeOnPath(t)
	h := apptest.NewWith(t, apptest.Options{Agent: "claude"})

	tests := []struct {
		name, title, wantTitle string
	}{
		{"a title", "what does the queue do", "what does the queue do"},
		{"no title", "  ", "chat"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := h.NewChat(tt.title)
			if err != nil {
				t.Fatal(err)
			}
			if !c.Chat || c.Title != tt.wantTitle || c.Issue != 0 || c.State != "idle" {
				t.Errorf("chat = %+v", c)
			}
			args := claudeArgs(t, dir, c.ID)
			if i := slices.Index(args, "--permission-mode"); i < 0 || args[i+1] != "plan" {
				t.Errorf("args = %q", args)
			}
			if i := slices.Index(args, "--allowedTools"); i < 0 || !strings.Contains(args[i+1], "Bash(agentos new:*)") {
				t.Errorf("args = %q", args)
			}
			if slices.Contains(args, "--mcp-config") {
				t.Errorf("a chat has a browser: %q", args)
			}
			if joined := strings.Join(args, "\n"); !strings.Contains(joined, prompts.Chat()) || strings.Contains(joined, prompts.SessionWork()) {
				t.Errorf("system prompt of the chat = %q", joined)
			}
		})
	}

	work, err := h.NewSession("work", "")
	if err != nil {
		t.Fatal(err)
	}
	if work.Chat {
		t.Error("a session is flagged as a chat")
	}
	args := claudeArgs(t, dir, work.ID)
	if slices.Contains(args, "--permission-mode") || !strings.Contains(strings.Join(args, "\n"), prompts.SessionWork()) {
		t.Errorf("args of a session = %q", args)
	}
}

func TestChatFlagSurvivesARestart(t *testing.T) {
	h := newHarness(t)
	live, err := h.NewChat("live")
	if err != nil {
		t.Fatal(err)
	}
	gone, err := h.NewChat("gone")
	if err != nil {
		t.Fatal(err)
	}
	plain, err := h.NewSession("plain", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.KillSession(gone.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the chat ended", func() bool { got, _ := h.Session(gone.ID); return got.State == "ended" })

	fresh := h.Restart(t)
	for id, want := range map[string]bool{live.ID: true, gone.ID: true, plain.ID: false} {
		eventually(t, "the row after a restart", func() bool {
			got, ok := fresh.Session(id)
			return ok && got.Chat == want
		})
	}
	if got, _ := fresh.Session(gone.ID); got.State != "ended" {
		t.Errorf("ended chat = %+v", got)
	}
}

func TestChatStaysOutOfTheStats(t *testing.T) {
	h := newHarness(t)
	chat, err := h.NewChat("ask")
	if err != nil {
		t.Fatal(err)
	}
	work, err := h.NewSession("work", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{chat.ID, work.ID} {
		h.Hook(t, id, "UserPromptSubmit", `{"prompt":"a"}`)
		h.Hook(t, id, "Notification", `{"message":"Claude needs your permission","notification_type":"permission_prompt"}`)
		h.Hook(t, id, "Stop", `{}`)
	}
	eventually(t, "the session's prompt", func() bool { return len(h.Events(t, activity.Prompt)) == 1 })
	eventually(t, "the session's wait", func() bool { st, _ := h.Stats(7); return st.Total > 0 })

	for _, kind := range []string{activity.Prompt, activity.SessionStart, activity.Worked, activity.Wait} {
		for _, e := range h.Events(t, kind) {
			if e.Session == chat.ID {
				t.Errorf("%s event of the chat: %+v", kind, e)
			}
		}
	}
	st, _ := h.Stats(7)
	for _, w := range st.Recent {
		if w.SessionTitle == chat.Title {
			t.Errorf("wait of the chat: %+v", w)
		}
	}
	led, err := h.Ledger(30)
	if err != nil {
		t.Fatal(err)
	}
	if led.Totals.Prompts != 1 || led.Totals.Sessions != 1 {
		t.Errorf("totals = %+v", led.Totals)
	}

	if err := h.KillSession(chat.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the chat ended", func() bool { got, _ := h.Session(chat.ID); return got.State == "ended" })
	if got := h.Events(t, activity.SessionEnd); len(got) != 0 {
		t.Errorf("session_end events = %+v", got)
	}
}

func TestChatHasNoBranchOrPR(t *testing.T) {
	h := newHarness(t)
	repoFixture(t, h)
	ownBranch(t, h.Dir)
	h.GH.SetPR(prJSON("OPEN", true, "[]", 0, 0))

	chat, err := h.NewChat("ask")
	if err != nil {
		t.Fatal(err)
	}
	work := plainSession(h, t)
	worksIn(h, t, chat, h.Dir)
	worksIn(h, t, work, h.Dir)
	eventually(t, "the session's branch", func() bool { got, _ := h.Session(work.ID); return got.Branch == "my-fix" })
	eventually(t, "the session's PR", hasPR(h, work.ID))

	got, _ := h.Session(chat.ID)
	if got.Branch != "" || got.PR != nil || got.Worktree != "" {
		t.Errorf("chat = %+v", got)
	}
	if err := h.Cleanup(chat.ID, true); err == nil {
		t.Error("cleaning up a chat did not fail")
	}
}
