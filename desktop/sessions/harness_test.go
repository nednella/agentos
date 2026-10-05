package sessions_test

import (
	"strings"
	"testing"
)

func TestHarnessCheckTypesThePromptWithoutSendingIt(t *testing.T) {
	h := newHarness(t)
	s, err := h.HarnessCheck()
	if err != nil || s.Title != "Harness check" {
		t.Fatalf("HarnessCheck = %+v, %v", s, err)
	}
	eventually(t, "the prompt typed in", func() bool {
		pane := h.Pane(t, s.ID)
		return strings.Contains(pane, "Review this project") && strings.Contains(pane, "agentos note")
	})
	if pane := h.Pane(t, s.ID); strings.Contains(pane, "not found") {
		t.Errorf("the prompt was submitted:\n%s", pane)
	}
}
