package sessions_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/desktop/control"
	ctl "github.com/nednella/agentos/internal/control"
)

func TestSessionCommands(t *testing.T) {
	h := newHarness(t)

	if resp := h.Ask(t, ctl.Request{Cmd: "new", Args: []string{"fix", "login"}}); !resp.OK || resp.Out != "started session 1: fix login" {
		t.Fatalf("new = %+v", resp)
	}
	if resp := h.Ask(t, ctl.Request{Cmd: "new", Args: []string{"fix logout"}}); !resp.OK || resp.Out != "started session 2: fix logout" {
		t.Fatalf("second new = %+v", resp)
	}

	tests := []struct {
		name string
		args []string
		want string // the UI command opened, or the error
		fail bool
	}{
		{"by number", []string{"2"}, "2", false},
		{"by hash number", []string{"#1"}, "#1", false},
		{"by title", []string{"login"}, "login", false},
		{"ambiguous title", []string{"fix"}, "2 sessions match", true},
		{"unknown number", []string{"9"}, "no session 9", true},
		{"unknown title", []string{"nothing"}, "no session matching", true},
		{"nothing", nil, "usage:", true},
	}
	for _, tt := range tests {
		t.Run("open "+tt.name, func(t *testing.T) {
			resp := h.Ask(t, ctl.Request{Cmd: "open", Args: tt.args})
			if tt.fail {
				if resp.OK || !strings.Contains(resp.Error, tt.want) {
					t.Errorf("open = %+v, want error with %q", resp, tt.want)
				}
				return
			}
			want := control.UICommand{Name: "open", Args: []string{tt.want}}
			if got := h.Rec.LastUI(); !resp.OK || resp.Out != "ok" || got.Name != want.Name || !slices.Equal(got.Args, want.Args) {
				t.Errorf("open = %+v, ui %+v, want %+v", resp, got, want)
			}
		})
	}

	if resp := h.Ask(t, ctl.Request{Cmd: "kill", Args: []string{"2"}}); !resp.OK || resp.Out != "killed session 2: fix logout" {
		t.Errorf("kill = %+v", resp)
	}
	for _, bad := range [][]string{nil, {"x"}, {"9"}, {"2"}} {
		if resp := h.Ask(t, ctl.Request{Cmd: "kill", Args: bad}); resp.OK {
			t.Errorf("kill %v succeeded", bad)
		}
	}
}

func TestPRAndCleanupCommands(t *testing.T) {
	h := newHarness(t)
	issueSession(h, t, 12)
	if _, err := h.NewSession("plain", ""); err != nil {
		t.Fatal(err)
	}

	if resp := h.Ask(t, ctl.Request{Cmd: "pr"}); !resp.OK || resp.Out != "no pull requests" {
		t.Errorf("pr before any = %+v", resp)
	}
	if resp := h.Ask(t, ctl.Request{Cmd: "pr", Args: []string{"1"}}); !resp.OK || resp.Out != "1: no pull request for issue-12" {
		t.Errorf("pr 1 = %+v", resp)
	}
	h.GH.SetPR(prJSON("OPEN", false, "[]", 2, 0))
	h.RefreshPRs()
	want := "1: #12 open, checks none, 2 comments, https://github.com/acme/widgets/pull/12"
	if resp := h.Ask(t, ctl.Request{Cmd: "pr"}); !resp.OK || resp.Out != want {
		t.Errorf("pr = %+v, want %q", resp, want)
	}
	if resp := h.Ask(t, ctl.Request{Cmd: "pr", Args: []string{"x"}}); resp.OK {
		t.Error("pr x succeeded")
	}

	if resp := h.Ask(t, ctl.Request{Cmd: "cleanup"}); !resp.OK || resp.Out != "nothing to clean up" {
		t.Errorf("cleanup list = %+v", resp)
	}
	for _, bad := range [][]string{{"9"}, {"x"}, {"2"}} { // no such session, not a number, a session without an issue
		if resp := h.Ask(t, ctl.Request{Cmd: "cleanup", Args: bad}); resp.OK {
			t.Errorf("cleanup %v succeeded", bad)
		}
	}
}

func TestCommandsWorkInTheCallersProject(t *testing.T) {
	h := newHarness(t)
	root := t.TempDir()
	if _, err := h.AddProjectDir(root); err != nil {
		t.Fatal(err)
	}
	other := h.Snapshot().Project.Name
	if _, err := h.SwitchProject("main"); err != nil {
		t.Fatal(err)
	}

	resp := h.Ask(t, ctl.Request{Cmd: "new", Project: other, Args: []string{"elsewhere"}})
	if !resp.OK || h.Snapshot().Project.Name != other {
		t.Fatalf("new = %+v, project now %q", resp, h.Snapshot().Project.Name)
	}
	eventually(t, "the session in the other project", func() bool {
		for _, s := range h.Sessions().List() {
			if s.Title == "elsewhere" {
				return true
			}
		}
		return false
	})
}
