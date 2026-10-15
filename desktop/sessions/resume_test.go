package sessions_test

import (
	"slices"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/sessions"
)

// ended is a session that had the conversation abc and has ended.
func ended(t *testing.T, h *apptest.Harness, s sessions.Session) sessions.Session {
	t.Helper()
	h.Hook(t, s.ID, "Stop", `{"session_id":"abc"}`)
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	var got sessions.Session
	eventually(t, "the session ended", func() bool { got, _ = h.Session(s.ID); return got.State == "ended" })
	return got
}

func TestResumeContinuesAnEndedSession(t *testing.T) {
	dir := apptest.ClaudeOnPath(t)
	h := apptest.NewWith(t, apptest.Options{Agent: "claude"})
	for _, tt := range []struct {
		name  string
		start func() (sessions.Session, error)
	}{
		{"task", func() (sessions.Session, error) { return h.Sessions().Create("#12 work", "", false, 12) }},
		{"chat", func() (sessions.Session, error) { return h.NewChat("ask") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			old, err := tt.start()
			if err != nil {
				t.Fatal(err)
			}
			if gone := ended(t, h, old); !gone.Resumable {
				t.Fatalf("the ended row cannot resume: %+v", gone)
			}
			fresh, err := h.ResumeSession(old.ID)
			if err != nil {
				t.Fatal(err)
			}
			if fresh.ID == old.ID || fresh.State == "ended" || fresh.Title != old.Title || fresh.Issue != old.Issue || fresh.Chat != old.Chat {
				t.Errorf("resumed %+v from %+v", fresh, old)
			}
			args := claudeArgs(t, dir, fresh.ID)
			if i := slices.Index(args, "--resume"); i < 0 || args[i+1] != "abc" {
				t.Errorf("args = %q", args)
			}
			if old.Chat != slices.Contains(args, "--permission-mode") {
				t.Errorf("args of a resumed chat=%v: %q", old.Chat, args)
			}
			if _, ok := h.Session(old.ID); ok {
				t.Error("the ended row stayed")
			}
		})
	}
}

func TestResumeGivesBackTheIssuesLiveSession(t *testing.T) {
	apptest.ClaudeOnPath(t)
	h := apptest.NewWith(t, apptest.Options{Agent: "claude"})
	old, err := h.Sessions().Create("#12 work", "", false, 12)
	if err != nil {
		t.Fatal(err)
	}
	ended(t, h, old)
	live, err := h.Sessions().Create("#12 again", "", false, 12)
	if err != nil {
		t.Fatal(err)
	}
	got, err := h.ResumeSession(old.ID)
	if err != nil || got.ID != live.ID {
		t.Errorf("resume = %+v, %v; want the live %s", got, err, live.ID)
	}
	if _, ok := h.Session(old.ID); !ok {
		t.Error("the ended row went")
	}
}

func TestOnlyAConversationOfClaudeResumes(t *testing.T) {
	t.Run("no conversation", func(t *testing.T) {
		apptest.ClaudeOnPath(t)
		h := apptest.NewWith(t, apptest.Options{Agent: "claude"})
		s, err := h.NewSession("work", "")
		if err != nil {
			t.Fatal(err)
		}
		if err := h.KillSession(s.ID); err != nil {
			t.Fatal(err)
		}
		eventually(t, "the session ended", func() bool { got, _ := h.Session(s.ID); return got.State == "ended" })
		assertNoResume(t, h, s.ID)
	})
	t.Run("plain agent", func(t *testing.T) {
		h := newHarness(t)
		s, err := h.NewSession("work", "")
		if err != nil {
			t.Fatal(err)
		}
		ended(t, h, s)
		assertNoResume(t, h, s.ID)
	})
}

func assertNoResume(t *testing.T, h *apptest.Harness, id string) {
	t.Helper()
	if got, _ := h.Session(id); got.Resumable {
		t.Errorf("row = %+v", got)
	}
	if _, err := h.ResumeSession(id); err == nil {
		t.Error("resume started a session")
	}
	if _, ok := h.Session(id); !ok {
		t.Error("the ended row went")
	}
}
