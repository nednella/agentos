// Package apptest runs the whole app on a private tmux socket, state dir and config, for the services' tests.
// A test must never use the default socket: it holds live sessions.
package apptest

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/browser"
	"github.com/nednella/agentos/desktop/control"
	"github.com/nednella/agentos/desktop/digest"
	"github.com/nednella/agentos/desktop/evidence"
	"github.com/nednella/agentos/desktop/internal/app"
	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/desktop/internal/scoped"
	"github.com/nednella/agentos/desktop/internal/warn"
	"github.com/nednella/agentos/desktop/issues"
	"github.com/nednella/agentos/desktop/notes"
	"github.com/nednella/agentos/desktop/projects"
	"github.com/nednella/agentos/desktop/sessions"
	"github.com/nednella/agentos/desktop/stats"
	"github.com/nednella/agentos/desktop/terminal"
	ctl "github.com/nednella/agentos/internal/control"
)

type Recorder struct {
	mu     sync.Mutex
	events []recorded
	clips  []string
	pick   string // what the folder picker returns
}

type recorded struct {
	name    string
	payload any
}

func (r *Recorder) Emit(name string, payload any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, recorded{name, payload})
}

func (r *Recorder) Picker() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pick, nil
}

func (r *Recorder) Clip(text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clips = append(r.clips, text)
}

// SetPick sets what the folder picker returns.
func (r *Recorder) SetPick(dir string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pick = dir
}

// HasClip says whether the clipboard was set to that text.
func (r *Recorder) HasClip(text string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Contains(r.clips, text)
}

// Output is everything a terminal stream has sent for the session, decoded.
func (r *Recorder) Output(id string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out bytes.Buffer
	for _, e := range r.events {
		if m, ok := e.payload.(map[string]string); ok && e.name == "term:data" && m["id"] == id {
			b, _ := base64.StdEncoding.DecodeString(m["data"])
			out.Write(b)
		}
	}
	return out.String()
}

func (r *Recorder) Count(name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if e.name == name {
			n++
		}
	}
	return n
}

// countAttention counts the attention events of one state.
func (r *Recorder) CountAttention(state string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if m, ok := e.payload.(map[string]string); ok && e.name == "attention" && m["state"] == state {
			n++
		}
	}
	return n
}

// last is the payload of the latest event of that name, or nil.
// CountAttentionFor counts the attention events of one state for one session.
func (r *Recorder) CountAttentionFor(id, state string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if m, ok := e.payload.(map[string]string); ok && e.name == "attention" && m["state"] == state && m["id"] == id {
			n++
		}
	}
	return n
}

// LastEvent is the name and payload of the latest event.
func (r *Recorder) LastEvent() (string, any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.events) == 0 {
		return "", nil
	}
	e := r.events[len(r.events)-1]
	return e.name, e.payload
}

// Last is the payload of the latest event of that name, or nil.
func (r *Recorder) Last(name string) any {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range slices.Backward(r.events) {
		if e.name == name {
			return e.payload
		}
	}
	return nil
}

// LastUI is the latest command the app told the front end to run.
func (r *Recorder) LastUI() control.UICommand {
	cmd, _ := r.Last("ui:command").(control.UICommand)
	return cmd
}

// lastList is the payload of the latest event of that name, which carries one project's list.
func lastList[T any](r *Recorder, name string) scoped.List[T] {
	list, _ := r.Last(name).(scoped.List[T])
	return list
}

// LastSessions is the list of the latest sessions event.
func (r *Recorder) LastSessions() []sessions.Session {
	return lastList[sessions.Session](r, "sessions").Items
}

// LastNotes is the list of the latest notes event.
func (r *Recorder) LastNotes() []notes.Note { return lastList[notes.Note](r, "notes").Items }

// LastIssues is the list of the latest issues event.
func (r *Recorder) LastIssues() []issues.Issue { return lastList[issues.Issue](r, "issues").Items }

// LastCleanups is the list of the latest cleanups event.
func (r *Recorder) LastCleanups() []sessions.Cleanup {
	return lastList[sessions.Cleanup](r, "cleanups").Items
}

// Warnings are the warnings events so far, in order.
func (r *Recorder) Warnings() []warn.Warning {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []warn.Warning
	for _, e := range r.events {
		if w, ok := e.payload.(warn.Warning); ok && e.name == "warnings" {
			out = append(out, w)
		}
	}
	return out
}

// Ended are the ids of the sessions that any sessions event showed as ended.
func (r *Recorder) Ended() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, e := range r.events {
		if list, ok := e.payload.(scoped.List[sessions.Session]); ok && e.name == "sessions" {
			for _, s := range list.Items {
				if s.State == "ended" && !slices.Contains(out, s.ID) {
					out = append(out, s.ID)
				}
			}
		}
	}
	return out
}

// ProjectOf is the project key of the latest event of that name (sessions, notes, issues or cleanups), or "".
func (r *Recorder) ProjectOf(name string) string {
	switch name {
	case "sessions":
		return lastList[sessions.Session](r, name).Project
	case "notes":
		return lastList[notes.Note](r, name).Project
	case "issues":
		return lastList[issues.Issue](r, name).Project
	case "cleanups":
		return lastList[sessions.Cleanup](r, name).Project
	}
	return ""
}

func Eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// FakeGH answers the gh calls with canned output; any other command runs for real.

type FakeGH struct {
	mu        sync.Mutex
	calls     []string
	Repo      error
	Create    string        // what gh issue create prints
	CreateErr error         // what gh issue create fails with, if set; it still prints Create
	Delay     time.Duration // how long gh issue create takes
	pr        string        // what gh pr list prints
	PRErr     error         // what gh pr list fails with, if set
	pulls     string        // the body gh api prints for the repo's pull request list
	etag      string        // its ETag: a request carrying it gets a 304
	hooks     []*io.PipeWriter
	hooked    bool // gh webhook forward works
}

func (f *FakeGH) Run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	if name != "gh" {
		return run.Exec(ctx, dir, name, args...)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, strings.Join(args, " "))
	switch {
	case args[0] == "api" && args[1] == "-i":
		if f.pulls == "" {
			return nil, fmt.Errorf("gh api: exit status 1: gh: HTTP 404: Not Found")
		}
		if i := slices.Index(args, "-H"); i >= 0 && args[i+1] == "If-None-Match: "+f.etag {
			return nil, fmt.Errorf("gh api: exit status 1: gh: HTTP 304")
		}
		return []byte("HTTP/2.0 200 OK\r\nEtag: " + f.etag + "\r\nX-Ratelimit-Remaining: 4999\r\n\r\n" + f.pulls), nil
	case args[0] == "pr":
		if f.PRErr != nil {
			return nil, f.PRErr
		}
		if f.pr == "" {
			return []byte("[]"), nil
		}
		return []byte(f.pr), nil
	case args[0] == "repo":
		if f.Repo != nil {
			return nil, f.Repo
		}
		return []byte(`{"nameWithOwner":"acme/widgets"}`), nil
	case args[0] == "api" && strings.HasSuffix(args[1], "/comments?per_page=100"):
		if !strings.Contains(args[1], "/issues/7/") {
			return []byte("[]"), nil
		}
		return []byte(`[{"user":{"login":"amy"},"created_at":"2026-09-03T10:00:00Z","body_html":"<p>Agreed.</p>"}]`), nil
	case args[0] == "api":
		if strings.HasSuffix(args[1], "/issues/7") {
			return []byte(`{"number":7,"body_html":"<h2>Description</h2>\n<p>Fix it.</p>","comments":1}`), nil
		}
		if strings.HasSuffix(args[1], "/issues/11") {
			return []byte(`{"number":11,"body_html":"","comments":0}`), nil
		}
		return nil, fmt.Errorf("gh api: HTTP 404: Not Found")
	case args[0] == "issue" && args[1] == "create":
		f.mu.Unlock()
		time.Sleep(f.Delay)
		f.mu.Lock()
		return []byte(f.Create), f.CreateErr
	case args[0] == "issue":
		return []byte(`[
			{"number":7,"title":"Fix the thing","url":"https://x/7","author":{"login":"ned"},"assignees":[{"login":"ned"},{"login":"amy"}],"createdAt":"2026-09-01T10:00:00Z","updatedAt":"2026-09-02T10:00:00Z","labels":[{"name":"ready"},{"name":"type:bug"}]},
			{"number":8,"title":"Plan it","url":"https://x/8","labels":[{"name":"needs-plan"},{"name":"type:feature"}]},
			{"number":9,"title":"Human","url":"https://x/9","labels":[{"name":"needs-human"}]},
			{"number":10,"title":"Maybe","url":"https://x/10","labels":[{"name":"idea"},{"name":"type:chore"}]},
			{"number":11,"title":"New","url":"https://x/11","labels":[]},
			{"number":12,"title":"Other","url":"https://x/12","labels":[{"name":"roadmap"}]}
		]`), nil
	}
	return nil, fmt.Errorf("unexpected %s %v", name, args)
}

func (f *FakeGH) IssueCalls() int { return f.Calls("issue list") }

// Calls counts the gh calls that start with prefix.
func (f *FakeGH) Calls(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

// SetPulls sets what the repo's pull request list answers: its ETag and the body.
func (f *FakeGH) SetPulls(etag, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.etag, f.pulls = etag, body
}

// Stream is the run.Streamer: gh webhook forward streams the events the test sends, when the test
// said it works; any other command fails.
func (f *FakeGH) Stream(_ context.Context, _, name string, args ...string) (io.ReadCloser, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, strings.Join(args, " "))
	if name != "gh" || args[0] != "webhook" || !f.hooked {
		return nil, fmt.Errorf("gh %s: exit status 1: unknown command %q for \"gh\"", args[0], args[0])
	}
	r, w := io.Pipe()
	f.hooks = append(f.hooks, w)
	return r, nil
}

// WebhookWorks makes gh webhook forward run, as it does where the user is admin.
func (f *FakeGH) WebhookWorks() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.hooked = true
}

// Event sends a webhook payload down every forward stream.
func (f *FakeGH) Event(payload string) {
	f.mu.Lock()
	hooks := slices.Clone(f.hooks)
	f.mu.Unlock()
	for _, w := range hooks {
		_, _ = io.WriteString(w, payload+"\n")
	}
}

// DropWebhooks ends every forward stream, as a lost connection would.
func (f *FakeGH) DropWebhooks() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, w := range f.hooks {
		_ = w.Close()
	}
	f.hooks = nil
}

// Forwards counts the gh webhook forward runs.
func (f *FakeGH) Forwards() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, "webhook forward") {
			n++
		}
	}
	return n
}

// SetPR sets what gh pr list prints.
func (f *FakeGH) SetPR(out string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pr = out
}

func (f *FakeGH) CallLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

// NoGH is a runner for tests that must not reach GitHub.
func NoGH(context.Context, string, string, ...string) ([]byte, error) {
	return nil, errors.New("gh is not available in this test")
}

// NoStream is a streamer for tests that must not start anything.
func NoStream(context.Context, string, string, ...string) (io.ReadCloser, error) {
	return nil, errors.New("streams are not available in this test")
}

// NoClaude is an env runner for tests that must not run claude.
func NoClaude(context.Context, string, []string, string, ...string) ([]byte, error) {
	return nil, errors.New("claude is not available in this test")
}

// Options changes the harness's config.
type Options struct {
	ProjectExtra string        // yaml lines added under the project "main"
	DigestFirst  time.Duration // wait before the first automatic digest check; an hour by default
	DigestTick   time.Duration
}

// Harness is an app with a plain bash as its agent.
type Harness struct {
	App    *app.App
	Rec    *Recorder
	GH     *FakeGH
	Claude *FakeClaude
	Socket string
	Dir    string
	State  string
	Conf   string

	projects *projects.Service
	sessions *sessions.Service
	terminal *terminal.Service
	issues   *issues.Service
	notes    *notes.Service
	stats    *stats.Service
	evidence *evidence.Service
	browser  *browser.Service
	digest   *digest.Service
}

// New starts an app for the test; the cleanup stops it and its tmux server.
func New(t *testing.T) *Harness { return NewWith(t, Options{}) }

func NewWith(t *testing.T, o Options) *Harness {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	state, err := os.MkdirTemp("", "aos")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	confPath := filepath.Join(state, "config.yaml")
	conf := fmt.Sprintf("agent: bash\nprojects:\n  - name: main\n    dir: %s\n    commands:\n      ready: \"/ship {n}\"\n      inbox: \"/investigate {n}\"\n%s", dir, o.ProjectExtra)
	if err := os.WriteFile(confPath, []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	socket := fmt.Sprintf("aostest-%d", os.Getpid())
	t.Setenv("AGENTOS_TMUX_SOCKET", socket)
	t.Setenv("AGENTOS_STATE_DIR", state)
	t.Setenv("AGENTOS_CONFIG", confPath)
	t.Setenv("AGENTOS_DIR", dir)
	t.Setenv("AGENTOS_DATA_DIR", filepath.Join(state, "data"))

	h := &Harness{Rec: &Recorder{}, GH: &FakeGH{}, Claude: &FakeClaude{}, Socket: socket, Dir: dir, State: state, Conf: confPath}
	h.App = h.build(t)
	h.App.Sessions().SetPrefillWait(300 * time.Millisecond)
	first, tick := o.DigestFirst, o.DigestTick
	if first == 0 {
		first, tick = time.Hour, time.Hour
	}
	h.App.Digests().SetLoop(first, tick) // a test must never start the real claude
	ctx, cancel := context.WithCancel(context.Background())
	if err := h.App.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		h.App.Stop()
		_ = exec.Command("tmux", "-L", socket, "kill-server").Run()
		os.RemoveAll(state)
	})
	return h
}

// build wires an app from the environment the harness set up.
func (h *Harness) build(t *testing.T) *app.App {
	t.Helper()
	cfg, err := app.Load()
	if err != nil {
		t.Fatal(err)
	}
	a := app.New(cfg, app.Host{Emit: h.Rec.Emit, Clipboard: h.Rec.Clip, PickDir: h.Rec.Picker}, h.GH.Run, h.GH.Stream, h.Claude.RunEnv)
	for _, svc := range a.Services() {
		switch s := svc.(type) {
		case *projects.Service:
			h.projects = s
		case *sessions.Service:
			h.sessions = s
		case *terminal.Service:
			h.terminal = s
		case *issues.Service:
			h.issues = s
		case *notes.Service:
			h.notes = s
		case *stats.Service:
			h.stats = s
		case *evidence.Service:
			h.evidence = s
		case *browser.Service:
			h.browser = s
		case *digest.Service:
			h.digest = s
		}
	}
	return a
}

// Restart starts a second app on the same dirs, as if the first was quit and opened again.
func (h *Harness) Restart(t *testing.T) *Harness {
	t.Helper()
	second := &Harness{Rec: &Recorder{}, GH: h.GH, Claude: h.Claude, Socket: h.Socket, Dir: h.Dir, State: h.State, Conf: h.Conf}
	second.App = second.build(t)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(second.App.Stop)
	if err := second.App.Start(ctx); err != nil {
		t.Fatal(err)
	}
	return second
}

// Sessions is the session list behind the bound service.
func (h *Harness) Sessions() *sessions.Sessions { return h.App.Sessions() }

// Notes is the notes store behind the bound service.
func (h *Harness) Notes() *notes.Notes { return h.App.Notes() }

// Hook runs the built agentos command as Claude Code would.
func (h *Harness) Hook(t *testing.T, id, event, stdin string) {
	t.Helper()
	cmd := exec.Command(CLIPath(t), "hook", event)
	cmd.Env = append(os.Environ(), "AGENTOS_SESSION="+id, "AGENTOS_SOCKET="+filepath.Join(h.State, "agentos.sock"))
	cmd.Stdin = strings.NewReader(stdin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("agentos hook: %v: %s", err, out)
	}
}

// Pane is what the session's terminal shows now.
func (h *Harness) Pane(t *testing.T, id string) string {
	t.Helper()
	out, err := exec.Command("tmux", "-L", h.Socket, "capture-pane", "-p", "-t", id).Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// Session is the session with that id in the current project's list.
func (h *Harness) Session(id string) (sessions.Session, bool) {
	for _, s := range h.Sessions().List() {
		if s.ID == id {
			return s, true
		}
	}
	return sessions.Session{}, false
}

var (
	cliOnce sync.Once
	cliBin  string
	cliDir  string
	cliErr  error
)

// CLIPath builds the agentos command from this checkout, so hooks run the code under test.
func CLIPath(t *testing.T) string {
	t.Helper()
	cliOnce.Do(func() {
		if cliDir, cliErr = os.MkdirTemp("", "aoscli"); cliErr != nil {
			return
		}
		cliBin = filepath.Join(cliDir, "agentos")
		_, file, _, _ := runtime.Caller(0)
		root := filepath.Join(filepath.Dir(file), "..", "..", "..")
		cmd := exec.Command("go", "build", "-o", cliBin, ".")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			cliErr = fmt.Errorf("building the agentos command: %w: %s", err, out)
		}
	})
	if cliErr != nil {
		t.Fatal(cliErr)
	}
	return cliBin
}

// Main runs a package's tests and removes the command they built.
func Main(m *testing.M) int {
	code := m.Run()
	if cliDir != "" {
		os.RemoveAll(cliDir)
	}
	return code
}

// Exists says whether a path exists.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// The methods below call the bound services the way the front end does.

func (h *Harness) Snapshot() projects.Snapshot { return h.projects.Snapshot() }
func (h *Harness) SwitchProject(name string) (projects.Snapshot, error) {
	return h.projects.SwitchProject(name)
}
func (h *Harness) AddProject() (projects.Snapshot, error) { return h.projects.AddProject() }
func (h *Harness) AddProjectDir(dir string) (projects.Snapshot, error) {
	return h.projects.AddProjectDir(dir)
}
func (h *Harness) RemoveProject(name string) (projects.Snapshot, error) {
	return h.projects.RemoveProject(name)
}

func (h *Harness) NewSession(title, prefill string) (sessions.Session, error) {
	return h.sessions.NewSession(title, prefill)
}
func (h *Harness) ShellOpen() (sessions.ShellInfo, error) { return h.sessions.ShellOpen() }
func (h *Harness) KillSession(id string) error            { return h.sessions.KillSession(id) }
func (h *Harness) DismissSession(id string) error         { return h.sessions.DismissSession(id) }
func (h *Harness) RenameSession(id, title string) error   { return h.sessions.RenameSession(id, title) }
func (h *Harness) TypeInto(id, text string) error         { return h.sessions.TypeInto(id, text) }
func (h *Harness) RefreshPRs()                            { h.sessions.RefreshPRs() }
func (h *Harness) AckPR(id string) error                  { return h.sessions.AckPR(id) }
func (h *Harness) Cleanup(id string, force bool) error    { return h.sessions.Cleanup(id, force) }
func (h *Harness) Cleanups() []sessions.Cleanup           { return h.sessions.Cleanups() }

func (h *Harness) TermOpen(id string, cols, rows int) error {
	return h.terminal.TermOpen(id, cols, rows)
}
func (h *Harness) TermWrite(id, data string) error { return h.terminal.TermWrite(id, data) }
func (h *Harness) TermResize(id string, cols, rows int) error {
	return h.terminal.TermResize(id, cols, rows)
}
func (h *Harness) TermClose(id string) { h.terminal.TermClose(id) }

func (h *Harness) Issues(refresh bool) ([]issues.Issue, error) { return h.issues.Issues(refresh) }
func (h *Harness) StartIssue(number int) (sessions.Session, error) {
	return h.issues.StartIssue(number)
}
func (h *Harness) IssueDetail(number int) (issues.IssueDetail, error) {
	return h.issues.IssueDetail(number)
}

func (h *Harness) AddNote(text string) (notes.Note, error) { return h.notes.AddNote(text) }
func (h *Harness) UpdateNote(id, text string) (notes.Note, error) {
	return h.notes.UpdateNote(id, text)
}
func (h *Harness) SetNotePinned(id string, pinned bool) (notes.Note, error) {
	return h.notes.SetNotePinned(id, pinned)
}
func (h *Harness) SetNoteArchived(id string, archived bool) (notes.Note, error) {
	return h.notes.SetNoteArchived(id, archived)
}
func (h *Harness) DeleteNote(id string) error { return h.notes.DeleteNote(id) }
func (h *Harness) AddNoteImage(id, data, mime string) (notes.Note, error) {
	return h.notes.AddNoteImage(id, data, mime)
}
func (h *Harness) RemoveNoteImage(id, url string) (notes.Note, error) {
	return h.notes.RemoveNoteImage(id, url)
}
func (h *Harness) NoteToIssue(id string) error { return h.notes.NoteToIssue(id) }
func (h *Harness) NoteToSession(id string) (sessions.Session, error) {
	return h.notes.NoteToSession(id)
}

func (h *Harness) Stats(days int) (stats.Stats, error) { return h.stats.Stats(days) }

// PRJSON is what gh pr list prints for one pull request.
func PRJSON(state string, draft bool, rollup string, comments, reviews int) string {
	repeat := func(n int) string {
		return "[" + strings.TrimSuffix(strings.Repeat(`{"id":"x"},`, n), ",") + "]"
	}
	return fmt.Sprintf(`[{"number":12,"url":"https://github.com/acme/widgets/pull/12","state":%q,"isDraft":%v,"statusCheckRollup":%s,"comments":%s,"reviews":%s,"updatedAt":"2026-10-01T12:00:00Z"}]`,
		state, draft, rollup, repeat(comments), repeat(reviews))
}

// WithHead adds the commit the PR's branch ended on to what PRJSON printed.
func WithHead(prJSON, oid string) string {
	return strings.Replace(prJSON, `"isDraft"`, fmt.Sprintf(`"headRefOid":%q,"isDraft"`, oid), 1)
}

// Ask sends a command to the app's control socket, as the agentos command does.
func (h *Harness) Ask(t *testing.T, req ctl.Request) ctl.Response {
	t.Helper()
	resp, err := ctl.Call(context.Background(), ctl.SocketPath(h.State), req)
	if err != nil {
		t.Fatalf("asking %s: %v", req.Cmd, err)
	}
	return resp
}

func (h *Harness) Evidence(id string) []evidence.Evidence { return h.evidence.Evidence(id) }
func (h *Harness) DeleteEvidence(id, evidenceID string) error {
	return h.evidence.DeleteEvidence(id, evidenceID)
}

// CountFrames counts the live-view frames sent for a session.
func (r *Recorder) CountFrames(id string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if m, ok := e.payload.(map[string]any); ok && e.name == "browser:frame" && m["id"] == id {
			n++
		}
	}
	return n
}

// LastFrame is the latest live-view frame.
func (r *Recorder) LastFrame() map[string]any {
	r.mu.Lock()
	defer r.mu.Unlock()
	var last map[string]any
	for _, e := range r.events {
		if m, ok := e.payload.(map[string]any); ok && e.name == "browser:frame" {
			last = m
		}
	}
	return last
}

// HasFrame says whether a live-view frame of that size was sent.
func (r *Recorder) HasFrame(width, height int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range r.events {
		if m, ok := e.payload.(map[string]any); ok && e.name == "browser:frame" && m["width"] == width && m["height"] == height {
			return true
		}
	}
	return false
}

func (h *Harness) BrowserOpen(id, url string) (browser.BrowserState, error) {
	return h.browser.BrowserOpen(id, url)
}
func (h *Harness) BrowserState(id string) browser.BrowserState { return h.browser.BrowserState(id) }
func (h *Harness) BrowserView(id string, visible bool) error {
	return h.browser.BrowserView(id, visible)
}
func (h *Harness) BrowserResize(id string, w, ht int) error {
	return h.browser.BrowserResize(id, w, ht)
}
func (h *Harness) BrowserInput(id string, in browser.BrowserInput) error {
	return h.browser.BrowserInput(id, in)
}
func (h *Harness) BrowserScreenshot(id, caption string) (evidence.Evidence, error) {
	return h.browser.BrowserScreenshot(id, caption)
}
func (h *Harness) BrowserClose(id string) { h.browser.BrowserClose(id) }

func (h *Harness) HarnessCheck() (sessions.Session, error) { return h.sessions.HarnessCheck() }

// DigestCall is one run of claude as the digest makes it.
type DigestCall struct {
	Dir  string
	Env  []string
	Args []string
}

// FakeClaude stands in for claude: it records each call and runs Do, if set, as claude would act.
type FakeClaude struct {
	mu    sync.Mutex
	calls []DigestCall
	Do    func(call DigestCall)
	Err   error
}

// RunEnv is the run.EnvRunner of the digest.
func (f *FakeClaude) RunEnv(_ context.Context, dir string, env []string, name string, args ...string) ([]byte, error) {
	call := DigestCall{Dir: dir, Env: env, Args: append([]string{name}, args...)}
	f.mu.Lock()
	f.calls = append(f.calls, call)
	do, err := f.Do, f.Err
	f.mu.Unlock()
	if do != nil {
		do(call)
	}
	return nil, err
}

// Calls are the runs so far.
func (f *FakeClaude) Calls() []DigestCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

// SetDo sets what claude does when called.
func (f *FakeClaude) SetDo(do func(DigestCall)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Do = do
}

func (h *Harness) Digest() digest.Digest                      { return h.digest.Digest() }
func (h *Harness) RunDigest() error                           { return h.digest.RunDigest() }
func (h *Harness) DismissDigestItem(id string) error          { return h.digest.DismissDigestItem(id) }
func (h *Harness) DigestToNote(id string) (notes.Note, error) { return h.digest.DigestToNote(id) }
