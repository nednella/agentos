package main

import (
	"context"
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
type harness struct {
	app    *App
	rec    *recorder
	socket string
	dir    string
	state  string
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
	conf := fmt.Sprintf("agent: bash\nprojects:\n  - name: main\n    dir: %s\n", dir)
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
	rec := &recorder{}
	app := newApp(cfg, host{emit: rec.emit})
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
		_ = exec.Command("tmux", "-L", socket, "kill-server").Run()
		os.RemoveAll(state)
	})
	return &harness{app: app, rec: rec, socket: socket, dir: dir, state: state}
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
	app := newApp(cfg, host{emit: func(string, any) {}})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := app.start(ctx); err != nil {
		t.Fatal(err)
	}
	return app
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
