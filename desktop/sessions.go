package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/session"
	"github.com/nednella/agentos/internal/term"
)

const (
	pollEvery     = 2 * time.Second
	tmuxTimeout   = 5 * time.Second
	defaultTitle  = "agent"
	historyCap    = 200
	endedKept     = 7 * 24 * time.Hour
	endedSettle   = 15 * time.Second // a state file this fresh may belong to a session tmux has not listed yet
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

	EndedAt int64 `json:"endedAt"` // unix ms; 0 while the session runs
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
	info       []term.Info
	known      map[string]term.Info     // the last info of every session seen, for when it ends
	ended      map[string]*endedSession // sessions whose tmux session is gone, until dismissed
	dismissed  map[string]bool          // ended sessions the user dismissed: the poll must not bring them back
	records    map[string]session.Record
	seen       map[string]session.State
	history    map[string][]Change
	loaded     bool // the first refresh is in, so later changes may raise attention
	lastSent   string
	lastProjs  string
	lastIssues string
	life       *Lifecycle
	repoOf     func(dir string) string
	onIssues   func()
	ready      map[string]chan struct{} // closed when the agent's SessionStart arrives
	prefillFor time.Duration
}

func newSessions(tmux *term.Tmux, ag agent.Agent, stateDir string, registry *Registry, current project.Project, emit func(string, any)) *Sessions {
	return &Sessions{
		tmux: tmux, agent: ag, stateDir: stateDir, emit: emit,
		project: current, registry: registry,
		repoOf: func(string) string { return "" }, onIssues: func() {},
		known: map[string]term.Info{}, ended: map[string]*endedSession{}, dismissed: map[string]bool{},
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

// Refresh reads tmux and the state files. A state file with no tmux session
// behind it is a session that ended.
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
	s.mu.Lock()
	defer s.mu.Unlock()
	s.info = infos
	for name, rec := range records {
		if slices.ContainsFunc(infos, func(i term.Info) bool { return i.Name.String() == name }) {
			s.take(rec)
		} else {
			s.adoptEnded(name, rec)
		}
	}
	s.expireEnded(time.Now())
	s.loaded = true
	s.sync()
}

// endedSession is a session that is gone but whose row stays until dismissed.
type endedSession struct {
	info term.Info
	rec  session.Record
	at   time.Time
}

// persist saves what the row needs to outlive the app. It needs mu.
func (e *endedSession) record() session.Record {
	rec := e.rec
	rec.State = session.Ended
	rec.Title, rec.Path, rec.EndedAt = e.info.Title, e.info.Path, e.at.UnixMilli()
	rec.Issue, _ = strconv.Atoi(e.info.Issue)
	if !e.info.Created.IsZero() {
		rec.Created = e.info.Created.UnixMilli()
	}
	return rec
}

// adoptEnded takes in a state file whose tmux session is gone. It needs mu.
func (s *Sessions) adoptEnded(name string, rec session.Record) {
	n, err := session.ParseName(name)
	if err != nil || s.dismissed[name] {
		return
	}
	if e, ok := s.ended[name]; ok {
		if rec.Title == "" && e.info.Title != "" { // a late hook overwrote the file
			_ = bus.WriteState(s.stateDir, e.record())
		}
		return
	}
	at := time.UnixMilli(rec.EndedAt)
	if rec.EndedAt == 0 {
		at = rec.At
	}
	if time.Since(rec.At) < endedSettle && rec.EndedAt == 0 {
		return
	}
	if time.Since(at) > endedKept {
		_ = bus.RemoveState(s.stateDir, name)
		return
	}
	info := term.Info{Name: n, Title: rec.Title, Path: rec.Path}
	if rec.Issue > 0 {
		info.Issue = strconv.Itoa(rec.Issue)
	}
	if rec.Created > 0 {
		info.Created = time.UnixMilli(rec.Created)
	}
	e := &endedSession{info: info, rec: rec, at: at}
	s.ended[name] = e
	if rec.EndedAt == 0 || rec.State != session.Ended {
		_ = bus.WriteState(s.stateDir, e.record())
	}
	if h := s.history[name]; len(h) == 0 || h[len(h)-1].State != session.Ended {
		s.history[name] = append(h, Change{State: session.Ended, At: at.UnixMilli()})
	}
}

// endSession turns a session that is gone into a row that stays, and saves it. It needs mu.
func (s *Sessions) endSession(id string, at time.Time) {
	if _, ok := s.ended[id]; ok || s.dismissed[id] {
		return
	}
	n, err := session.ParseName(id)
	if err != nil {
		return
	}
	info, ok := s.known[id]
	if !ok {
		info = term.Info{Name: n}
	}
	rec := s.records[id]
	if rec.Session == "" {
		rec = session.Record{Session: id, At: at}
	}
	if rec.State == session.Ended && !rec.At.IsZero() {
		at = rec.At
	}
	e := &endedSession{info: info, rec: rec, at: at}
	s.ended[id] = e
	if err := bus.WriteState(s.stateDir, e.record()); err != nil {
		fmt.Fprintf(os.Stderr, "agentos: saving an ended session: %v\n", err)
	}
	if h := s.history[id]; len(h) == 0 || h[len(h)-1].State != session.Ended {
		s.history[id] = append(h, Change{State: session.Ended, At: at.UnixMilli()})
	}
}

// expireEnded drops ended sessions that stayed a week. It needs mu.
func (s *Sessions) expireEnded(now time.Time) {
	for id, e := range s.ended {
		if now.Sub(e.at) > endedKept {
			s.drop(id)
		}
	}
}

// drop forgets an ended session for good. It needs mu.
func (s *Sessions) drop(id string) {
	delete(s.ended, id)
	delete(s.history, id)
	delete(s.known, id)
	s.dismissed[id] = true
	_ = bus.RemoveState(s.stateDir, id)
}

// Dismiss removes an ended session's row.
func (s *Sessions) Dismiss(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.ended[id]; !ok {
		return fmt.Errorf("session %s has not ended", id)
	}
	s.drop(id)
	s.sync()
	return nil
}

func (s *Sessions) observe(rec session.Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.take(rec)
	s.sync()
}

// take folds a record in. It needs mu.
func (s *Sessions) take(rec session.Record) {
	if _, ended := s.ended[rec.Session]; ended || s.dismissed[rec.Session] {
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
	var attention [][2]string // session id and the attention state
	live := map[string]bool{}
	for _, in := range s.info {
		id := in.Name.String()
		live[id] = true
		s.known[id] = in
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
		rec := s.records[id]
		replied := known && prev == session.Working && state == session.Idle && rec.Event == "Stop"
		if s.loaded && in.Name.Project == key {
			switch {
			case state == session.Waiting:
				attention = append(attention, [2]string{id, "waiting"})
			case replied:
				attention = append(attention, [2]string{id, "replied"})
			}
		}
	}
	for id := range s.seen {
		if !live[id] {
			s.life.forget(id)
			s.endSession(id, time.Now())
			delete(s.seen, id)
			delete(s.records, id)
		}
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
		s.emit("attention", map[string]string{"id": a[0], "state": a[1]})
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
		if v.Issue > 0 && v.State != session.Ended {
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
			state := session.Idle
			rec, ok := s.records[in.Name.String()]
			if ok {
				state = rec.State
			}
			switch {
			case state == session.Ended:
				continue
			case state == session.Waiting, state == session.Idle && ok && rec.Event != "SessionStart":
				v.NeedsYou++
			case state == session.Working:
				v.Working++
			}
			v.Sessions++
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
	for _, e := range s.ended {
		if e.info.Name.Project == key {
			byName[e.info.Name] = e.info
			rail = append(rail, session.Session{Name: e.info.Name, State: session.Ended, At: e.at})
		}
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
			History: slices.Clone(s.history[id]),
		}
		if !ss.At.IsZero() {
			view.LastEventAt = ss.At.UnixMilli()
		}
		if ss.State == session.Ended {
			view.EndedAt = ss.At.UnixMilli()
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
		if v.Issue > 0 && v.State != session.Ended {
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
	for _, e := range s.ended { // an ended session keeps its number until it is dismissed
		if e.info.Name.Project == proj.Key() {
			used = append(used, e.info.Name.N)
		}
	}
	name := session.Name{Project: proj.Key(), N: session.NextN(used)}
	delete(s.dismissed, name.String())
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
	}
	tctx, cancel := context.WithTimeout(ctx, tmuxTimeout)
	defer cancel()
	err := s.start(tctx, name, title, proj.Dir, env, s.agent.Command(name.String()), issue)
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
	s.mu.Lock()
	_, ended := s.ended[id]
	s.mu.Unlock()
	if ended {
		return fmt.Errorf("session %s has ended: dismiss it", id)
	}
	ctx, cancel := context.WithTimeout(s.context(), tmuxTimeout)
	defer cancel()
	if err := s.tmux.Kill(ctx, name); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	// The session is gone: sync turns it into an ended row.
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

// projectKeyOf is the project key of a session id.
func projectKeyOf(id string) string {
	name, err := session.ParseName(id)
	if err != nil {
		return "project"
	}
	return name.Project
}
