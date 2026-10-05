package app

import (
	"context"
	"net/http"
	"sync/atomic"

	"github.com/nednella/agentos/desktop/browser"
	"github.com/nednella/agentos/desktop/control"
	"github.com/nednella/agentos/desktop/digest"
	"github.com/nednella/agentos/desktop/evidence"
	"github.com/nednella/agentos/desktop/internal/media"
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
	digests  *digest.Manager
	router   *control.Router
	media    http.Handler
	stateDir string
	services []any
}

// New wires the services: who needs whom is decided here, and nowhere else.
func New(c Config, h Host, runner run.Runner, claude run.EnvRunner) *App {
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
	a.media = media.Handler(c.DataDir, c.LocalDir)
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
	mgr := digest.NewManager(digest.New(c.DataDir), sess, store, claude, h.Emit, c.StateDir, ctx)
	a.digests = mgr
	proj := projects.NewService(c.Registry, sess, store, iss, c.StateDir, h.PickDir, ctx)
	sessSvc := sessions.NewService(sess, ctx)
	a.router = control.New(sess, func(name string) error { _, err := proj.SwitchProject(name); return err }, h.Emit)
	changes := evidence.Changes{Emit: h.Emit, Touch: sess.Touch}
	digests, sc, ic := digest.NewCommands(mgr, a.router), sessions.NewCommands(sess, a.router), issues.NewCommands(iss, sessSvc)
	a.handle(map[string]control.Handler{
		"digest-add": digests.Add,
		"digest":     digests.Digest,
		"note":       notes.NewCommands(store, a.router, sess, h.Emit).Note,
		"show":       evidence.NewCommands(proofs, a.router, changes).Show,
		"browser":    browser.NewCommands(browsers, a.router, a.router, proofs, changes).Browser,
		"stats":      stats.NewCommands(waits, a.router, a.router).Stats,
		"project":    projects.NewCommands(proj).Project,
	})
	a.handleScoped(map[string]control.Handler{
		"issue":   ic.Issue,
		"refresh": ic.Refresh,
		"new":     sc.New,
		"open":    sc.Open,
		"kill":    sc.Kill,
		"pr":      sc.PR,
		"cleanup": sc.Cleanup,
		"harness": sc.Harness,
	})
	for _, view := range []string{"queue", "notes", "evidence", "term", "next", "filter"} {
		a.router.Handle(view, a.router.View(view))
	}
	a.services = []any{
		proj,
		sessSvc,
		terminal.NewService(terms),
		issues.NewService(iss, ctx),
		notes.NewService(store, sess, sess, iss, runner, h.Emit, ctx),
		evidence.NewService(proofs, changes),
		browser.NewService(browsers, proofs, changes, ctx),
		digest.NewService(mgr),
		stats.NewService(waits, sess),
	}
	return a
}

func (a *App) handle(handlers map[string]control.Handler) {
	for cmd, h := range handlers {
		a.router.Handle(cmd, h)
	}
}

// handleScoped registers commands that work in the project they come from.
func (a *App) handleScoped(handlers map[string]control.Handler) {
	for cmd, h := range handlers {
		a.router.Handle(cmd, a.router.Scope(h))
	}
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
	go a.digests.Loop(ctx)
	return nil
}

// Stop closes the terminal streams and stops answering the command line.
func (a *App) Stop() {
	a.terms.CloseAll()
	a.browsers.CloseAll()
	a.sessions.Stop()
	a.router.Close()
}

// Media serves the pictures of notes and evidence under /media/<project key>/<folder>/.
func (a *App) Media() http.Handler { return a.media }

// Sessions is the session list, for tests and wiring.
func (a *App) Sessions() *sessions.Sessions { return a.sessions }

// Notes is the notes store, for tests.
func (a *App) Notes() *notes.Notes { return a.notes }

// Browsers are the hidden browsers, for tests.
func (a *App) Browsers() *browser.Browsers { return a.browsers }

// Digests is the digest manager, for tests.
func (a *App) Digests() *digest.Manager { return a.digests }
