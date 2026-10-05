package projects_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/desktop/control"
	ctl "github.com/nednella/agentos/internal/control"
)

func TestProjectCommand(t *testing.T) {
	h := newHarness(t)
	dir := filepath.Join(t.TempDir(), "side")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	ask := func(args ...string) ctl.Response { return h.Ask(t, ctl.Request{Cmd: "project", Args: args}) }

	if resp := ask(); !resp.OK || resp.Out != "* main  "+h.Dir+"  0 running" {
		t.Errorf("list = %+v", resp)
	}
	if resp := ask("add", dir); !resp.OK || resp.Out != "added project side" || h.Snapshot().Project.Name != "side" {
		t.Errorf("add = %+v", resp)
	}
	if resp := ask(); !resp.OK || !strings.Contains(resp.Out, "  main  ") || !strings.Contains(resp.Out, "* side  "+dir) {
		t.Errorf("list after add = %+v", resp)
	}
	if resp := ask("main"); !resp.OK || resp.Out != "project main" || h.Snapshot().Project.Name != "main" {
		t.Errorf("switch = %+v", resp)
	}
	if resp := ask("remove", "side"); !resp.OK || resp.Out != "removed project side" || len(h.Snapshot().Projects) != 1 {
		t.Errorf("remove = %+v", resp)
	}
	for _, bad := range [][]string{{"nowhere"}, {"add"}, {"add", filepath.Join(dir, "gone")}, {"remove"}, {"remove", "nowhere"}} {
		if resp := ask(bad...); resp.OK {
			t.Errorf("project %v succeeded", bad)
		}
	}
}

func TestViewCommands(t *testing.T) {
	h := newHarness(t)
	dir := filepath.Join(t.TempDir(), "side")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := h.AddProjectDir(dir); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		req  ctl.Request
		want control.UICommand
	}{
		{ctl.Request{Cmd: "queue"}, control.UICommand{Name: "queue", Args: []string{}}},
		{ctl.Request{Cmd: "notes"}, control.UICommand{Name: "notes", Args: []string{}}},
		{ctl.Request{Cmd: "evidence"}, control.UICommand{Name: "evidence", Args: []string{}}},
		{ctl.Request{Cmd: "term"}, control.UICommand{Name: "term", Args: []string{}}},
		{ctl.Request{Cmd: "next"}, control.UICommand{Name: "next", Args: []string{}}},
		{ctl.Request{Cmd: "filter", Args: []string{"@me", "type:bug"}}, control.UICommand{Name: "filter", Args: []string{"@me", "type:bug"}}},
	}
	for _, tt := range tests {
		t.Run(tt.req.Cmd, func(t *testing.T) {
			if _, err := h.SwitchProject("side"); err != nil {
				t.Fatal(err)
			}
			tt.req.Project = "main"
			resp := h.Ask(t, tt.req)
			got := h.Rec.LastUI()
			if !resp.OK || resp.Out != "ok" || got.Name != tt.want.Name || !slices.Equal(got.Args, tt.want.Args) {
				t.Errorf("%s = %+v, ui %+v, want %+v", tt.req.Cmd, resp, got, tt.want)
			}
			if name := h.Snapshot().Project.Name; name != "main" {
				t.Errorf("the screen stayed on %q, not the caller's project", name)
			}
		})
	}
}
