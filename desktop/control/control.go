// Package control answers the agentos command line: each command is a handler that a service registers.
package control

import (
	"context"
	"errors"
	"fmt"
	"log"

	ctl "github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/session"
)

// Handler runs one command and returns what the command prints.
type Handler func(ctx context.Context, req ctl.Request) (string, error)

// Sessions tells who may ask.
type Sessions interface {
	Has(id string) bool
	Current() project.Project
}

// UICommand is what the front end is told to run, as the payload of the ui:command event.
type UICommand struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

// Router sends each command to its handler.
type Router struct {
	sessions Sessions
	switchTo func(name string) error
	emit     func(event string, payload any)
	handlers map[string]Handler
	server   *ctl.Server
}

// New routes commands. switchTo makes a project current; emit tells the front end.
func New(s Sessions, switchTo func(name string) error, emit func(string, any)) *Router {
	return &Router{sessions: s, switchTo: switchTo, emit: emit, handlers: map[string]Handler{}}
}

// Handle registers the handler of a command.
func (r *Router) Handle(cmd string, h Handler) { r.handlers[cmd] = h }

// Listen starts answering at the control socket in stateDir. Without it agents
// cannot use the app, which is logged and not fatal.
func (r *Router) Listen(stateDir string) {
	srv, err := ctl.Listen(ctl.SocketPath(stateDir), r.answer)
	if err != nil {
		log.Printf("agentos: agents cannot reach the app: %v", err)
		return
	}
	r.server = srv
}

// Close stops answering.
func (r *Router) Close() {
	if r.server != nil {
		r.server.Close()
	}
}

func (r *Router) answer(ctx context.Context, req ctl.Request) ctl.Response {
	h, ok := r.handlers[req.Cmd]
	if !ok {
		return ctl.Response{Error: fmt.Sprintf("unknown command %q", req.Cmd)}
	}
	out, err := h(ctx, req)
	if err != nil {
		return ctl.Response{Error: err.Error()}
	}
	return ctl.Response{OK: true, Out: out}
}

// AskerSession checks that the command comes from a session that runs.
func (r *Router) AskerSession(req ctl.Request) (string, error) {
	if _, err := session.ParseName(req.Session); err != nil {
		return "", errors.New("this command works inside an agentos session; AGENTOS_SESSION is not set")
	}
	if !r.sessions.Has(req.Session) {
		return "", fmt.Errorf("session %s is not running", req.Session)
	}
	return req.Session, nil
}

// AskerProject is the project key of the asking session, else the current project.
func (r *Router) AskerProject(req ctl.Request) string {
	if req.Project != "" {
		return req.Project
	}
	if name, err := session.ParseName(req.Session); err == nil {
		return name.Project
	}
	return r.sessions.Current().Key()
}

// Enter makes the project the command comes from the current one: the caller's
// own project, else its session's. The screen follows.
func (r *Router) Enter(req ctl.Request) {
	key := req.Project
	if name, err := session.ParseName(req.Session); key == "" && err == nil {
		key = name.Project
	}
	if key == "" || key == r.sessions.Current().Key() {
		return
	}
	_ = r.switchTo(key) // an unknown project leaves the current one
}

// Scope wraps a handler so it works in the project the command comes from.
func (r *Router) Scope(h Handler) Handler {
	return func(ctx context.Context, req ctl.Request) (string, error) {
		r.Enter(req)
		return h(ctx, req)
	}
}

// UI asks the front end to run a command in the project the request comes from, and answers ok.
func (r *Router) UI(req ctl.Request, name string, args ...string) string {
	r.Enter(req)
	if args == nil {
		args = []string{}
	}
	r.emit("ui:command", UICommand{Name: name, Args: args})
	return "ok"
}

// View is the handler of a command that only changes what the front end shows.
func (r *Router) View(name string) Handler {
	return func(_ context.Context, req ctl.Request) (string, error) {
		return r.UI(req, name, req.Args...), nil
	}
}
