package issues_test

import (
	"strings"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
	ctl "github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/prompts"
)

func TestIssueCommand(t *testing.T) {
	h := newHarness(t)
	tests := []struct {
		name string
		args []string
		want string
		fail bool
	}{
		{"two issues", []string{"7", "#8"}, "started 2 sessions: #7 (1), #8 (2)", false},
		{"one running one new", []string{"7", "11"}, "started 1 session: #11 (3); already running: #7 (1)", false},
		{"some fail", []string{"x", "999", "8"}, "already running: #8 (2); failed: x (not a number), #999 (issue #999 is not open in this project)", false},
		{"all fail", []string{"x"}, "failed: x (not a number)", true},
		{"none", nil, "usage:", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := h.Ask(t, ctl.Request{Cmd: "issue", Args: tt.args})
			if resp.OK == tt.fail || !strings.Contains(resp.Out+resp.Error, tt.want) {
				t.Errorf("issue %v = %+v, want %q", tt.args, resp, tt.want)
			}
		})
	}
}

func TestIssueCommandHandsThePromptOver(t *testing.T) {
	dir := apptest.ClaudeOnPath(t)
	h := apptest.NewWith(t, apptest.Options{Agent: "claude"})
	resp := h.Ask(t, ctl.Request{Cmd: "issue", Args: []string{"7"}, Opts: map[string]string{"prompt": "the patch is in /tmp/x"}})
	if !resp.OK {
		t.Fatalf("issue 7 = %+v", resp)
	}
	list := h.Sessions().List()
	if len(list) != 1 || list[0].Issue != 7 {
		t.Fatalf("sessions = %+v, want one for #7", list)
	}
	if args := argsOf(t, dir, list[0].ID); !strings.Contains(args, prompts.Handover("the patch is in /tmp/x")) {
		t.Errorf("the system prompt lacks the handover:\n%s", args)
	}

	resp = h.Ask(t, ctl.Request{Cmd: "issue", Args: []string{"7"}, Opts: map[string]string{"prompt": "more"}})
	if !resp.OK || !strings.Contains(resp.Out, "already running: #7 (1, the prompt was not given to it)") {
		t.Errorf("issue 7 again = %+v", resp)
	}
}

func TestRefreshCommand(t *testing.T) {
	h := newHarness(t)
	if _, err := h.Issues(false); err != nil {
		t.Fatal(err)
	}
	resp := h.Ask(t, ctl.Request{Cmd: "refresh"})
	if !resp.OK || resp.Out != "refreshed 6 issues and the pull requests" || h.GH.IssueCalls() != 2 {
		t.Errorf("refresh = %+v, gh issue list ran %d times", resp, h.GH.IssueCalls())
	}
}
