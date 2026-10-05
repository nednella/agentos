package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/bus"
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
	pickDir   func() (string, error) // "" when the user cancels
}

type config struct {
	registry *Registry
	project  project.Project
	stateDir string
	dataDir  string // notes: may be a synced folder
	tmux     *term.Tmux
	agent    agent.Agent
}

// loadConfig reads the config file and the environment. The project is the one holding AGENTOS_DIR
// or the current folder. An app started from Finder (folder "/") gets the first
// configured project, else the home folder.
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
	dataDir := cmp.Or(cfg.DataDir, filepath.Join(home, ".local", "share", "agentos"))
	if override := os.Getenv("AGENTOS_DATA_DIR"); override != "" {
		dataDir = override
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
	return config{registry: &Registry{path: path, cfg: cfg}, project: current, stateDir: stateDir, dataDir: dataDir, tmux: tmux, agent: pickAgent(cfg.Agent)}, nil
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
	run      runner
	media    map[string]string // folder of a /media/ URL -> the dir that holds it
}

func newApp(c config, h host, run runner) *App {
	a := &App{
		ctx:      context.Background(),
		host:     h,
		stateDir: c.stateDir,
		sessions: newSessions(c.tmux, c.agent, c.stateDir, c.registry, c.project, h.emit),
		terms:    newTerms(c.tmux, h.emit, h.clipboard),
		issues:   newIssues(run),
		notes:    &Notes{dir: c.dataDir},
		run:      run,
		media:    map[string]string{mediaFolder: c.dataDir},
	}
	a.sessions.repoOf = a.issues.CachedRepo
	a.sessions.onIssues = a.emitIssues
	return a
}

// start begins listening for hooks; it runs until ctx ends.
func (a *App) start(ctx context.Context) error {
	a.ctx = ctx
	return a.sessions.Start(ctx)
}

func (a *App) stop() { a.terms.CloseAll() }

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

func (a *App) emitNotes() {
	notes, err := a.notes.List(a.sessions.Current().Key())
	if err != nil {
		log.Printf("agentos: %v", err)
		return
	}
	a.host.emit("notes", notes)
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

func (a *App) TermOpen(id string, cols, rows int) error { return a.terms.Open(id, cols, rows) }
func (a *App) TermWrite(id, data string) error          { return a.terms.Write(id, data) }
func (a *App) TermResize(id string, cols, rows int) error {
	return a.terms.Resize(id, cols, rows)
}
func (a *App) TermClose(id string) { a.terms.Close(id) }

// mediaHandler serves the note pictures under /media/.
func (a *App) mediaHandler() http.Handler { return mediaHandler(a.media) }
