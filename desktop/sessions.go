package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/guard"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/session"
	"github.com/nednella/agentos/internal/term"
)

const (
	pollEvery     = 2 * time.Second
	tmuxTimeout   = 5 * time.Second
	defaultTitle  = "agent"
	historyCap    = 200
	startCols     = 120
	startRows     = 32
	prefillWait   = 6 * time.Second
	promptSettles = 300 * time.Millisecond
)

// Change is one entry of a session's state timeline.
type Change struct {
	State session.State `json:"state"`
	At    int64         `json:"at"`
}

// Session is one agent as the front end sees it.
type Session struct {
	ID          string        `json:"id"`
	N           int           `json:"n"`
	Title       string        `json:"title"`
	State       session.State `json:"state"`
	Detail      string        `json:"detail"`
	LastEventAt int64         `json:"lastEventAt"`
	CreatedAt   int64         `json:"createdAt"`
	Issue       int           `json:"issue"`
	History     []Change      `json:"history"`

	Branch        string `json:"branch"`
	Worktree      string `json:"worktree"`
	PR            *PR    `json:"pr"`
	PRAttention   string `json:"prAttention"`
	Cleanup       string `json:"cleanup"`
	CleanupReason string `json:"cleanupReason"`

	Browser  bool `json:"browser"`
	Evidence int  `json:"evidence"`
}

// Project is a project as the front end sees it, with how its sessions stand.
type Project struct {
	Name     string `json:"name"`
	Dir      string `json:"dir"`
	Repo     string `json:"repo"`
	NeedsYou int    `json:"needsYou"`
	Working  int    `json:"working"`
	Sessions int    `json:"sessions"`
}

// Sessions tracks the agents in the private tmux server and tells the front end when they change.
type Sessions struct {
	tmux     *term.Tmux
	agent    agent.Agent
	stateDir string
	emit     func(event string, payload any)

	createMu sync.Mutex // one creation at a time, so two sessions never pick the same number

	mu         sync.Mutex
	ctx        context.Context
	project    project.Project
	registry   *Registry
	waits      *Waits
	browsers   *Browsers
	ev         *EvidenceStore
	onGone     func(id string)
	life       *Lifecycle
	repoOf     func(dir string) string
	onIssues   func()
	info       []term.Info
	records    map[string]session.Record
	seen       map[string]session.State
	history    map[string][]Change
	loaded     bool // the first refresh is in, so later changes may raise attention
	lastSent   string
	lastProjs  string
	lastIssues string
	ready      map[string]chan struct{} // closed when the agent's SessionStart arrives
	prefillFor time.Duration
}

func newSessions(tmux *term.Tmux, ag agent.Agent, stateDir, dataDir string, registry *Registry, current project.Project, emit func(string, any)) *Sessions {
	return &Sessions{
		tmux: tmux, agent: ag, stateDir: stateDir, emit: emit,
		project: current, registry: registry, waits: newWaits(dataDir),
		repoOf: func(string) string { return "" }, onIssues: func() {}, onGone: func(string) {},
		records:    map[string]session.Record{},
		seen:       map[string]session.State{},
		history:    map[string][]Change{},
		ready:      map[string]chan struct{}{},
		prefillFor: prefillWait,
		ctx:        context.Background(),
	}
}

// Start listens for hooks and polls tmux until ctx ends. A hook socket held by
// another agentos leaves only the poll.
func (s *Sessions) Start(ctx context.Context) error {
	s.mu.Lock()
	s.ctx = ctx
	s.mu.Unlock()
	ln, err := bus.Listen(bus.SocketPath(s.stateDir), s.observe)
	if err != nil && !errors.Is(err, bus.ErrInUse) {
		return fmt.Errorf("listening for hooks: %w", err)
	}
	s.Refresh()
	go func() {
		if ln != nil {
			defer ln.Close()
		}
		tick := time.NewTicker(pollEvery)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				s.Refresh()
			}
		}
	}()
	return nil
}

// Refresh reads tmux and the state files, and sweeps files of sessions that no longer exist.
func (s *Sessions) Refresh() {
	s.mu.Lock()
	ctx := s.ctx
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, tmuxTimeout)
	defer cancel()
	infos, err := s.tmux.List(ctx)
	if err != nil {
		return
	}
	records := bus.ReadAll(s.stateDir)
	for name := range records {
		if !slices.ContainsFunc(infos, func(i term.Info) bool { return i.Name.String() == name }) {
			_ = bus.RemoveState(s.stateDir, name)
			delete(records, name)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.info = infos
	for _, rec := range records {
		s.take(rec)
	}
	s.loaded = true
	s.sync()
}

func (s *Sessions) observe(rec session.Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.take(rec)
	s.sync()
}

// take folds a record in. It needs mu.
func (s *Sessions) take(rec session.Record) {
	if rec.State == session.Gone {
		delete(s.records, rec.Session)
		return
	}
	if prev, ok := s.records[rec.Session]; ok && prev.At.After(rec.At) {
		return
	}
	s.records[rec.Session] = rec
	if ch, ok := s.ready[rec.Session]; ok && rec.Event == "SessionStart" {
		close(ch)
		delete(s.ready, rec.Session)
	}
}

// sync updates history and attention from what is known, and tells the front
// end if the list changed. It needs mu.
func (s *Sessions) sync() {
	key := s.project.Key()
	var attention []Session
	waitsChanged := false
	live := map[string]bool{}
	for _, in := range s.info {
		id := in.Name.String()
		live[id] = true
		state, at := session.Idle, time.Now()
		if rec, ok := s.records[id]; ok {
			state, at = rec.State, rec.At
		}
		prev, known := s.seen[id]
		if known && prev == state {
			continue
		}
		s.seen[id] = state
		s.history[id] = append(s.history[id], Change{State: state, At: at.UnixMilli()})
		if h := s.history[id]; len(h) > historyCap {
			s.history[id] = slices.Clone(h[len(h)-historyCap:])
		}
		if s.loaded && state.NeedsUser() && in.Name.Project == key {
			attention = append(attention, Session{ID: id, State: state})
		}
		if s.loaded {
			ended := s.waits.End(id, at)
			if state.NeedsUser() {
				kind, label := causeOf(s.records[id])
				issue, _ := strconv.Atoi(in.Issue)
				s.waits.Begin(id, Wait{SessionTitle: cmp.Or(in.Title, defaultTitle), Issue: issue, Kind: kind, Label: label, StartedAt: at.UnixMilli()})
			}
			if ended || state.NeedsUser() {
				waitsChanged = true
			}
		}
	}
	for id := range s.seen {
		if !live[id] {
			if s.waits.End(id, time.Now()) {
				waitsChanged = true
			}
			s.life.forget(id)
			go s.onGone(id)
			delete(s.seen, id)
			delete(s.history, id)
			delete(s.records, id)
		}
	}
	if waitsChanged {
		s.emit("stats", nil)
	}

	s.announceProjects()
	s.announceIssues()
	list := s.list()
	b, err := json.Marshal(list)
	if err != nil || string(b) == s.lastSent {
		return
	}
	s.lastSent = string(b)
	s.emit("sessions", list)
	for _, a := range attention {
		s.emit("attention", map[string]any{"id": a.ID, "state": a.State})
	}
}

// announceProjects tells the front end when a project appears, goes, or its counts move. It needs mu.
func (s *Sessions) announceProjects() {
	views := s.projectViews()
	b, err := json.Marshal(views)
	if err != nil || string(b) == s.lastProjs {
		return
	}
	s.lastProjs = string(b)
	s.emit("projects", views)
}

// announceIssues asks for a fresh issues event when the sessions tied to issues change. It needs mu.
func (s *Sessions) announceIssues() {
	var sig strings.Builder
	sig.WriteString(s.project.Key())
	for _, v := range s.list() {
		if v.Issue > 0 {
			fmt.Fprintf(&sig, " %d=%s", v.Issue, v.ID)
		}
	}
	if sig.String() == s.lastIssues {
		return
	}
	first := s.lastIssues == ""
	s.lastIssues = sig.String()
	if !first {
		go s.onIssues()
	}
}

// projectViews lists every known project with its session counts. It needs mu.
func (s *Sessions) projectViews() []Project {
	var views []Project
	for _, p := range s.projects() {
		v := Project{Name: p.Name, Dir: p.Dir, Repo: s.repoOf(p.Dir)}
		for _, in := range s.info {
			if in.Name.Project != p.Key() {
				continue
			}
			v.Sessions++
			state := session.Idle
			if rec, ok := s.records[in.Name.String()]; ok {
				state = rec.State
			}
			switch {
			case state.NeedsUser():
				v.NeedsYou++
			case state == session.Working:
				v.Working++
			}
		}
		views = append(views, v)
	}
	return views
}

// list is the current project's sessions in rail order. It needs mu.
func (s *Sessions) list() []Session {
	key := s.project.Key()
	var rail []session.Session
	byName := map[session.Name]term.Info{}
	for _, in := range s.info {
		if in.Name.Project != key {
			continue
		}
		byName[in.Name] = in
		ss := session.Session{Name: in.Name, State: session.Idle}
		if rec, ok := s.records[in.Name.String()]; ok {
			ss.State, ss.At, ss.Detail = rec.State, rec.At, rec.Detail
		}
		rail = append(rail, ss)
	}
	session.Sort(rail)
	out := make([]Session, 0, len(rail))
	for _, ss := range rail {
		in := byName[ss.Name]
		id := ss.Name.String()
		issue, _ := strconv.Atoi(in.Issue)
		view := Session{
			ID: id, N: ss.Name.N, Title: cmp.Or(in.Title, defaultTitle),
			State: ss.State, Detail: ss.Detail,
			CreatedAt: in.Created.UnixMilli(), Issue: issue,
			Browser: s.browsers.Has(id), Evidence: s.ev.Count(id),
			History: slices.Clone(s.history[id]),
		}
		if !ss.At.IsZero() {
			view.LastEventAt = ss.At.UnixMilli()
		}
		if issue > 0 {
			f := s.life.fields(id)
			view.Branch, view.Worktree, view.PR = s.project.BranchFor(issue), f.worktree, f.pr
			view.PRAttention, view.Cleanup, view.CleanupReason = f.attention, f.cleanup, f.reason
		}
		if view.History == nil {
			view.History = []Change{}
		}
		out = append(out, view)
	}
	return out
}

func (s *Sessions) List() []Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.list()
}

func (s *Sessions) Current() project.Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.project
}

// Projects lists every known project: the configured ones, the current one, and any other with live sessions.
func (s *Sessions) Projects() []Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.projectViews()
}

func (s *Sessions) projects() []project.Project {
	out := s.registry.List()
	if !slices.ContainsFunc(out, func(p project.Project) bool { return p.Key() == s.project.Key() }) {
		out = append(out, s.project)
	}
	for _, in := range s.info {
		if !slices.ContainsFunc(out, func(p project.Project) bool { return p.Key() == in.Name.Project }) {
			out = append(out, project.Project{Name: in.Name.Project, Dir: in.Path})
		}
	}
	return out
}

// SwitchProject makes the named project current and returns it.
func (s *Sessions) SwitchProject(name string) (project.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.projects() {
		if strings.EqualFold(p.Name, name) || p.Key() == name {
			s.project = p
			s.sync()
			return p, nil
		}
	}
	return project.Project{}, fmt.Errorf("no project %q", name)
}

// Forget removes a configured project, moving to another one if it was current.
func (s *Sessions) Forget(name string) (project.Project, error) {
	removed, err := s.registry.Remove(name)
	if err != nil {
		return project.Project{}, err
	}
	if !removed {
		return project.Project{}, fmt.Errorf("%q is not in the config", name)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.project.Name == name {
		for _, p := range s.projects() {
			if p.Key() != s.project.Key() {
				s.project = p
				break
			}
		}
	}
	s.sync()
	return s.project, nil
}

// IssueSessions maps issue numbers to the live session started for each, in the current project.
func (s *Sessions) IssueSessions() map[int]string {
	out := map[int]string{}
	for _, v := range s.List() {
		if v.Issue > 0 {
			out[v.Issue] = v.ID
		}
	}
	return out
}

// Create starts the agent in the current project. A non-empty prefill is typed
// into its prompt, without Enter, once the agent is ready.
func (s *Sessions) Create(title, prefill string, issue int) (Session, error) {
	s.createMu.Lock()
	defer s.createMu.Unlock()

	s.mu.Lock()
	ctx, proj := s.ctx, s.project
	var used []int
	for _, in := range s.info {
		if in.Name.Project == proj.Key() {
			used = append(used, in.Name.N)
		}
	}
	name := session.Name{Project: proj.Key(), N: session.NextN(used)}
	var ready chan struct{}
	if prefill != "" {
		ready = make(chan struct{})
		s.ready[name.String()] = ready
	}
	s.mu.Unlock()

	title = cmp.Or(strings.TrimSpace(title), defaultTitle)
	env := []string{
		"AGENTOS_SESSION=" + name.String(),
		"AGENTOS_SOCKET=" + bus.SocketPath(s.stateDir),
		guard.Env + "=" + guard.Encode(proj.GuardPatterns()),
	}
	tctx, cancel := context.WithTimeout(ctx, tmuxTimeout)
	defer cancel()
	s.ev.Purge(name.String())
	err := s.start(tctx, name, title, proj.Dir, env, s.commandFor(name, proj), issue)
	if err != nil {
		s.mu.Lock()
		delete(s.ready, name.String())
		s.mu.Unlock()
		return Session{}, err
	}

	s.mu.Lock()
	in := term.Info{Name: name, Title: title, Path: proj.Dir, Created: time.Now()}
	if issue > 0 {
		in.Issue = strconv.Itoa(issue)
	}
	if !slices.ContainsFunc(s.info, func(i term.Info) bool { return i.Name == name }) {
		s.info = append(s.info, in)
	}
	s.sync()
	var view Session
	for _, v := range s.list() {
		if v.ID == name.String() {
			view = v
		}
	}
	s.mu.Unlock()

	if ready != nil {
		go s.typeWhenReady(name, ready, prefill)
	}
	return view, nil
}

func (s *Sessions) start(ctx context.Context, name session.Name, title, dir string, env, argv []string, issue int) error {
	// A number is reused, so a leftover file must not leak its old state into the new session.
	if err := bus.RemoveState(s.stateDir, name.String()); err != nil {
		return err
	}
	if err := s.tmux.NewSession(ctx, name, title, dir, env, argv, startCols, startRows); err != nil {
		return err
	}
	if issue > 0 {
		return s.tmux.SetIssue(ctx, name, issue)
	}
	return nil
}

func (s *Sessions) typeWhenReady(name session.Name, ready <-chan struct{}, text string) {
	s.mu.Lock()
	ctx := s.ctx
	wait := s.prefillFor
	s.mu.Unlock()
	timeout := time.NewTimer(wait)
	defer timeout.Stop()
	select {
	case <-ready:
		select {
		case <-time.After(promptSettles):
		case <-ctx.Done():
			return
		}
	case <-timeout.C:
	case <-ctx.Done():
		return
	}
	s.mu.Lock()
	delete(s.ready, name.String())
	s.mu.Unlock()
	tctx, cancel := context.WithTimeout(ctx, tmuxTimeout)
	defer cancel()
	_ = s.tmux.Type(tctx, name, text)
}

func (s *Sessions) Kill(id string) error {
	name, err := session.ParseName(id)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(s.context(), tmuxTimeout)
	defer cancel()
	if err := s.tmux.Kill(ctx, name); err != nil {
		return err
	}
	if err := bus.RemoveState(s.stateDir, id); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.info = slices.DeleteFunc(s.info, func(i term.Info) bool { return i.Name == name })
	s.sync()
	return nil
}

func (s *Sessions) Rename(id, title string) error {
	name, err := session.ParseName(id)
	if err != nil {
		return err
	}
	title = cmp.Or(strings.TrimSpace(title), defaultTitle)
	ctx, cancel := context.WithTimeout(s.context(), tmuxTimeout)
	defer cancel()
	if err := s.tmux.Rename(ctx, name, title); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.info {
		if s.info[i].Name == name {
			s.info[i].Title = title
		}
	}
	s.sync()
	return nil
}

func (s *Sessions) context() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ctx
}

// changed re-checks what the front end shows after something outside the session list moved.
func (s *Sessions) changed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sync()
}

// issueTargets are the sessions that work on an issue, in any project we know.
func (s *Sessions) issueTargets() []target {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []target
	for _, in := range s.info {
		if t, ok := s.targetOf(in); ok {
			out = append(out, t)
		}
	}
	return out
}

func (s *Sessions) target(id string) (target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, in := range s.info {
		if in.Name.String() == id {
			if t, ok := s.targetOf(in); ok {
				return t, nil
			}
			return target{}, fmt.Errorf("session %s has no issue to clean up after", id)
		}
	}
	return target{}, fmt.Errorf("no session %s", id)
}

// targetOf needs mu.
func (s *Sessions) targetOf(in term.Info) (target, bool) {
	issue, _ := strconv.Atoi(in.Issue)
	if issue == 0 {
		return target{}, false
	}
	for _, p := range s.projects() {
		if p.Key() == in.Name.Project {
			return target{id: in.Name.String(), title: cmp.Or(in.Title, defaultTitle), issue: issue, proj: p}, true
		}
	}
	return target{}, false
}

// TypeInto puts text into the session's prompt without pressing Enter.
func (s *Sessions) TypeInto(id, text string) error {
	name, err := session.ParseName(id)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(s.context(), tmuxTimeout)
	defer cancel()
	return s.tmux.Type(ctx, name, text)
}

// Has says whether a session of that id exists, in any project.
func (s *Sessions) Has(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.ContainsFunc(s.info, func(in term.Info) bool { return in.Name.String() == id })
}

// commandFor is the agent's command line. Claude learns about the browser when the project allows it and one exists.
func (s *Sessions) commandFor(name session.Name, proj project.Project) []string {
	argv := s.agent.Command(name.String())
	if _, ok := s.agent.(agent.Claude); ok && proj.BrowserOn() && s.browsers.Available() {
		argv = append(argv, "--append-system-prompt", agent.BrowserPrompt)
	}
	return argv
}
