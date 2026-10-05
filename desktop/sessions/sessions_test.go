package sessions_test

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/sessions"
)

func TestSessionLifecycle(t *testing.T) {
	h := newHarness(t)

	first, err := h.NewSession("first", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.NewSession("second", "")
	if err != nil {
		t.Fatal(err)
	}
	snap := h.Snapshot()
	if len(snap.Sessions) != 2 || snap.Project.Dir != h.Dir || snap.Sessions[0].History == nil {
		t.Fatalf("snapshot = %+v", snap)
	}
	if first.N != 1 || second.N != 2 || first.State != "idle" || first.CreatedAt == 0 {
		t.Errorf("sessions = %+v %+v", first, second)
	}

	t.Run("hooks", func(t *testing.T) {
		before := h.Rec.Count("attention")
		h.Hook(t, second.ID, "Notification", `{"message":"Claude needs your permission"}`)
		eventually(t, "waiting", func() bool {
			s := h.Rec.LastSessions()
			return len(s) == 2 && s[0].ID == second.ID && s[0].State == "waiting"
		})
		s := h.Rec.LastSessions()
		if s[0].Detail != "Claude needs your permission" || len(s[0].History) != 2 || s[0].LastEventAt == 0 {
			t.Errorf("waiting session = %+v", s[0])
		}
		if h.Rec.Count("attention") != before+1 {
			t.Errorf("attention events = %d, want %d", h.Rec.Count("attention"), before+1)
		}
		name, payload := h.Rec.LastEvent()
		if got, _ := json.Marshal(payload); name != "attention" || string(got) != fmt.Sprintf(`{"id":%q,"state":"waiting"}`, second.ID) {
			t.Errorf("last event = %s %s", name, got)
		}

		h.Hook(t, first.ID, "UserPromptSubmit", `{"prompt":"go"}`)
		eventually(t, "working", func() bool {
			s := h.Rec.LastSessions()
			return len(s) == 2 && s[1].ID == first.ID && s[1].State == "working"
		})
		repliedBefore := h.Rec.CountAttention("replied")
		h.Hook(t, first.ID, "Stop", `{}`)
		eventually(t, "idle after the reply", func() bool {
			s := h.Rec.LastSessions()
			return len(s) == 2 && s[1].ID == first.ID && s[1].State == "idle"
		})
		if h.Rec.CountAttention("replied") != repliedBefore+1 {
			t.Error("no replied attention for a turn that ended")
		}
		h.Hook(t, second.ID, "PreToolUse", `{"tool_name":"Edit","tool_input":{"file_path":"a.tsx"}}`)
		eventually(t, "idle then working order", func() bool {
			s := h.Rec.LastSessions()
			return len(s) == 2 && s[0].ID == first.ID && s[1].State == "working" && s[1].Detail == "Edit a.tsx"
		})
	})

	t.Run("rename and kill", func(t *testing.T) {
		if err := h.RenameSession(second.ID, "renamed"); err != nil {
			t.Fatal(err)
		}
		eventually(t, "rename", func() bool {
			s := h.Rec.LastSessions()
			return len(s) == 2 && s[1].Title == "renamed"
		})
		if err := h.KillSession(first.ID); err != nil {
			t.Fatal(err)
		}
		eventually(t, "the killed session to show as ended", func() bool {
			s := h.Rec.LastSessions()
			return len(s) == 2 && s[1].ID == first.ID && s[1].State == "ended" && s[1].EndedAt > 0
		})
		if got := h.Snapshot().Sessions; len(got) != 2 || got[0].ID != second.ID || got[1].State != "ended" || got[1].Title != "first" {
			t.Errorf("sessions after kill = %+v", got)
		}
		if out, err := exec.Command("tmux", "-L", h.Socket, "has-session", "-t", first.ID).CombinedOutput(); err == nil {
			t.Errorf("the killed tmux session is still there: %s", out)
		}
		if err := h.KillSession(first.ID); err == nil {
			t.Error("killing an ended session succeeded")
		}
		if err := h.DismissSession(second.ID); err == nil {
			t.Error("dismissed a session that is running")
		}
		if err := h.DismissSession(first.ID); err != nil {
			t.Fatal(err)
		}
		eventually(t, "the dismissed row to go", func() bool { return len(h.Rec.LastSessions()) == 1 })
		time.Sleep(2500 * time.Millisecond) // a poll must not bring it back
		if got := h.Snapshot().Sessions; len(got) != 1 || got[0].ID != second.ID {
			t.Errorf("sessions after dismiss and a poll = %+v", got)
		}
	})

	t.Run("session ends on its own", func(t *testing.T) {
		if err := exec.Command("tmux", "-L", h.Socket, "kill-session", "-t", second.ID).Run(); err != nil {
			t.Fatal(err)
		}
		eventually(t, "the row to turn ended", func() bool {
			s := h.Rec.LastSessions()
			return len(s) == 1 && s[0].State == "ended" && s[0].EndedAt > 0 && s[0].Title == "renamed"
		})
		if h.Snapshot().Projects[0].Sessions != 0 {
			t.Error("an ended session counts as a running one")
		}

		// A new session does not take the ended one's number.
		third, err := h.NewSession("third", "")
		if err != nil || third.ID == second.ID {
			t.Errorf("new session = %+v, %v", third, err)
		}
		if err := h.DismissSession(second.ID); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPrefillIsTypedNotSubmitted(t *testing.T) {
	h := newHarness(t)
	s, err := h.NewSession("p", "typed prefill text")
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "prefill on the prompt", func() bool { return strings.Contains(h.Pane(t, s.ID), "typed prefill text") })
	time.Sleep(300 * time.Millisecond)
	if pane := h.Pane(t, s.ID); strings.Contains(pane, "not found") {
		t.Errorf("prefill was submitted:\n%s", pane)
	}
}

func TestSessionsSurviveWithoutALocale(t *testing.T) {
	for _, name := range []string{"LANG", "LC_ALL", "LC_CTYPE"} {
		t.Setenv(name, "")
	}
	h := newHarness(t)
	s, err := h.NewSession("naïve", "")
	if err != nil {
		t.Fatal(err)
	}
	h.Sessions().Refresh()
	list := h.Sessions().List()
	if len(list) != 1 || list[0].ID != s.ID || list[0].Title != "naïve" {
		t.Errorf("sessions after a refresh = %+v", list)
	}
}

func TestIdleReminders(t *testing.T) {
	h := newHarness(t)
	s, err := h.NewSession("r", "")
	if err != nil {
		t.Fatal(err)
	}
	state := func() string { got, _ := h.Session(s.ID); return string(got.State) }
	waits := func() int { st, _ := h.Stats(7); return st.Total }

	h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"go"}`)
	eventually(t, "working", func() bool { return state() == "working" })
	replied := h.Rec.CountAttention("replied")
	h.Hook(t, s.ID, "Notification", `{"message":"Claude is waiting for your input","notification_type":"idle_prompt"}`)
	eventually(t, "a turn killed by an error to stop showing as working", func() bool { return state() == "idle" })
	if h.Rec.CountAttention("replied") != replied || waits() != 0 {
		t.Errorf("an idle reminder raised attention (%d) or opened a wait (%d)", h.Rec.CountAttention("replied")-replied, waits())
	}

	h.Hook(t, s.ID, "Notification", `{"message":"needs permission","notification_type":"permission_prompt"}`)
	eventually(t, "waiting", func() bool { return state() == "waiting" })
	h.Hook(t, s.ID, "Notification", `{"message":"Claude is waiting for your input","notification_type":"idle_prompt"}`)
	time.Sleep(300 * time.Millisecond)
	if state() != "waiting" {
		t.Errorf("an idle reminder changed a waiting session to %s", state())
	}

	h.Hook(t, s.ID, "PostToolUse", `{"tool_name":"Edit"}`)
	eventually(t, "working again", func() bool { return state() == "working" })
	h.Hook(t, s.ID, "Stop", `{}`)
	eventually(t, "a wait for the reply", func() bool { return waits() == 2 })
	st, _ := h.Stats(7)
	var kinds []string
	for _, c := range st.ByCause {
		kinds = append(kinds, c.Kind+"/"+c.Label)
	}
	slices.Sort(kinds)
	if !slices.Equal(kinds, []string{"idle/Reply landed", "permission/Permission"}) {
		t.Errorf("causes = %q", kinds)
	}
}

func TestEndedSessionsSurviveARestart(t *testing.T) {
	h := newHarness(t)
	s, err := h.NewSession("keep me", "")
	if err != nil {
		t.Fatal(err)
	}
	h.Hook(t, s.ID, "SessionEnd", `{}`)
	eventually(t, "ended by its hook", func() bool { got, _ := h.Session(s.ID); return got.State == "ended" })
	if err := h.KillSession(s.ID); err != nil && !strings.Contains(err.Error(), "ended") {
		t.Fatal(err)
	}
	eventually(t, "ended after the process is gone", func() bool { got, ok := h.Session(s.ID); return ok && got.State == "ended" && got.EndedAt > 0 })

	fresh := h.Restart(t)
	var got sessions.Session
	for _, v := range fresh.Sessions().List() {
		if v.ID == s.ID {
			got = v
		}
	}
	if got.State != "ended" || got.Title != "keep me" || got.EndedAt == 0 {
		t.Errorf("after a restart = %+v", got)
	}
	if err := fresh.DismissSession(s.ID); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(h.State, "sessions", s.ID+".json")) {
		t.Error("the state file of a dismissed session is still there")
	}
}

func TestProjectCountsFollowTheSessions(t *testing.T) {
	h := newHarness(t)
	counts := func() sessions.Project { return h.Snapshot().Project }
	a, err := h.NewSession("a", "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := h.NewSession("b", "")
	if err != nil {
		t.Fatal(err)
	}
	if got := counts(); got.Sessions != 2 || got.NeedsYou != 0 || got.Working != 0 {
		t.Errorf("fresh counts = %+v", got)
	}
	h.Hook(t, a.ID, "Notification", `{"message":"needs you","notification_type":"permission_prompt"}`)
	h.Hook(t, b.ID, "PreToolUse", `{"tool_name":"Edit"}`)
	eventually(t, "counts to follow the hooks", func() bool {
		got := counts()
		return got.Sessions == 2 && got.NeedsYou == 1 && got.Working == 1
	})
	if h.Rec.Count("projects") == 0 {
		t.Error("no projects event")
	}
	h.Hook(t, b.ID, "Stop", `{}`)
	eventually(t, "a replied session to count as idle, not needing you", func() bool {
		got := counts()
		return got.NeedsYou == 1 && got.Working == 0
	})
	h.Hook(t, a.ID, "SessionEnd", `{}`)
	eventually(t, "an ended session to leave the counts", func() bool { return counts().Sessions == 1 })
}
