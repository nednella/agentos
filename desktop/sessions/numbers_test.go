package sessions_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/evidence"
	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/session"
)

func evidenceDir(h *apptest.Harness, id string) string {
	name, _ := session.ParseName(id)
	return filepath.Join(h.State, "data", name.Project, "evidence", name.Token)
}

func TestNumbersAreNeverReused(t *testing.T) {
	h := newHarness(t)
	first, err := h.NewSession("first", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.NewSession("second", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.KillSession(first.ID); err != nil {
		t.Fatal(err)
	}
	if err := h.DismissSession(first.ID); err != nil {
		t.Fatal(err)
	}
	third, err := h.NewSession("third", "")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID || first.ID == third.ID || second.ID == third.ID {
		t.Errorf("ids are not distinct: %s %s %s", first.ID, second.ID, third.ID)
	}
	if first.N != 1 || second.N != 2 || third.N != 3 {
		t.Errorf("numbers = %d %d %d, want 1 2 3", first.N, second.N, third.N)
	}
}

func TestNumbersAfterARestartFollowCreationOrder(t *testing.T) {
	h := newHarness(t)
	a, err := h.NewSession("a", "")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond) // tmux dates a session to the second
	if _, err := h.NewSession("b", ""); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * time.Hour)
	rec := session.Record{Session: "main/old1", State: session.Ended, Event: "SessionEnd", At: old, Title: "old", Created: old.UnixMilli(), EndedAt: old.Add(time.Hour).UnixMilli()}
	if err := bus.WriteState(h.State, rec); err != nil {
		t.Fatal(err)
	}

	second := h.Restart(t)
	got := map[string]int{}
	for _, s := range second.Sessions().List() {
		got[s.Title] = s.N
	}
	if got["old"] != 1 || got["a"] != 2 || got["b"] != 3 {
		t.Errorf("numbers = %v, want old 1, a 2, b 3 (a is %s)", got, a.ID)
	}
	c, err := second.NewSession("c", "")
	if err != nil || c.N != 4 {
		t.Errorf("the next session = %+v, %v, want number 4", c, err)
	}
}

func TestNewSessionSeesNothingOfADismissedOne(t *testing.T) {
	h := newHarness(t)
	old, err := h.NewSession("old", "")
	if err != nil {
		t.Fatal(err)
	}
	store := evidence.New(filepath.Join(h.State, "data"))
	if _, err := store.AddText(old.ID, "proof", "", "agent"); err != nil {
		t.Fatal(err)
	}
	if err := h.KillSession(old.ID); err != nil {
		t.Fatal(err)
	}
	if err := h.DismissSession(old.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the evidence to go", func() bool { return !exists(evidenceDir(h, old.ID)) })

	fresh, err := h.NewSession("fresh", "")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Evidence != 0 || exists(evidenceDir(h, fresh.ID)) {
		t.Errorf("the new session has evidence: %+v", fresh)
	}
}

func TestExpiredEndedRowGoesWithItsEvidenceAndPRs(t *testing.T) {
	h := newHarness(t)
	const id = "main/old1"
	long := time.Now().Add(-8 * 24 * time.Hour)
	rec := session.Record{Session: id, State: session.Ended, Event: "SessionEnd", At: long, Title: "old", EndedAt: long.UnixMilli()}
	if err := bus.WriteState(h.State, rec); err != nil {
		t.Fatal(err)
	}
	if _, err := evidence.New(filepath.Join(h.State, "data")).AddText(id, "proof", "", "agent"); err != nil {
		t.Fatal(err)
	}
	prs := filepath.Join(h.State, "data", "main", "prs.json")
	if err := os.WriteFile(prs, []byte(`{"acks":{},"live":{"main/old1":true},"branches":{"main/old1":["issue-7"]}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	h.Sessions().Refresh()
	eventually(t, "the evidence to go", func() bool { return !exists(evidenceDir(h, id)) })
	data, err := os.ReadFile(prs)
	if err != nil || strings.Contains(string(data), id) {
		t.Errorf("prs.json = %s, %v", data, err)
	}
	if exists(filepath.Join(h.State, "sessions", "main", "old1.json")) || len(h.Sessions().List()) != 0 {
		t.Error("the expired row stayed")
	}
}
