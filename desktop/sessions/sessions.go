// Package sessions tracks the agents in the private tmux server: their state, their pull requests and their clean-up.
package sessions

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

	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/desktop/internal/scoped"
	"github.com/nednella/agentos/desktop/internal/warn"
	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/prompts"
	"github.com/nednella/agentos/internal/session"
	"github.com/nednella/agentos/internal/term"
	"github.com/nednella/agentos/internal/util"
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

var errNotSession = errors.New("the shell is not a session")

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
	Model       string        `json:"model"`  // the model the agent was started with, "" when unknown
	Effort      string        `json:"effort"` // the effort the agent was started with, "" when unknown
	History     []Change      `json:"history"`
	Evidence    int           `json:"evidence"`
	Browser     bool          `json:"browser"`

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
	Key      string `json:"key"` // the project's key, as in the project field of events and in session ids
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
	model    project.Model // the config's default, before a project, lane or label changes it
	stateDir string
	emit     func(event string, payload any)
	warn     *warn.Warnings

	createMu sync.Mutex // one creation, or one refresh of the list from tmux, at a time: a list read before a creation must not be applied after it
	shellMu  sync.Mutex // one shell check-and-create at a time: two callers at once would both find no shell and tmux refuses the second

	mu         sync.Mutex
	ctx        context.Context
	project    project.Project
	configured ProjectList
	tally      Tally
	closeTerm  func(id string)
	evidence   Evidence
	browsers   Browsers
	awake      Awake
	keepAwake  func(projectKey string) bool
	info       []term.Info
	shells     []string                 // projects that have a shell session
	known      map[string]term.Info     // the last info of every session seen, for when it ends
	ended      map[string]*endedSession // sessions whose tmux session is gone, until dismissed
	dismissed  map[string]bool          // ended sessions the user dismissed: the poll must not bring them back
	records    map[string]session.Record
	cwds       map[string]string // the working directory each session's last hook reported
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
	wakes      map[string]string        // prompts for sessions that were waiting on the user when their PR needed them
	prefillFor time.Duration
}

// ProjectList is the configured projects.
type ProjectList interface {
	List() []project.Project
	Remove(name string) (bool, error)
}

// Tally records when a session waits on the user.
type Tally interface {
	Opened(id, title string, issue int, model string, rec session.Record, at time.Time)
	Closed(id string, at time.Time) bool
}

// Evidence is what the sessions need of the evidence store.
type Evidence interface {
	Count(id string) int
	Purge(id string) int
}

// Browsers is what the sessions need of the browsers.
type Browsers interface {
	Available() bool
	Has(id string) bool
	Close(ctx context.Context, id string)
}

// Awake holds off idle sleep.
type Awake interface {
	Hold(on bool)
}

// Options is what a Sessions needs from outside.
type Options struct {
	Tmux          *term.Tmux
	Agent         agent.Agent
	Model         project.Model
	StateDir      string
	LocalDir      string // pull-request tracking and the clean-up log: never synced
	Projects      ProjectList
	Current       project.Project
	Emit          func(event string, payload any)
	Run           run.Runner
	Stream        run.Streamer // long-running commands: gh webhook forward
	Tally         Tally
	CloseTerminal func(id string)
	Evidence      Evidence
	Browsers      Browsers
	Awake         Awake
	KeepAwake     func(projectKey string) bool
}

// New tracks the sessions of o.Tmux, and follows their pull requests.
func New(o Options) *Sessions {
	s := newSessions(o)
	s.life = newLifecycle(o.Run, o.Stream, o.LocalDir, o.StateDir, s, o.Emit)
	return s
}

// Stop ends the clean-ups that are running and waits for them.
func (s *Sessions) Stop() { s.life.stopCleanups() }

// Hook sets what the sessions ask of the issues: the repo of a folder, a nudge when the
// sessions tied to issues change, and what the pull request watch needs.
func (s *Sessions) Hook(repoOf func(dir string) string, onIssues func(), repos Repos) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repoOf, s.onIssues = repoOf, onIssues
	s.life.repos = repos
}

// Touch tells the front end that something shown in the session list changed.
func (s *Sessions) Touch() { s.changed() }

func newSessions(o Options) *Sessions {
	return &Sessions{
		tmux: o.Tmux, agent: o.Agent, model: o.Model, stateDir: o.StateDir, emit: o.Emit, warn: warn.New(o.Emit),
		project: o.Current, configured: o.Projects, tally: o.Tally, closeTerm: o.CloseTerminal, evidence: o.Evidence, browsers: o.Browsers, awake: o.Awake, keepAwake: o.KeepAwake,
		repoOf: func(string) string { return "" }, onIssues: func() {},
		known: map[string]term.Info{}, ended: map[string]*endedSession{}, dismissed: map[string]bool{},
		records:    map[string]session.Record{},
		cwds:       map[string]string{},
		seen:       map[string]session.State{},
		history:    map[string][]Change{},
		ready:      map[string]chan struct{}{},
		wakes:      map[string]string{},
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
// behind it is a session that ended. When tmux cannot be listed the state stays as it was:
// a failed listing says nothing about which sessions are gone.
func (s *Sessions) Refresh() {
	s.createMu.Lock()
	defer s.createMu.Unlock()
	s.mu.Lock()
	parent := s.ctx
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(parent, tmuxTimeout)
	defer cancel()
	infos, shells, err := s.tmux.ListAll(ctx)
	if err != nil {
		if parent.Err() == nil {
			s.warn.Report("tmux", util.FirstLine(err.Error()))
		}
		return
	}
	s.warn.Clear("tmux")
	records := bus.ReadAll(s.stateDir)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.info = infos
	s.shells = shells
	for _, in := range infos {
		if name := in.Name.String(); s.ended[name] != nil {
			s.revive(name)
			delete(records, name)
		}
	}
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

// revive takes back a session that tmux lists although it was marked ended. The saved
// record says ended, so it goes: the next hook writes a true one. It needs mu.
func (s *Sessions) revive(name string) {
	delete(s.ended, name)
	_ = bus.RemoveState(s.stateDir, name)
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
	rec.Model, rec.Effort = e.info.Model, e.info.Effort
	if !e.info.Created.IsZero() {
		rec.Created = e.info.Created.UnixMilli()
	}
	return rec
}

// adoptEnded takes in a state file whose tmux session is gone. It needs mu.
func (s *Sessions) adoptEnded(name string, rec session.Record) {
	n, err := session.ParseName(name)
	if err != nil {
		return
	}
	if s.dismissed[name] {
		_ = bus.RemoveState(s.stateDir, name) // a hook that ran late wrote it again
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
	info := term.Info{Name: n, Title: rec.Title, Path: rec.Path, Model: rec.Model, Effort: rec.Effort}
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
	s.life.forget(id)
	_ = bus.RemoveState(s.stateDir, id)
	go s.evidence.Purge(id)
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
	s.followBranch(rec)
	if ch, ok := s.ready[rec.Session]; ok && rec.Event == "SessionStart" {
		close(ch)
		delete(s.ready, rec.Session)
	}
}

// followBranch has the lifecycle look at the branch of a session, when the agent moved or
// finished a turn. It needs mu.
func (s *Sessions) followBranch(rec session.Record) {
	moved := rec.Cwd != "" && (rec.Cwd != s.cwds[rec.Session] || rec.Event == "Stop")
	if !moved {
		return
	}
	s.cwds[rec.Session] = rec.Cwd
	for _, in := range s.info {
		if in.Name.String() != rec.Session {
			continue
		}
		if proj, ok := s.projectOf(in); ok {
			go s.life.learn(s.ctx, rec.Session, proj, rec.Cwd)
		}
	}
}

// sync updates history and attention from what is known, and tells the front
// end if the list changed. It needs mu.
func (s *Sessions) sync() {
	key := s.project.Key()
	var attention [][2]string // session id and the attention state
	waitsChanged := false
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
		opened := s.loaded && state == session.Idle && s.life.opened(id)
		if opened {
			attention = append(attention, [2]string{id, "opened"})
		}
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
			case replied && !opened:
				attention = append(attention, [2]string{id, "replied"})
			}
		}
		if s.loaded {
			ended := s.tally.Closed(id, at)
			opens := state == session.Waiting || replied
			if opens {
				issue, _ := strconv.Atoi(in.Issue)
				s.tally.Opened(id, cmp.Or(in.Title, defaultTitle), issue, in.Model, rec, at)
			}
			if ended || opens {
				waitsChanged = true
			}
		}
	}
	for id := range s.seen {
		if !live[id] {
			if s.tally.Closed(id, time.Now()) {
				waitsChanged = true
			}
			go s.closeBrowser(id)
			s.endSession(id, time.Now())
			delete(s.seen, id)
			delete(s.records, id)
		}
	}
	if waitsChanged {
		s.emit("stats", nil)
	}
	s.deliverWakes()
	s.holdAwake()

	s.announceProjects()
	s.announceIssues()
	list := s.list()
	b, err := json.Marshal(list)
	if err != nil || string(b) == s.lastSent {
		return
	}
	s.lastSent = string(b)
	s.emit("sessions", scoped.Of(key, list))
	for _, a := range attention {
		s.emit("attention", map[string]string{"id": a[0], "state": a[1]})
	}
}

// holdAwake keeps the Mac awake while a session of a project that keeps awake is working. It needs mu.
func (s *Sessions) holdAwake() {
	if s.awake == nil {
		return
	}
	working := slices.ContainsFunc(s.info, func(in term.Info) bool {
		return s.records[in.Name.String()].State == session.Working && s.keepAwake(in.Name.Project)
	})
	s.awake.Hold(working)
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
		v := Project{Key: p.Key(), Name: p.Name, Dir: p.Dir, Repo: s.repoOf(p.Dir)}
		for _, in := range s.info {
			if in.Name.Project != p.Key() {
				continue
			}
			state := session.Idle
			if rec, ok := s.records[in.Name.String()]; ok {
				state = rec.State
			}
			switch state {
			case session.Ended:
				continue
			case session.Waiting:
				v.NeedsYou++
			case session.Working:
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
			CreatedAt: in.Created.UnixMilli(), Issue: issue, Model: in.Model, Effort: in.Effort, Evidence: s.evidence.Count(id), Browser: s.browsers.Has(id),
			History: slices.Clone(s.history[id]),
		}
		if !ss.At.IsZero() {
			view.LastEventAt = ss.At.UnixMilli()
		}
		if ss.State == session.Ended {
			view.EndedAt = ss.At.UnixMilli()
		}
		f := s.life.fields(id)
		view.Branch, view.Worktree, view.PR = f.branch, f.worktree, f.pr
		view.PRAttention, view.Cleanup, view.CleanupReason = f.attention, f.cleanup, f.reason
		if view.Branch == "" && issue > 0 {
			view.Branch = s.project.BranchFor(issue)
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
	out := s.configured.List()
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

// Forget removes a configured project and ends its sessions, moving to another one if it was current.
func (s *Sessions) Forget(name string) (project.Project, error) {
	if live := s.liveIssueSessions(project.Project{Name: name}.Key()); len(live) > 0 {
		return project.Project{}, fmt.Errorf("cannot remove %q: it has live issue sessions (%s), which need its branch pattern and clean-up command; end them first", name, strings.Join(live, ", "))
	}
	removed, err := s.configured.Remove(name)
	if err != nil {
		return project.Project{}, err
	}
	s.mu.Lock()
	current := s.project.Name == name
	s.mu.Unlock()
	// The current project can be missing from the config (the folder the app started in); removing
	// it still means moving away.
	if !removed && !current {
		return project.Project{}, fmt.Errorf("%q is not in the config", name)
	}
	if err := s.killProject(project.Project{Name: name}.Key()); err != nil {
		return project.Project{}, err
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

// killProject ends every running session of the project. It takes mu.
func (s *Sessions) killProject(key string) error {
	s.mu.Lock()
	var ids []string
	for _, in := range s.info {
		if in.Name.Project == key {
			ids = append(ids, in.Name.String())
		}
	}
	s.mu.Unlock()
	var errs []error
	for _, id := range ids {
		errs = append(errs, s.Kill(id))
	}
	return errors.Join(errs...)
}

// liveIssueSessions names the running sessions of the project that work on an issue. It takes mu.
func (s *Sessions) liveIssueSessions(key string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for _, in := range s.info {
		if issue, _ := strconv.Atoi(in.Issue); issue > 0 && in.Name.Project == key {
			out = append(out, fmt.Sprintf("session %d for #%d", in.Name.N, issue))
		}
	}
	return out
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

// Create starts the agent in the current project. A non-empty text is typed
// into its prompt once the agent is ready, and sent when send is set.
func (s *Sessions) Create(title, text string, send bool, issue int) (Session, error) {
	proj := s.Current()
	return s.createIn(proj, title, text, send, issue, proj.Pick(s.model, "", nil), "")
}

// CreateIssue starts the agent for an issue in the lane with the issue's labels, and sends text to it
// once it is ready. The lane and the labels pick the model.
func (s *Sessions) CreateIssue(title, text string, issue int, lane string, labels []string) (Session, error) {
	proj := s.Current()
	return s.createIn(proj, title, text, true, issue, proj.Pick(s.model, lane, labels), "")
}

// createIn is Create in the project given, which need not be the current one, with the model given.
// A conversation continues that of an earlier session.
func (s *Sessions) createIn(proj project.Project, title, text string, send bool, issue int, model project.Model, conversation string) (Session, error) {
	s.createMu.Lock()
	defer s.createMu.Unlock()

	name, ctx, ready := s.reserve(proj, text)
	title = cmp.Or(strings.TrimSpace(title), defaultTitle)
	tctx, cancel := context.WithTimeout(ctx, tmuxTimeout)
	defer cancel()
	if err := s.launch(tctx, name, title, proj, issue, model, conversation); err != nil {
		s.mu.Lock()
		delete(s.ready, name.String())
		s.mu.Unlock()
		return Session{}, err
	}
	view := s.register(name, title, proj, issue, model)
	if ready != nil {
		go s.typeWhenReady(name, ready, text, send)
	}
	if issue > 0 {
		s.life.ensureWatchers(ctx)
	}
	return view, nil
}

// reserve picks the lowest free number of the project. With a prefill it also
// returns the channel that closes when the agent says it is ready. It takes mu.
func (s *Sessions) reserve(proj project.Project, prefill string) (name session.Name, ctx context.Context, ready chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ctx = s.ctx
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
	name = session.Name{Project: proj.Key(), N: session.NextN(used)}
	delete(s.dismissed, name.String())
	if prefill != "" {
		ready = make(chan struct{})
		s.ready[name.String()] = ready
	}
	return name, ctx, ready
}

// launch starts the agent of the session in tmux.
func (s *Sessions) launch(ctx context.Context, name session.Name, title string, proj project.Project, issue int, model project.Model, conversation string) error {
	env := []string{
		"AGENTOS_SESSION=" + name.String(),
		"AGENTOS_SOCKET=" + bus.SocketPath(s.stateDir),
	}
	if issue > 0 {
		env = append(env, "AGENTOS_ISSUE="+strconv.Itoa(issue))
	}
	if _, ok := s.agent.(agent.Claude); !ok {
		model = project.Model{} // another agent takes no model flags, so none was chosen
	}
	return s.start(ctx, name, title, proj.Dir, env, s.commandFor(name, proj, model, conversation), issue, model)
}

// register adds the new session to the list and returns its row. It takes mu.
func (s *Sessions) register(name session.Name, title string, proj project.Project, issue int, model project.Model) Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	in := term.Info{Name: name, Title: title, Path: proj.Dir, Created: time.Now()}
	if _, ok := s.agent.(agent.Claude); ok {
		in.Model, in.Effort = model.Model, model.Effort
	}
	if issue > 0 {
		in.Issue = strconv.Itoa(issue)
	}
	if !slices.ContainsFunc(s.info, func(i term.Info) bool { return i.Name == name }) {
		s.info = append(s.info, in)
	}
	s.sync()
	for _, v := range s.list() {
		if v.ID == name.String() {
			return v
		}
	}
	return Session{}
}

func (s *Sessions) start(ctx context.Context, name session.Name, title, dir string, env, argv []string, issue int, model project.Model) error {
	// A number is reused, so a leftover file must not leak its old state into the new session.
	if err := bus.RemoveState(s.stateDir, name.String()); err != nil {
		return err
	}
	if err := s.tmux.NewSession(ctx, name, title, dir, env, argv, startCols, startRows); err != nil {
		return err
	}
	tag := func(err error) error {
		if err == nil {
			return nil
		}
		// Without its tags the session would be a stray: untracked, and tmux would still run it.
		killCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), tmuxTimeout)
		defer cancel()
		_ = s.tmux.Kill(killCtx, name)
		return err
	}
	if issue > 0 {
		if err := tag(s.tmux.SetIssue(ctx, name, issue)); err != nil {
			return err
		}
	}
	if model != (project.Model{}) {
		return tag(s.tmux.SetModel(ctx, name, model.Model, model.Effort))
	}
	return nil
}

func (s *Sessions) typeWhenReady(name session.Name, ready <-chan struct{}, text string, send bool) {
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
	s.send(ctx, name, text, send)
}

// send types text into the session's prompt and, when submit is set, presses Enter.
func (s *Sessions) send(ctx context.Context, name session.Name, text string, submit bool) {
	tctx, cancel := context.WithTimeout(ctx, tmuxTimeout)
	defer cancel()
	if err := s.tmux.Type(tctx, name, text); err != nil || !submit {
		return
	}
	// Enter right behind the text can reach the agent before the text has landed.
	select {
	case <-time.After(promptSettles):
	case <-ctx.Done():
		return
	}
	_ = s.tmux.Submit(tctx, name)
}

// wake sends the prompt to the session, once it is not waiting on the user: typed into a
// permission prompt, the text could answer it. A session that has ended gets a new one for
// its issue, with the prompt sent, and its row goes. The new one continues the old's conversation
// when the agent can.
func (s *Sessions) wake(ctx context.Context, t target, prompt string) {
	if t.ended && t.issue == 0 {
		return
	}
	if t.ended {
		if _, err := s.createIn(t.proj, t.title, prompt, true, t.issue, t.model, t.conversation); err != nil {
			fmt.Fprintf(os.Stderr, "agentos: starting a session for #%d: %v\n", t.issue, err)
			return
		}
		_ = s.Dismiss(t.id)
		return
	}
	name, err := session.ParseName(t.id)
	if err != nil {
		return
	}
	s.mu.Lock()
	if rec := s.records[t.id]; rec.State == session.Waiting {
		s.wakes[t.id] = prompt
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	s.send(ctx, name, prompt, true)
}

// deliverWakes sends the prompts held back while their sessions waited on the user. It needs mu.
func (s *Sessions) deliverWakes() {
	for id, prompt := range s.wakes {
		_, live := s.seen[id]
		switch {
		case s.dismissed[id]:
			delete(s.wakes, id)
		case s.ended[id] != nil:
			delete(s.wakes, id)
			if t, err := s.targetOrErr(s.ended[id].info, true); err == nil {
				go s.wake(s.ctx, t, prompt)
			}
		case live && s.records[id].State != session.Waiting:
			delete(s.wakes, id)
			go s.send(s.ctx, s.known[id].Name, prompt, true)
		}
	}
}

func (s *Sessions) Kill(id string) error {
	name, err := session.ParseName(id)
	if err != nil {
		return err
	}
	if name.IsShell() {
		return errNotSession
	}
	s.mu.Lock()
	_, ended := s.ended[id]
	s.mu.Unlock()
	if ended {
		return fmt.Errorf("session %s has ended: dismiss it", id)
	}
	s.closeTerm(id)
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
	if name.IsShell() {
		return errNotSession
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

// workTargets are the sessions that have a branch to follow, live or ended, in any project we know.
func (s *Sessions) workTargets() []target {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []target
	for _, in := range s.info {
		if t, ok := s.targetOf(in); ok {
			out = append(out, t)
		}
	}
	for _, e := range s.ended {
		if t, ok := s.targetOf(e.info); ok {
			t.ended = true
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
			return s.targetOrErr(in, false)
		}
	}
	if e, ok := s.ended[id]; ok {
		return s.targetOrErr(e.info, true)
	}
	return target{}, fmt.Errorf("no session %s", id)
}

// targetOrErr needs mu.
func (s *Sessions) targetOrErr(in term.Info, ended bool) (target, error) {
	t, ok := s.targetOf(in)
	if !ok {
		return target{}, fmt.Errorf("session %s has no issue or branch to clean up after", in.Name)
	}
	t.ended = ended
	return t, nil
}

// targetOf needs mu.
func (s *Sessions) targetOf(in term.Info) (target, bool) {
	proj, ok := s.projectOf(in)
	if !ok {
		return target{}, false
	}
	issue, _ := strconv.Atoi(in.Issue)
	id := in.Name.String()
	branches := s.life.branchesOf(proj, id)
	if fallback := proj.BranchFor(issue); len(branches) == 0 && issue > 0 && fallback != "" {
		branches = []string{fallback}
	}
	if len(branches) == 0 {
		return target{}, false
	}
	return target{id: id, title: cmp.Or(in.Title, defaultTitle), issue: issue, branches: branches, proj: proj, model: project.Model{Model: in.Model, Effort: in.Effort}, conversation: s.conversationOf(id)}, true
}

// conversationOf is the agent's conversation id of a session, running or ended. It needs mu.
func (s *Sessions) conversationOf(id string) string {
	if e, ok := s.ended[id]; ok {
		return e.rec.Conversation
	}
	return s.records[id].Conversation
}

// projectOf needs mu.
func (s *Sessions) projectOf(in term.Info) (project.Project, bool) {
	for _, p := range s.projects() {
		if p.Key() == in.Name.Project {
			return p, true
		}
	}
	return project.Project{}, false
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

// Run follows tmux, the hook bus and the pull requests until ctx ends.
func (s *Sessions) Run(ctx context.Context) error {
	if err := s.Start(ctx); err != nil {
		return err
	}
	go s.life.Run(ctx)
	return nil
}

// SetPrefillWait sets how long to wait for an agent to be ready before typing in the prefill.
func (s *Sessions) SetPrefillWait(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prefillFor = d
}

// Has says whether a session of that id is running, in any project.
// Track records branch as the one the session works on, and watches its worktree and pull request.
func (s *Sessions) Track(id, branch string) error {
	s.mu.Lock()
	var proj project.Project
	var found bool
	for _, in := range s.info {
		if in.Name.String() == id {
			proj, found = s.projectOf(in)
			break
		}
	}
	s.mu.Unlock()
	if !found {
		return fmt.Errorf("session %q is not running", id)
	}
	s.life.track(s.ctx, id, proj, branch)
	return nil
}

func (s *Sessions) Has(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.ContainsFunc(s.info, func(in term.Info) bool { return in.Name.String() == id })
}

func (s *Sessions) closeBrowser(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), tmuxTimeout)
	defer cancel()
	s.browsers.Close(ctx, id)
}

// ShellID is the current project's shell session, or "" until it is opened.
func (s *Sessions) ShellID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := s.project.Key()
	if slices.Contains(s.shells, key) {
		return session.Name{Project: key}.String()
	}
	return ""
}

// OpenShell makes sure the current project has a shell session, a login shell
// in the project folder, and returns its id. The shell reports nothing: it is
// not an agent, so it gets no hooks and no session number.
func (s *Sessions) OpenShell() (string, error) {
	s.shellMu.Lock()
	defer s.shellMu.Unlock()
	s.mu.Lock()
	ctx, proj := s.ctx, s.project
	s.mu.Unlock()
	name := session.Name{Project: proj.Key()}
	tctx, cancel := context.WithTimeout(ctx, tmuxTimeout)
	defer cancel()
	if !s.tmux.Has(tctx, name) {
		login := cmp.Or(os.Getenv("SHELL"), "/bin/zsh")
		env := []string{
			"AGENTOS_PROJECT=" + proj.Key(),
			"AGENTOS_SOCKET=" + bus.SocketPath(s.stateDir),
		}
		if err := s.tmux.NewSession(tctx, name, "shell", proj.Dir, env, []string{login, "-l"}, startCols, startRows); err != nil {
			return "", err
		}
	}
	s.mu.Lock()
	if !slices.Contains(s.shells, proj.Key()) {
		s.shells = append(s.shells, proj.Key())
	}
	s.mu.Unlock()
	return name.String(), nil
}

// commandFor is the agent's command line. Claude learns about the browser when the project allows it and one exists.
func (s *Sessions) commandFor(name session.Name, proj project.Project, model project.Model, conversation string) []string {
	argv := s.agent.Command(name.String(), agent.Launch{Model: model.Model, Effort: model.Effort, Resume: conversation})
	if _, ok := s.agent.(agent.Claude); ok && proj.BrowserOn() && s.browsers.Available() {
		argv = append(argv, "--append-system-prompt", prompts.BrowserSession())
	}
	return argv
}
