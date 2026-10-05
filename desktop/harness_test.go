package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// recorder stands in for the window: it keeps every event the app emits.
type recorder struct {
	mu     sync.Mutex
	events []recorded
	clips  []string
}

type recorded struct {
	name    string
	payload any
}

func (r *recorder) emit(name string, payload any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, recorded{name, payload})
}

func (r *recorder) clip(text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clips = append(r.clips, text)
}

// output is everything a terminal stream has sent for the session, decoded.
func (r *recorder) output(id string) string {
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

func (r *recorder) count(name string) int {
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
func (r *recorder) countAttention(state string) int {
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
func (r *recorder) last(name string) any {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range slices.Backward(r.events) {
		if e.name == name {
			return e.payload
		}
	}
	return nil
}

func (r *recorder) lastSessions() []Session {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range slices.Backward(r.events) {
		if e.name == "sessions" {
			return e.payload.([]Session)
		}
	}
	return nil
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

// harness is an app on a private tmux socket, state dir and config, with a plain
// bash as its agent. A test must never use the default socket: it holds live sessions.
// fakeGH answers the gh calls with canned output; any other command runs for real.

type fakeGH struct {
	mu    sync.Mutex
	calls []string
	repo  error
}

func (f *fakeGH) run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	if name != "gh" {
		return execRunner(ctx, dir, name, args...)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, strings.Join(args, " "))
	switch {
	case args[0] == "repo":
		if f.repo != nil {
			return nil, f.repo
		}
		return []byte(`{"nameWithOwner":"acme/widgets"}`), nil
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

func (f *fakeGH) issueCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, "issue list") {
			n++
		}
	}
	return n
}

type harness struct {
	app    *App
	rec    *recorder
	gh     *fakeGH
	socket string
	dir    string
	state  string
}

// noGH is a runner for tests that must not reach GitHub.
func noGH(context.Context, string, string, ...string) ([]byte, error) {
	return nil, errors.New("gh is not available in this test")
}

func newHarness(t *testing.T) *harness { return newHarnessWith(t, nil) }

// newHarnessWith lets a test change the app before it starts.
func newHarnessWith(t *testing.T, tweak func(*App)) *harness {
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
	conf := fmt.Sprintf("agent: bash\nprojects:\n  - name: main\n    dir: %s\n    commands:\n      ready: \"/ship {n}\"\n      inbox: \"/investigate {n}\"\n", dir)
	if err := os.WriteFile(confPath, []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	socket := fmt.Sprintf("aostest-%d", os.Getpid())
	t.Setenv("AGENTOS_TMUX_SOCKET", socket)
	t.Setenv("AGENTOS_STATE_DIR", state)
	t.Setenv("AGENTOS_CONFIG", confPath)
	t.Setenv("AGENTOS_DIR", dir)

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	rec, gh := &recorder{}, &fakeGH{}
	app := newApp(cfg, host{emit: rec.emit, clipboard: rec.clip}, gh.run)
	app.sessions.prefillFor = 300 * time.Millisecond
	if tweak != nil {
		tweak(app)
	}
	ctx, cancel := context.WithCancel(context.Background())
	if err := app.start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		app.stop()
		_ = exec.Command("tmux", "-L", socket, "kill-server").Run()
		os.RemoveAll(state)
	})
	return &harness{app: app, rec: rec, gh: gh, socket: socket, dir: dir, state: state}
}

// hook runs the built agentos command as Claude Code would.
func (h *harness) hook(t *testing.T, id, event, stdin string) {
	t.Helper()
	cmd := exec.Command(cliPath(t), "hook", event)
	cmd.Env = append(os.Environ(), "AGENTOS_SESSION="+id, "AGENTOS_SOCKET="+filepath.Join(h.state, "agentos.sock"))
	cmd.Stdin = strings.NewReader(stdin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("agentos hook: %v: %s", err, out)
	}
}

// pane is what the session's terminal shows now.
func (h *harness) pane(t *testing.T, id string) string {
	t.Helper()
	out, err := exec.Command("tmux", "-L", h.socket, "capture-pane", "-p", "-t", id).Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// session is the session with that id in the current project's list.
func (h *harness) session(id string) (Session, bool) {
	for _, s := range h.app.sessions.List() {
		if s.ID == id {
			return s, true
		}
	}
	return Session{}, false
}

var (
	cliOnce sync.Once
	cliBin  string
	cliDir  string
	cliErr  error
)

// cliPath builds the agentos command from this checkout, so hooks run the code under test.
func cliPath(t *testing.T) string {
	t.Helper()
	cliOnce.Do(func() {
		if cliDir, cliErr = os.MkdirTemp("", "aoscli"); cliErr != nil {
			return
		}
		cliBin = filepath.Join(cliDir, "agentos")
		if out, err := exec.Command("go", "build", "-o", cliBin, "..").CombinedOutput(); err != nil {
			cliErr = fmt.Errorf("building the agentos command: %w: %s", err, out)
		}
	})
	if cliErr != nil {
		t.Fatal(cliErr)
	}
	return cliBin
}

func TestMain(m *testing.M) {
	code := m.Run()
	if cliDir != "" {
		os.RemoveAll(cliDir)
	}
	os.Exit(code)
}

// restart starts a second app on the same dirs, as if the first was quit and opened again.
func (h *harness) restart(t *testing.T) *App {
	t.Helper()
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	app := newApp(cfg, host{emit: func(string, any) {}, clipboard: func(string) {}}, h.gh.run)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	t.Cleanup(app.stop)
	if err := app.start(ctx); err != nil {
		t.Fatal(err)
	}
	return app
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
