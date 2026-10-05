package notes_test

import (
	"strings"
	"testing"

	ctl "github.com/nednella/agentos/internal/control"
)

func TestNoteCommand(t *testing.T) {
	h := newHarness(t)
	s, err := h.NewSession("asker", "")
	if err != nil {
		t.Fatal(err)
	}
	resp := h.Ask(t, ctl.Request{Cmd: "note", Session: s.ID, Args: []string{"check the", "login page"}})
	if !resp.OK || !strings.Contains(resp.Out, "note saved") {
		t.Fatalf("response = %+v", resp)
	}
	if got := h.Snapshot().Notes; len(got) != 1 || got[0].Text != "check the login page" {
		t.Errorf("notes = %+v", got)
	}
	if h.Rec.Last("notes") == nil {
		t.Error("no notes event")
	}
	if resp := h.Ask(t, ctl.Request{Cmd: "note", Session: s.ID, Args: []string{"  "}}); resp.OK {
		t.Error("an empty note was accepted")
	}
}
