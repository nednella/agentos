package issues_test

import (
	"strings"
	"testing"

	ctl "github.com/nednella/agentos/internal/control"
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
