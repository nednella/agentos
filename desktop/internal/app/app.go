package app

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/desktop/issues"
	"github.com/nednella/agentos/desktop/notes"
	"github.com/nednella/agentos/desktop/projects"
	"github.com/nednella/agentos/desktop/sessions"
	"github.com/nednella/agentos/desktop/stats"
	"github.com/nednella/agentos/desktop/terminal"
)

// Host is what the window provides. The services never touch Wails directly.
type Host struct {
	Emit      func(event string, payload any)
	Clipboard func(text string)
	PickDir   func() (string, error) // "" when the user cancels
}

// App is the services, wired together.
type App struct {
	ctx      atomic.Pointer[context.Context]
	sessions *sessions.Sessions
	terms    *terminal.Terms
	notes    *notes.Notes
	services []any
}

// New wires the services: who needs whom is decided here, and nowhere else.
func New(c Config, h Host, runner run.Runner) *App {
	a := &App{}
	ctx := func() context.Context {
		if c := a.ctx.Load(); c != nil {
			return *c
		}
		return context.Background()
	}
	waits := stats.New(c.DataDir)
	terms := terminal.New(c.Tmux, h.Emit, h.Clipboard)
	sess := sessions.New(sessions.Options{
		Tmux: c.Tmux, Agent: c.Agent, StateDir: c.StateDir, LocalDir: c.LocalDir, Projects: c.Registry,
		Current: c.Project, Emit: h.Emit, Run: runner, Tally: waits, CloseTerminal: terms.Close,
	})
	iss := issues.New(runner, sess, h.Emit)
	sess.Hook(iss.CachedRepo, iss.Emit)
	store := notes.New(c.DataDir)
	a.sessions, a.terms, a.notes = sess, terms, store
	a.services = []any{
		projects.NewService(c.Registry, sess, store, iss, c.StateDir, h.PickDir, ctx),
		sessions.NewService(sess, ctx),
		terminal.NewService(terms),
		issues.NewService(iss, ctx),
		notes.NewService(store, sess, sess, iss, runner, h.Emit, ctx),
		stats.NewService(waits, sess),
	}
	return a
}

// Services are the structs to bind to the front end.
func (a *App) Services() []any { return a.services }

// Start begins following the sessions; it runs until ctx ends.
func (a *App) Start(ctx context.Context) error {
	a.ctx.Store(&ctx)
	return a.sessions.Run(ctx)
}

// Stop closes the terminal streams.
func (a *App) Stop() { a.terms.CloseAll() }

// Media serves the pictures under /media/.
func (a *App) Media() http.Handler { return a.notes.MediaHandler() }

// Sessions is the session list, for tests and wiring.
func (a *App) Sessions() *sessions.Sessions { return a.sessions }

// Notes is the notes store, for tests.
func (a *App) Notes() *notes.Notes { return a.notes }
