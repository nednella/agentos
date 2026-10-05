package app

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/nednella/agentos/desktop/browser"
	"github.com/nednella/agentos/desktop/control"
	"github.com/nednella/agentos/desktop/evidence"
	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/desktop/issues"
	"github.com/nednella/agentos/desktop/notes"
	"github.com/nednella/agentos/desktop/projects"
	"github.com/nednella/agentos/desktop/sessions"
	"github.com/nednella/agentos/desktop/stats"
	"github.com/nednella/agentos/desktop/terminal"
	"github.com/nednella/agentos/internal/session"
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
	evidence *evidence.Store
	browsers *browser.Browsers
	router   *control.Router
	stateDir string
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
	proofs := evidence.New(c.LocalDir)
	browsers := browser.New(c.LocalDir, h.Emit)
	terms := terminal.New(c.Tmux, h.Emit, h.Clipboard)
	sess := sessions.New(sessions.Options{
		Tmux: c.Tmux, Agent: c.Agent, StateDir: c.StateDir, LocalDir: c.LocalDir, Projects: c.Registry,
		Current: c.Project, Emit: h.Emit, Run: runner, Tally: waits, CloseTerminal: terms.Close, Evidence: proofs, Browsers: browsers,
	})
	iss := issues.New(runner, sess, h.Emit)
	sess.Hook(iss.CachedRepo, iss.Emit)
	store := notes.New(c.DataDir)
	a.sessions, a.terms, a.notes, a.evidence, a.stateDir = sess, terms, store, proofs, c.StateDir
	a.browsers = browsers
	browsers.Hook(func(id string) string {
		name, err := session.ParseName(id)
		if err != nil {
			return ""
		}
		for _, p := range c.Registry.List() {
			if p.Key() == name.Project {
				return p.URL
			}
		}
		return ""
	}, sess.Touch)
	a.router = control.New(sess)
	a.router.Handle("note", notes.NewCommands(store, a.router, sess, h.Emit).Note)
	changes := evidence.Changes{Emit: h.Emit, Touch: sess.Touch}
	a.router.Handle("show", evidence.NewCommands(proofs, a.router, changes).Show)
	a.router.Handle("stats", stats.NewCommands(waits, a.router).Stats)
	a.services = []any{
		projects.NewService(c.Registry, sess, store, iss, c.StateDir, h.PickDir, ctx),
		sessions.NewService(sess, ctx),
		terminal.NewService(terms),
		issues.NewService(iss, ctx),
		notes.NewService(store, sess, sess, iss, runner, h.Emit, ctx),
		evidence.NewService(proofs, changes),
		browser.NewService(browsers, proofs, changes, ctx),
		stats.NewService(waits, sess),
	}
	return a
}

// Services are the structs to bind to the front end.
func (a *App) Services() []any { return a.services }

// Start begins following the sessions; it runs until ctx ends.
func (a *App) Start(ctx context.Context) error {
	a.ctx.Store(&ctx)
	if err := a.sessions.Run(ctx); err != nil {
		return err
	}
	a.router.Listen(a.stateDir)
	return nil
}

// Stop closes the terminal streams and stops answering the command line.
func (a *App) Stop() {
	a.terms.CloseAll()
	a.browsers.CloseAll()
	a.router.Close()
}

// Media serves the pictures of notes and evidence under /media/<project key>/<folder>/.
func (a *App) Media() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/evidence/") {
			a.evidence.MediaHandler().ServeHTTP(w, r)
			return
		}
		a.notes.MediaHandler().ServeHTTP(w, r)
	})
}

// Sessions is the session list, for tests and wiring.
func (a *App) Sessions() *sessions.Sessions { return a.sessions }

// Notes is the notes store, for tests.
func (a *App) Notes() *notes.Notes { return a.notes }

// Browsers are the hidden browsers, for tests.
func (a *App) Browsers() *browser.Browsers { return a.browsers }
