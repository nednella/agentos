package control

import (
	"context"
	"errors"
	"os"
	"reflect"
	"slices"
	"testing"

	ctl "github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/project"
)

type fakeSessions struct{ running map[string]bool }

func (f fakeSessions) Has(id string) bool       { return f.running[id] }
func (f fakeSessions) Current() project.Project { return project.Project{Name: "Current One"} }

func TestRouterAnswers(t *testing.T) {
	dir, err := os.MkdirTemp("", "aosrt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	r := New(fakeSessions{}, nil, nil)
	r.Handle("echo", func(_ context.Context, req ctl.Request) (string, error) { return "got " + req.Args[0], nil })
	r.Handle("fail", func(context.Context, ctl.Request) (string, error) { return "", errors.New("it failed") })
	r.Listen(dir)
	defer r.Close()

	for _, tt := range []struct {
		req  ctl.Request
		want ctl.Response
	}{
		{ctl.Request{Cmd: "echo", Args: []string{"x"}}, ctl.Response{OK: true, Out: "got x"}},
		{ctl.Request{Cmd: "fail"}, ctl.Response{Error: "it failed"}},
		{ctl.Request{Cmd: "nope"}, ctl.Response{Error: `unknown command "nope"`}},
	} {
		got, err := ctl.Call(context.Background(), ctl.SocketPath(dir), tt.req)
		if err != nil || got != tt.want {
			t.Errorf("%s: %+v, %v; want %+v", tt.req.Cmd, got, err, tt.want)
		}
	}
}

func TestAskers(t *testing.T) {
	r := New(fakeSessions{running: map[string]bool{"p/1": true}}, nil, nil)
	if id, err := r.AskerSession(ctl.Request{Session: "p/1"}); err != nil || id != "p/1" {
		t.Errorf("AskerSession = %q, %v", id, err)
	}
	for _, bad := range []string{"", "p/2", "not a session"} {
		if _, err := r.AskerSession(ctl.Request{Session: bad}); err == nil {
			t.Errorf("session %q was accepted", bad)
		}
	}
	tests := []struct {
		req  ctl.Request
		want string
	}{
		{ctl.Request{Project: "digest-project", Session: "p/1"}, "digest-project"},
		{ctl.Request{Session: "p/1"}, "p"},
		{ctl.Request{}, "current-one"},
	}
	for _, tt := range tests {
		if got := r.AskerProject(tt.req); got != tt.want {
			t.Errorf("AskerProject(%+v) = %q, want %q", tt.req, got, tt.want)
		}
	}
}

func TestRouterEntersTheAskersProject(t *testing.T) {
	tests := []struct {
		name string
		req  ctl.Request
		want []string
	}{
		{"caller project", ctl.Request{Project: "other", Session: "p/1"}, []string{"other"}},
		{"session project", ctl.Request{Session: "p/1"}, []string{"p"}},
		{"current project", ctl.Request{}, nil},
		{"already there", ctl.Request{Project: "current-one"}, nil},
		{"not a session", ctl.Request{Session: "nope"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var switched []string
			r := New(fakeSessions{}, func(name string) error {
				switched = append(switched, name)
				return errors.New("no such project")
			}, nil)
			r.Enter(tt.req)
			if !slices.Equal(switched, tt.want) {
				t.Errorf("switched to %v, want %v", switched, tt.want)
			}
		})
	}
}

func TestRouterViewsAndScopes(t *testing.T) {
	var switched []string
	var events []any
	r := New(fakeSessions{}, func(name string) error { switched = append(switched, name); return nil },
		func(event string, payload any) {
			if event != "ui:command" {
				t.Errorf("event = %q", event)
			}
			events = append(events, payload)
		})
	req := ctl.Request{Project: "other", Args: []string{"@me", "type:bug"}}
	if out, err := r.View("filter")(context.Background(), req); err != nil || out != "ok" {
		t.Errorf("View = %q, %v", out, err)
	}
	if out, _ := r.View("queue")(context.Background(), ctl.Request{}); out != "ok" {
		t.Errorf("View without args = %q", out)
	}
	want := []any{UICommand{Name: "filter", Args: []string{"@me", "type:bug"}}, UICommand{Name: "queue", Args: []string{}}}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("events = %v, want %v", events, want)
	}
	ran := false
	scoped := r.Scope(func(context.Context, ctl.Request) (string, error) { ran = true; return "done", nil })
	if out, err := scoped(context.Background(), ctl.Request{Project: "third"}); err != nil || out != "done" || !ran {
		t.Errorf("Scope = %q, %v", out, err)
	}
	if !slices.Equal(switched, []string{"other", "third"}) {
		t.Errorf("switched = %v", switched)
	}
}
