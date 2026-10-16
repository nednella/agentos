package sessions_test

import (
	"os/exec"
	"slices"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/sessions"
	"github.com/nednella/agentos/internal/bus"
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

// endedWhileClosed quits the app, ends the session as a computer restart would, and opens the app again a minute later.
func endedWhileClosed(t *testing.T, h *apptest.Harness, id string) *apptest.Harness {
	t.Helper()
	h.Hook(t, id, "Stop", `{"session_id":"abc"}`)
	h.App.Stop()
	if err := exec.Command("tmux", "-L", h.Socket, "kill-session", "-t", "="+id).Run(); err != nil {
		t.Fatal(err)
	}
	rec, err := bus.ReadState(h.State, id)
	if err != nil {
		t.Fatal(err)
	}
	rec.At = rec.At.Add(-time.Minute)
	if err := bus.WriteState(h.State, rec); err != nil {
		t.Fatal(err)
	}
	return h.Restart(t)
}

func TestSessionThatEndedWhileTheAppWasClosedResumesAsItself(t *testing.T) {
	dir := apptest.ClaudeOnPath(t)
	for _, tt := range []struct {
		name  string
		start func(h *apptest.Harness) (sessions.Session, error)
	}{
		{"chat", func(h *apptest.Harness) (sessions.Session, error) { return h.NewChat("ask") }},
		{"issue task", func(h *apptest.Harness) (sessions.Session, error) {
			return h.Sessions().Create("#12 work", "", false, 12)
		}},
		{"renamed task", func(h *apptest.Harness) (sessions.Session, error) {
			s, err := h.NewSession("work", "")
			if err != nil {
				return s, err
			}
			if err := h.RenameSession(s.ID, "better"); err != nil {
				return s, err
			}
			s, _ = h.Session(s.ID)
			return s, nil
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := apptest.NewWith(t, apptest.Options{Agent: "claude"})
			old, err := tt.start(h)
			if err != nil {
				t.Fatal(err)
			}
			fresh := endedWhileClosed(t, h, old.ID)
			var gone sessions.Session
			eventually(t, "the ended row", func() bool { gone, _ = fresh.Session(old.ID); return gone.State == "ended" })
			if gone.Title != old.Title || gone.Issue != old.Issue || gone.Chat != old.Chat || !gone.Resumable {
				t.Errorf("ended row %+v of %+v", gone, old)
			}
			resumed, err := fresh.ResumeSession(old.ID)
			if err != nil {
				t.Fatal(err)
			}
			if resumed.Title != old.Title || resumed.Issue != old.Issue || resumed.Chat != old.Chat {
				t.Errorf("resumed %+v from %+v", resumed, old)
			}
			if args := claudeArgs(t, dir, resumed.ID); old.Chat != slices.Contains(args, "--permission-mode") {
				t.Errorf("args of a resumed chat=%v: %q", old.Chat, args)
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
