package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/term"
	"github.com/nednella/agentos/internal/version"
)

const defaultTmuxSocket = "agentos"

var issueURL = regexp.MustCompile(`https://[^\s/]+/[^\s/]+/[^\s/]+/issues/(\d+)`)

type Snapshot struct {
	Project  Project   `json:"project"`
	Projects []Project `json:"projects"`
	Sessions []Session `json:"sessions"`
	Notes    []Note    `json:"notes"`
	Version  string    `json:"version"`
}

// host is what the window provides. The logic never touches Wails directly.
type host struct {
	emit      func(event string, payload any)
	clipboard func(text string)
	openURL   func(url string)
	pickDir   func() (string, error) // "" when the user cancels
}

type config struct {
	registry *Registry
	project  project.Project
	stateDir string
	dataDir  string // notes, stats and digests: may be a synced folder
	localDir string // evidence, PR tracking, clean-up log and browser profile: never synced
	tmux     *term.Tmux
	agent    agent.Agent
}

// loadConfig reads the same environment and config file as the terminal UI. The
// project is the one holding AGENTOS_DIR or the current folder. An app started
// from Finder (folder "/") gets the last project used, else the first configured
// one, else the home folder.
func loadConfig() (config, error) {
	path := os.Getenv("AGENTOS_CONFIG")
	home, err := os.UserHomeDir()
	if err != nil {
		return config{}, fmt.Errorf("finding home dir: %w", err)
	}
	if path == "" {
		path = filepath.Join(home, ".config", "agentos", "config.yaml")
	}
	cfg, err := project.Load(path)
	if err != nil {
		return config{}, fmt.Errorf("loading config: %w", err)
	}
	stateDir, err := bus.DefaultDir()
	if err != nil {
		return config{}, err
	}
	localDir := filepath.Join(home, ".local", "share", "agentos")
	dataDir := cmp.Or(cfg.DataDir, localDir)
	if override := os.Getenv("AGENTOS_DATA_DIR"); override != "" {
		localDir, dataDir = override, override
	}

	dir := os.Getenv("AGENTOS_DIR")
	if dir == "" {
		if dir, err = os.Getwd(); err != nil {
			return config{}, fmt.Errorf("finding the current folder: %w", err)
		}
	}
	current := project.Project{}
	if dir == "/" {
		last := readLastProject(stateDir)
		switch i := slices.IndexFunc(cfg.Projects, func(p project.Project) bool { return p.Name == last }); {
		case i >= 0:
			current = cfg.Projects[i]
		case len(cfg.Projects) > 0:
			current = cfg.Projects[0]
		default:
			current = cfg.Resolve(home)
		}
	} else {
		current = cfg.Resolve(dir)
	}

	socket := os.Getenv("AGENTOS_TMUX_SOCKET")
	if socket == "" {
		socket = defaultTmuxSocket
	}
	tmux, err := term.NewTmux(socket, stateDir)
	if err != nil {
		return config{}, err
	}
	return config{
		registry: &Registry{path: path, cfg: cfg}, project: current, stateDir: stateDir,
		dataDir: dataDir, localDir: localDir, tmux: tmux, agent: pickAgent(cfg.Agent),
	}, nil
}

// pickAgent builds the agent adapter. Without the agentos command on PATH the
// agents' hooks cannot call back, so Claude runs plain and sessions stay idle.
func pickAgent(name string) agent.Agent {
	exe, err := exec.LookPath("agentos")
	if err == nil {
		return agent.New(name, exe)
	}
	log.Printf("agentos: the agentos command is not on PATH, so agents will not report their state: %v", err)
	if name == "" || name == "claude" {
		return agent.Plain{Argv: []string{"claude"}}
	}
	return agent.New(name, "")
}

// App is the one struct bound to the front end. Every exported method is part of the contract.
type App struct {
	ctx      context.Context
	host     host
	stateDir string
	sessions *Sessions
	terms    *Terms
	issues   *Issues
	notes    *Notes
	life     *Lifecycle
	run      runner
	runEnv   envRunner
	media    map[string]string // folder of a /media/ URL -> the dir that holds it
	dataDir  string            // notes, stats and digests
	localDir string            // everything that must stay on this machine

	browsers *Browsers
	evidence *EvidenceStore
	digests  *Digests
	ctl      *control.Server

	digestFirst time.Duration // wait before the first automatic digest check
	digestTick  time.Duration // between checks after that
}

func newApp(c config, h host, run runner) *App {
	a := &App{
		ctx:      context.Background(),
		host:     h,
		stateDir: c.stateDir,
		sessions: newSessions(c.tmux, c.agent, c.stateDir, c.dataDir, c.registry, c.project, h.emit),
		terms:    newTerms(c.tmux, h.emit, h.clipboard),
		issues:   newIssues(run),
		notes:    &Notes{dir: c.dataDir},
		run:      run,
		runEnv:   execRunnerEnv,
		browsers: newBrowsers(c.localDir, h.emit),
		evidence: newEvidenceStore(c.localDir),
		digests:  newDigests(c.dataDir),

		digestFirst: time.Minute,
		digestTick:  time.Hour,
	}
	a.life = newLifecycle(run, c.localDir, c.stateDir, a.sessions, h.emit, a.terms.Close)
	a.sessions.life = a.life
	a.dataDir, a.localDir = c.dataDir, c.localDir
	a.media = map[string]string{mediaFolder: c.dataDir, evidenceFolder: c.localDir}
	a.sessions.browsers, a.sessions.ev = a.browsers, a.evidence
	a.sessions.onGone = a.sessionGone
	a.sessions.onDismiss = func(id string) { a.evidence.Purge(id) }
	a.browsers.urlFor = a.projectURL
	a.browsers.onTabs = a.sessions.changed
	a.life.release = a.release
	a.sessions.repoOf = a.issues.CachedRepo
	a.sessions.onIssues = a.emitIssues
	return a
}

// start begins listening for hooks; it runs until ctx ends.
func (a *App) start(ctx context.Context) error {
	a.ctx = ctx
	migrateLayout(a.dataDir, a.localDir, log.Printf)
	if err := a.sessions.Start(ctx); err != nil {
		return err
	}
	ctl, err := control.Listen(control.SocketPath(a.stateDir), a.handleControl)
	if err != nil {
		log.Printf("agentos: agents cannot use the browser and evidence commands: %v", err)
	}
	a.ctl = ctl
	go a.life.Run(ctx)
	go a.digestLoop(ctx)
	return nil
}

func (a *App) stop() {
	a.terms.CloseAll()
	a.browsers.CloseAll()
	if a.ctl != nil {
		a.ctl.Close()
	}
}

// sessionGone closes the browser tab of a session that ended. Its evidence stays until it is dismissed.
func (a *App) sessionGone(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	a.browsers.Close(ctx, id)
}

// release is the clean-up step for the session's browser tab and evidence; it names what it removed.
func (a *App) release(ctx context.Context, id string) []string {
	var removed []string
	if a.browsers.Has(id) {
		a.browsers.Close(ctx, id)
		removed = append(removed, "browser tab")
	}
	if a.evidence.Purge(id) > 0 {
		removed = append(removed, "evidence")
	}
	return removed
}

func (a *App) projectURL(id string) string {
	for _, p := range a.sessions.registry.List() {
		if p.Key() == projectKeyOf(id) {
			return p.URL
		}
	}
	return ""
}

func (a *App) Snapshot() Snapshot {
	cur := a.sessions.Current()
	repo := a.issues.Repo(a.ctx, cur.Dir)
	projects := a.sessions.Projects()
	snap := Snapshot{Projects: projects, Sessions: a.sessions.List(), Version: version.Version}
	snap.Project = Project{Name: cur.Name, Dir: cur.Dir}
	for i, p := range projects {
		if p.Dir == cur.Dir {
			projects[i].Repo = repo
			snap.Project = projects[i]
		}
	}
	var err error
	if snap.Notes, err = a.notes.List(cur.Key()); err != nil {
		log.Printf("agentos: %v", err)
		snap.Notes = []Note{}
	}
	return snap
}

func (a *App) NewSession(title, prefill string) (Session, error) {
	return a.sessions.Create(title, prefill, 0)
}

func (a *App) KillSession(id string) error {
	a.terms.Close(id)
	return a.sessions.Kill(id)
}

// DismissSession removes the row of an ended session.
func (a *App) DismissSession(id string) error { return a.sessions.Dismiss(id) }

func (a *App) RenameSession(id, title string) error { return a.sessions.Rename(id, title) }

func (a *App) SwitchProject(name string) (Snapshot, error) {
	p, err := a.sessions.SwitchProject(name)
	if err != nil {
		return Snapshot{}, err
	}
	a.remember(p)
	return a.Snapshot(), nil
}

func (a *App) remember(p project.Project) {
	if err := writeLastProject(a.stateDir, p.Name); err != nil {
		log.Printf("agentos: remembering the project: %v", err)
	}
}

// AddProject asks for a folder and makes it the current project.
func (a *App) AddProject() (Snapshot, error) {
	dir, err := a.host.pickDir()
	if err != nil {
		return Snapshot{}, fmt.Errorf("choosing a folder: %w", err)
	}
	if dir == "" {
		return a.Snapshot(), nil
	}
	return a.AddProjectDir(dir)
}

func (a *App) AddProjectDir(dir string) (Snapshot, error) {
	p, err := a.sessions.registry.Add(dir)
	if err != nil {
		return Snapshot{}, err
	}
	return a.SwitchProject(p.Name)
}

func (a *App) RemoveProject(name string) (Snapshot, error) {
	p, err := a.sessions.Forget(name)
	if err != nil {
		return Snapshot{}, err
	}
	a.remember(p)
	return a.Snapshot(), nil
}

func (a *App) Issues(refresh bool) ([]Issue, error) {
	list, err := a.issues.List(a.ctx, a.sessions.Current(), refresh)
	if err != nil {
		return nil, err
	}
	return a.withSessions(list), nil
}

func (a *App) withSessions(list []Issue) []Issue {
	live := a.sessions.IssueSessions()
	for i := range list {
		list[i].SessionID = live[list[i].Number]
	}
	return list
}

// emitIssues sends the cached issues, when there are any, with their current sessions.
func (a *App) emitIssues() {
	if list, ok := a.issues.Cached(a.sessions.Current().Dir); ok {
		a.host.emit("issues", a.withSessions(list))
	}
}

func (a *App) emitNotes() {
	notes, err := a.notes.List(a.sessions.Current().Key())
	if err != nil {
		log.Printf("agentos: %v", err)
		return
	}
	a.host.emit("notes", notes)
}

// StartIssue opens a session for the issue. An issue that already has a live
// session gets that session back instead of a second agent.
func (a *App) StartIssue(number int) (Session, error) {
	list, err := a.Issues(false)
	if err != nil {
		return Session{}, err
	}
	for _, is := range list {
		if is.Number != number {
			continue
		}
		if is.SessionID != "" {
			for _, s := range a.sessions.List() {
				if s.ID == is.SessionID {
					return s, nil
				}
			}
		}
		return a.sessions.Create(is.sessionTitle(), a.sessions.Current().IssueCommand(is.Lane, number), number)
	}
	return Session{}, fmt.Errorf("issue #%d is not open in this project", number)
}

func (a *App) AddNote(text string) (Note, error) {
	n, err := a.notes.Add(a.sessions.Current().Key(), text)
	if err == nil {
		a.emitNotes()
	}
	return n, err
}

func (a *App) UpdateNote(id, text string) (Note, error) {
	n, err := a.notes.Update(a.sessions.Current().Key(), id, text)
	if err == nil {
		a.emitNotes()
	}
	return n, err
}

func (a *App) SetNotePinned(id string, pinned bool) (Note, error) {
	n, err := a.notes.SetPinned(a.sessions.Current().Key(), id, pinned)
	if err == nil {
		a.emitNotes()
	}
	return n, err
}

func (a *App) SetNoteArchived(id string, archived bool) (Note, error) {
	n, err := a.notes.SetArchived(a.sessions.Current().Key(), id, archived)
	if err == nil {
		a.emitNotes()
	}
	return n, err
}

// SetNoteDone is the old name of SetNoteArchived.
func (a *App) SetNoteDone(id string, done bool) (Note, error) { return a.SetNoteArchived(id, done) }

func (a *App) AddNoteImage(id, data, mime string) (Note, error) {
	n, err := a.notes.AddImage(a.sessions.Current().Key(), id, data, mime)
	if err == nil {
		a.emitNotes()
	}
	return n, err
}

func (a *App) RemoveNoteImage(id, url string) (Note, error) {
	n, err := a.notes.RemoveImage(a.sessions.Current().Key(), id, url)
	if err == nil {
		a.emitNotes()
	}
	return n, err
}

func (a *App) DeleteNote(id string) error {
	err := a.notes.Delete(a.sessions.Current().Key(), id)
	if err == nil {
		a.emitNotes()
	}
	return err
}

// NoteToIssue files the note on GitHub: its first line is the title, the rest the body.
func (a *App) NoteToIssue(id string) (Note, error) {
	cur := a.sessions.Current()
	n, err := a.notes.Get(cur.Key(), id)
	if err != nil {
		return Note{}, err
	}
	if n.Issue > 0 {
		return Note{}, fmt.Errorf("this note is already issue #%d", n.Issue)
	}
	if a.issues.Repo(a.ctx, cur.Dir) == "" {
		return Note{}, errors.New("this project has no GitHub repo")
	}
	ctx, cancel := context.WithTimeout(a.ctx, ghTimeout)
	defer cancel()
	out, err := a.run(ctx, cur.Dir, "gh", "issue", "create", "--title", n.title(), "--body", issueBody(n), "--assignee", "@me")
	if err != nil {
		return Note{}, fmt.Errorf("filing the issue: %w", err)
	}
	m := issueURL.FindStringSubmatch(string(out))
	if m == nil {
		return Note{}, fmt.Errorf("gh did not print an issue address: %q", strings.TrimSpace(string(out)))
	}
	number, _ := strconv.Atoi(m[1])
	n, err = a.notes.SetIssue(cur.Key(), id, number, m[0])
	if err != nil {
		return Note{}, err
	}
	a.emitNotes()
	if _, err := a.issues.List(a.ctx, cur, true); err == nil {
		a.emitIssues()
	}
	return n, nil
}

// NoteToSession starts a session titled by the note, with the project's note command typed in.
func (a *App) NoteToSession(id string) (Session, error) {
	cur := a.sessions.Current()
	n, err := a.notes.Get(cur.Key(), id)
	if err != nil {
		return Session{}, err
	}
	return a.sessions.Create(n.title(), cur.NoteCommand(n.Text), 0)
}

func (a *App) TermOpen(id string, cols, rows int) error { return a.terms.Open(id, cols, rows) }
func (a *App) TermWrite(id, data string) error          { return a.terms.Write(id, data) }
func (a *App) TermResize(id string, cols, rows int) error {
	return a.terms.Resize(id, cols, rows)
}
func (a *App) TermClose(id string) { a.terms.Close(id) }

func (a *App) OpenURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return fmt.Errorf("not a web address: %q", rawURL)
	}
	a.host.openURL(rawURL)
	return nil
}

// issueBody is the issue template filled with the note's text. gh cannot upload
// pictures, so a note with pictures says they stay in agentos.
func issueBody(n Note) string {
	body := "## Description\n\n" + n.Text
	switch len(n.Images) {
	case 0:
	case 1:
		body += "\n\n(1 screenshot is attached to the note in agentos.)"
	default:
		body += fmt.Sprintf("\n\n(%d screenshots are attached to the note in agentos.)", len(n.Images))
	}
	return body
}

func (a *App) Stats(days int) (Stats, error) {
	return a.sessions.waits.Stats(a.sessions.Current().Key(), days, time.Now())
}

func (a *App) RefreshPRs() { a.life.Poll(a.ctx) }

func (a *App) AckPR(id string) error { return a.life.Ack(id) }

func (a *App) TypeInto(id, text string) error { return a.sessions.TypeInto(id, text) }

func (a *App) Cleanup(id string, force bool) error { return a.life.Cleanup(a.ctx, id, force) }

func (a *App) Cleanups() []Cleanup { return a.life.Cleanups(a.sessions.Current().Key()) }

func (a *App) browserCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(a.ctx, browserStartLimit+callLimit)
}

func (a *App) BrowserOpen(id, url string) (BrowserState, error) {
	ctx, cancel := a.browserCtx()
	defer cancel()
	return a.browsers.Open(ctx, id, url)
}

func (a *App) BrowserGoto(id, url string) error {
	ctx, cancel := a.browserCtx()
	defer cancel()
	return a.browsers.Goto(ctx, id, url)
}

func (a *App) BrowserNav(id, action string) error {
	ctx, cancel := a.browserCtx()
	defer cancel()
	return a.browsers.Nav(ctx, id, action)
}

func (a *App) BrowserInput(id string, in BrowserInput) error {
	ctx, cancel := context.WithTimeout(a.ctx, callLimit)
	defer cancel()
	return a.browsers.Input(ctx, id, in)
}

func (a *App) BrowserResize(id string, width, height int) error {
	ctx, cancel := context.WithTimeout(a.ctx, callLimit)
	defer cancel()
	return a.browsers.Resize(ctx, id, width, height)
}

func (a *App) BrowserView(id string, visible bool) error {
	ctx, cancel := context.WithTimeout(a.ctx, callLimit)
	defer cancel()
	return a.browsers.View(ctx, id, visible)
}

func (a *App) BrowserState(id string) BrowserState { return a.browsers.State(id) }

// BrowserScreenshot files the user's own capture of the tab as evidence.
func (a *App) BrowserScreenshot(id, caption string) (Evidence, error) {
	ctx, cancel := a.browserCtx()
	defer cancel()
	png, err := a.browsers.Screenshot(ctx, id, false, "")
	if err != nil {
		return Evidence{}, err
	}
	item, err := a.evidence.AddImage(id, png, caption, "user")
	if err == nil {
		a.evidenceChanged(id, false)
	}
	return item, err
}

func (a *App) BrowserClose(id string) {
	ctx, cancel := context.WithTimeout(a.ctx, callLimit)
	defer cancel()
	a.browsers.Close(ctx, id)
}

func (a *App) Evidence(id string) []Evidence { return a.evidence.List(id) }

func (a *App) DeleteEvidence(id, evidenceID string) error {
	err := a.evidence.Delete(id, evidenceID)
	if err == nil {
		a.evidenceChanged(id, false)
	}
	return err
}

// evidenceChanged tells the front end; an agent's new evidence also raises the session.
func (a *App) evidenceChanged(id string, byAgent bool) {
	a.host.emit("evidence", map[string]any{"id": id, "items": a.evidence.List(id)})
	if byAgent {
		a.host.emit("attention", map[string]string{"id": id, "state": "evidence"})
	}
	a.sessions.changed()
}

// HarnessCheck starts a session that reviews the project's harness; the owner sends its prompt.
func (a *App) HarnessCheck() (Session, error) {
	return a.sessions.Create("Harness check", harnessPrompt, 0)
}

func (a *App) Digest() Digest {
	cur := a.sessions.Current()
	return a.digests.View(cur.Key(), cur.DigestOn())
}

func (a *App) emitDigest(key string) {
	if cur := a.sessions.Current(); cur.Key() == key {
		a.host.emit("digest", a.digests.View(key, cur.DigestOn()))
	}
}

// RunDigest starts the current project's digest run in the background.
func (a *App) RunDigest() error { return a.startDigest(a.sessions.Current()) }

func (a *App) startDigest(proj project.Project) error {
	key := proj.Key()
	if err := a.digests.Begin(key, time.Now()); err != nil {
		return err
	}
	a.emitDigest(key)
	go func() {
		ctx, cancel := context.WithTimeout(a.ctx, digestTimeout)
		defer cancel()
		env := append(slices.DeleteFunc(os.Environ(), func(kv string) bool {
			return strings.HasPrefix(kv, "AGENTOS_SESSION=") || strings.HasPrefix(kv, digestProjects+"=")
		}), digestProjects+"="+key, "AGENTOS_SOCKET="+bus.SocketPath(a.stateDir))
		_, err := a.runEnv(ctx, proj.Dir, env, "claude", "-p", digestPrompt, "--allowedTools", digestTools)
		failure := ""
		switch {
		case ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded):
			failure = "the run took longer than 10 minutes"
		case errors.Is(err, exec.ErrNotFound):
			failure = "claude was not found on PATH"
		case err != nil:
			failure = firstLine(err.Error())
		}
		a.digests.Finish(key, time.Now(), failure)
		a.emitDigest(key)
	}()
	return nil
}

// digestLoop starts the current project's digest when it is due: first after a minute, then hourly.
func (a *App) digestLoop(ctx context.Context) {
	timer := time.NewTimer(a.digestFirst)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if cur := a.sessions.Current(); cur.DigestOn() && a.digests.Due(cur.Key(), time.Now()) {
				_ = a.startDigest(cur)
			}
			timer.Reset(a.digestTick)
		}
	}
}

func (a *App) DigestToNote(itemID string) (Note, error) {
	key := a.sessions.Current().Key()
	item, ok := a.digests.Item(key, itemID)
	if !ok {
		return Note{}, errors.New("no such digest item")
	}
	n, err := a.notes.Add(key, item.Title+"\n\n"+item.Why+"\n"+item.URL)
	if err != nil {
		return Note{}, err
	}
	if err := a.digests.Update(key, itemID, func(it *DigestItem) { it.NoteID = n.ID }); err != nil {
		return Note{}, err
	}
	a.emitNotes()
	a.emitDigest(key)
	return n, nil
}

func (a *App) DismissDigestItem(itemID string) error {
	key := a.sessions.Current().Key()
	if err := a.digests.Dismiss(key, itemID); err != nil {
		return err
	}
	a.emitDigest(key)
	return nil
}

// mediaHandler serves the pictures of notes and evidence under /media/.
func (a *App) mediaHandler() http.Handler { return mediaHandler(a.media) }
