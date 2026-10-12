package sessions_test

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/bus"
)

func interruptLine(at time.Time, text string) string {
	return fmt.Sprintf(`{"type":"user","timestamp":%q,"message":{"role":"user","content":[{"type":"text","text":%q}]}}`+"\n",
		at.UTC().Format(time.RFC3339Nano), text)
}

func TestEscStopsTheTurn(t *testing.T) {
	h := newHarness(t)
	s, err := h.Sessions().Create("#4 work", "", false, 4)
	if err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(t.TempDir(), "conv.jsonl")
	withTranscript := fmt.Sprintf(`{"transcript_path":%q}`, transcript)
	state := func() string { row, _ := h.Session(s.ID); return string(row.State) }

	h.Hook(t, s.ID, "UserPromptSubmit", withTranscript)
	eventually(t, "the session working", func() bool { return state() == "working" })
	appendLines(t, transcript, interruptLine(time.Now(), "[Request interrupted by user]"))
	eventually(t, "the interrupted session idle", func() bool { return state() == "idle" })
	if rec, _ := bus.ReadState(h.State, s.ID); rec.State != "idle" || rec.Event != "Interrupt" {
		t.Errorf("state file = %s after %s, want idle after Interrupt", rec.State, rec.Event)
	}

	h.Hook(t, s.ID, "UserPromptSubmit", withTranscript)
	h.Sessions().Refresh()
	if got := state(); got != "working" {
		t.Errorf("state after a new prompt = %s, want working: the old interrupt is before it", got)
	}

	h.Hook(t, s.ID, "Notification", `{"notification_type":"permission_prompt"}`)
	eventually(t, "the session waiting", func() bool { return state() == "waiting" })
	appendLines(t, transcript, interruptLine(time.Now(), "[Request interrupted by user for tool use]"))
	eventually(t, "the refused prompt idle", func() bool { return state() == "idle" })
}

func TestEscBeforeTheFirstReplyStopsTheTurn(t *testing.T) {
	h := newHarness(t)
	s, err := h.Sessions().Create("#4 work", "", false, 4)
	if err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(t.TempDir(), "conv.jsonl")
	state := func() string { row, _ := h.Session(s.ID); return string(row.State) }
	if err := h.TermOpen(s.ID, 80, 24); err != nil {
		t.Fatal(err)
	}

	h.Hook(t, s.ID, "UserPromptSubmit", fmt.Sprintf(`{"transcript_path":%q}`, transcript))
	appendLines(t, transcript, assistantLine("msg_1", time.Now(), 10, 1))
	if err := h.TermWrite(s.ID, "\x1b"); err != nil {
		t.Fatal(err)
	}
	h.Sessions().Refresh()
	if got := state(); got != "working" {
		t.Errorf("state after an Esc that left a reply going = %s, want working until the transcript says interrupted", got)
	}

	h.Hook(t, s.ID, "UserPromptSubmit", fmt.Sprintf(`{"transcript_path":%q}`, transcript))
	eventually(t, "the new turn working", func() bool { return state() == "working" })
	if err := h.TermWrite(s.ID, "\x1b"); err != nil {
		t.Fatal(err)
	}
	h.TermClose(s.ID)
	eventually(t, "the turn stopped while thinking idle", func() bool { return state() == "idle" })
}
