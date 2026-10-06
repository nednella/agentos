package cli

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/internal/control"
)

func TestAppCommandsSendTheirWords(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	tests := []struct {
		args []string
		cmd  string
		want []string
		opts map[string]string
	}{
		{[]string{"issue", "394", "#393"}, "issue", []string{"394", "#393"}, nil},
		{[]string{"new"}, "new", nil, nil},
		{[]string{"new", "fix", "login"}, "new", []string{"fix", "login"}, nil},
		{[]string{"open", "3"}, "open", []string{"3"}, nil},
		{[]string{"open", "the", "login"}, "open", []string{"the", "login"}, nil},
		{[]string{"kill", "4"}, "kill", []string{"4"}, nil},
		{[]string{"project"}, "project", nil, nil},
		{[]string{"project", "other"}, "project", []string{"other"}, nil},
		{[]string{"project", "add"}, "project", []string{"add", dir}, nil},
		{[]string{"project", "remove", "other"}, "project", []string{"remove", "other"}, nil},
		{[]string{"next"}, "next", nil, nil},
		{[]string{"filter"}, "filter", nil, nil},
		{[]string{"filter", "@me", "type:bug"}, "filter", []string{"@me", "type:bug"}, nil},
		{[]string{"queue"}, "queue", nil, nil},
		{[]string{"notes"}, "notes", nil, nil},
		{[]string{"evidence"}, "evidence", nil, nil},
		{[]string{"term"}, "term", nil, nil},
		{[]string{"refresh"}, "refresh", nil, nil},
		{[]string{"pr"}, "pr", nil, nil},
		{[]string{"pr", "2"}, "pr", []string{"2"}, nil},
		{[]string{"cleanup", "2"}, "cleanup", []string{"2"}, nil},
		{[]string{"digest"}, "digest", nil, map[string]string{}},
		{[]string{"digest", "--run"}, "digest", nil, map[string]string{"run": "1"}},
		{[]string{"stats", "--open"}, "stats", nil, map[string]string{"open": "1", "days": "7"}},
		{[]string{"browser"}, "browser", nil, nil},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			var got control.Request
			fakeApp(t, func(_ context.Context, req control.Request) control.Response {
				got = req
				return control.Response{OK: true, Out: "answer"}
			})
			if out, err := run(t, tt.args...); err != nil || out != "answer" {
				t.Fatalf("agentos %v = %q, %v", tt.args, out, err)
			}
			if got.Cmd != tt.cmd || !slices.Equal(got.Args, tt.want) {
				t.Errorf("request = %+v, want %s %v", got, tt.cmd, tt.want)
			}
			for k, v := range tt.opts {
				if got.Opts[k] != v {
					t.Errorf("option %s = %q, want %q", k, got.Opts[k], v)
				}
			}
		})
	}
}

func TestAppCommandsRejectBadArguments(t *testing.T) {
	fakeApp(t, func(context.Context, control.Request) control.Response {
		t.Error("the app was asked")
		return control.Response{OK: true}
	})
	for _, args := range [][]string{
		{"issue"}, {"open"}, {"next", "x"}, {"queue", "x"}, {"pr", "1", "2"}, {"kill", "1", "2"},
		{"project", "remove"}, {"project", "add", "a", "b"},
		{"project", "add", "/no/such/folder"},
	} {
		if _, err := run(t, args...); err == nil {
			t.Errorf("agentos %v succeeded", args)
		}
	}
}

func TestAppCommandsNeedTheApp(t *testing.T) {
	t.Setenv("AGENTOS_SOCKET", "/nonexistent/agentos.sock")
	for _, args := range [][]string{{"issue", "1"}, {"new"}, {"queue"}, {"project"}, {"kill", "1"}, {"browser"}} {
		if _, err := run(t, args...); err == nil || err.Error() != "agentos is not running: open the app and try again" {
			t.Errorf("agentos %v: error = %v", args, err)
		}
	}
}

func TestAppCommandsAnswerWithTheAppsError(t *testing.T) {
	fakeApp(t, func(context.Context, control.Request) control.Response {
		return control.Response{Error: "no session 9 in this project"}
	})
	if _, err := run(t, "kill", "9"); err == nil || err.Error() != "no session 9 in this project" {
		t.Errorf("error = %v", err)
	}
}

func TestRootGroupsTheCommands(t *testing.T) {
	out, err := run(t, "--help")
	if err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"Work:", "Views:", "Projects:", "From inside a session:"} {
		if !strings.Contains(out, title) {
			t.Errorf("help has no %q group:\n%s", title, out)
		}
	}
	if strings.Contains(out, "digest add") || strings.Contains(out, "hook") {
		t.Errorf("help shows a hidden command:\n%s", out)
	}
	if !strings.Contains(out, "digest") {
		t.Errorf("help does not list digest:\n%s", out)
	}
}
