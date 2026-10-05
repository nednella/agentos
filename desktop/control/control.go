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

// Router sends each command to its handler.
type Router struct {
	sessions Sessions
	handlers map[string]Handler
	server   *ctl.Server
}

func New(s Sessions) *Router { return &Router{sessions: s, handlers: map[string]Handler{}} }

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
