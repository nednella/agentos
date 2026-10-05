package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
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

	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/session"
)

type recorder struct {
	mu     sync.Mutex
	events []recorded
	clips  []string
	pick   string
}

func (r *recorder) picker() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pick, nil
}

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

type fakeGH struct {
	mu     sync.Mutex
	calls  []string
	repo   error
	create string
	pr     string // what gh pr list prints; git and sh run for real
}

func (f *fakeGH) run(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	if name != "gh" {
		return execRunner(ctx, dir, name, args...)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, strings.Join(args, " "))
	switch {
	case args[0] == "pr":
		if f.pr == "" {
			return []byte("[]"), nil
		}
		return []byte(f.pr), nil
	case args[0] == "repo":
		if f.repo != nil {
			return nil, f.repo
		}
		return []byte(`{"nameWithOwner":"acme/widgets"}`), nil
	case args[0] == "issue" && args[1] == "create":
		return []byte(f.create), nil
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
	conf   string
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
	conf := fmt.Sprintf("agent: bash\nprojects:\n  - name: main\n    dir: %s\n    commands:\n      ready: \"/ship {n}\"\n", dir)
	if err := os.WriteFile(confPath, []byte(conf), 0o600); err != nil {
		t.Fatal(err)
	}
	socket := fmt.Sprintf("aostest-%d", os.Getpid())
	t.Setenv("AGENTOS_TMUX_SOCKET", socket)
	t.Setenv("AGENTOS_STATE_DIR", state)
	t.Setenv("AGENTOS_CONFIG", confPath)
	t.Setenv("AGENTOS_DIR", dir)
	t.Setenv("AGENTOS_DATA_DIR", filepath.Join(state, "data"))

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	rec, gh := &recorder{}, &fakeGH{}
	app := newApp(cfg, host{emit: rec.emit, clipboard: rec.clip, openURL: func(string) {}, pickDir: rec.picker}, gh.run)
	app.sessions.prefillFor = 300 * time.Millisecond
	app.digestFirst = time.Hour // a test must never start the real claude
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
	return &harness{app: app, rec: rec, gh: gh, socket: socket, dir: dir, state: state, conf: confPath}
}

func (h *harness) hook(t *testing.T, id, event, stdin string) {
	t.Helper()
	cmd := exec.Command(cliPath(t), "hook", event)
	cmd.Env = append(os.Environ(), "AGENTOS_SESSION="+id, "AGENTOS_SOCKET="+filepath.Join(h.state, "agentos.sock"))
	cmd.Stdin = strings.NewReader(stdin)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("agentos hook: %v: %s", err, out)
	}
}

func (h *harness) pane(t *testing.T, id string) string {
	t.Helper()
	out, err := exec.Command("tmux", "-L", h.socket, "capture-pane", "-p", "-t", id).Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestSessionLifecycle(t *testing.T) {
	h := newHarness(t)

	first, err := h.app.NewSession("first", "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.app.NewSession("second", "")
	if err != nil {
		t.Fatal(err)
	}
	snap := h.app.Snapshot()
	if len(snap.Sessions) != 2 || snap.Project.Dir != h.dir || snap.Sessions[0].History == nil {
		t.Fatalf("snapshot = %+v", snap)
	}
	if first.N != 1 || second.N != 2 || first.State != "idle" || first.CreatedAt == 0 {
		t.Errorf("sessions = %+v %+v", first, second)
	}

	t.Run("terminal", func(t *testing.T) {
		if err := h.app.TermOpen(first.ID, 100, 30); err != nil {
			t.Fatal(err)
		}
		eventually(t, "shell output", func() bool { return len(h.rec.output(first.ID)) > 0 })
		if err := h.app.TermWrite(first.ID, "echo $((40+2))\n"); err != nil {
			t.Fatal(err)
		}
		eventually(t, "echo result", func() bool { return strings.Contains(h.rec.output(first.ID), "\r\n42\r\n") })

		if err := h.app.TermResize(first.ID, 77, 20); err != nil {
			t.Fatal(err)
		}
		time.Sleep(300 * time.Millisecond)
		if err := h.app.TermWrite(first.ID, "stty size\n"); err != nil {
			t.Fatal(err)
		}
		eventually(t, "resized pty", func() bool { return strings.Contains(h.rec.output(first.ID), "\r\n20 77\r\n") })
	})

	t.Run("clipboard", func(t *testing.T) {
		cmd := `printf '\033]52;c;%s\a' $(printf copied-text | base64)` + "\n"
		if err := h.app.TermWrite(first.ID, cmd); err != nil {
			t.Fatal(err)
		}
		eventually(t, "clipboard", func() bool {
			h.rec.mu.Lock()
			defer h.rec.mu.Unlock()
			return slices.Contains(h.rec.clips, "copied-text")
		})
		if !strings.Contains(h.rec.output(first.ID), "\x1b]52;") {
			t.Error("the OSC 52 bytes were not forwarded")
		}
	})

	t.Run("hooks", func(t *testing.T) {
		before := h.rec.count("attention")
		h.hook(t, second.ID, "Notification", `{"message":"Claude needs your permission"}`)
		eventually(t, "waiting", func() bool {
			s := h.rec.lastSessions()
			return len(s) == 2 && s[0].ID == second.ID && s[0].State == "waiting"
		})
		s := h.rec.lastSessions()
		if s[0].Detail != "Claude needs your permission" || len(s[0].History) != 2 || s[0].LastEventAt == 0 {
			t.Errorf("waiting session = %+v", s[0])
		}
		if h.rec.count("attention") != before+1 {
			t.Errorf("attention events = %d, want %d", h.rec.count("attention"), before+1)
		}
		h.rec.mu.Lock()
		last := h.rec.events[len(h.rec.events)-1]
		h.rec.mu.Unlock()
		if got, _ := json.Marshal(last.payload); last.name != "attention" || string(got) != fmt.Sprintf(`{"id":%q,"state":"waiting"}`, second.ID) {
			t.Errorf("last event = %s %s", last.name, got)
		}

		h.hook(t, first.ID, "UserPromptSubmit", `{"prompt":"go"}`)
		eventually(t, "working", func() bool {
			s := h.rec.lastSessions()
			return len(s) == 2 && s[1].ID == first.ID && s[1].State == "working"
		})
		repliedBefore := h.rec.countAttention("replied")
		h.hook(t, first.ID, "Stop", `{}`)
		eventually(t, "idle after the reply", func() bool {
			s := h.rec.lastSessions()
			return len(s) == 2 && s[1].ID == first.ID && s[1].State == "idle"
		})
		if h.rec.countAttention("replied") != repliedBefore+1 {
			t.Error("no replied attention for a turn that ended")
		}
		h.hook(t, second.ID, "PreToolUse", `{"tool_name":"Edit","tool_input":{"file_path":"a.tsx"}}`)
		eventually(t, "idle then working order", func() bool {
			s := h.rec.lastSessions()
			return len(s) == 2 && s[0].ID == first.ID && s[1].State == "working" && s[1].Detail == "Edit a.tsx"
		})
	})

	t.Run("rename and kill", func(t *testing.T) {
		if err := h.app.RenameSession(second.ID, "renamed"); err != nil {
			t.Fatal(err)
		}
		eventually(t, "rename", func() bool {
			s := h.rec.lastSessions()
			return len(s) == 2 && s[1].Title == "renamed"
		})
		exits := h.rec.count("term:exit")
		if err := h.app.KillSession(first.ID); err != nil {
			t.Fatal(err)
		}
		eventually(t, "the killed session to show as ended", func() bool {
			s := h.rec.lastSessions()
			return len(s) == 2 && s[1].ID == first.ID && s[1].State == "ended" && s[1].EndedAt > 0
		})
		if got := h.app.Snapshot().Sessions; len(got) != 2 || got[0].ID != second.ID || got[1].State != "ended" || got[1].Title != "first" {
			t.Errorf("sessions after kill = %+v", got)
		}
		if h.rec.count("term:exit") != exits {
			t.Error("a terminal we closed ourselves reported an exit")
		}
		if out, err := exec.Command("tmux", "-L", h.socket, "has-session", "-t", first.ID).CombinedOutput(); err == nil {
			t.Errorf("the killed tmux session is still there: %s", out)
		}
		if err := h.app.KillSession(first.ID); err == nil {
			t.Error("killing an ended session succeeded")
		}
		if err := h.app.DismissSession(second.ID); err == nil {
			t.Error("dismissed a session that is running")
		}
		if err := h.app.DismissSession(first.ID); err != nil {
			t.Fatal(err)
		}
		eventually(t, "the dismissed row to go", func() bool { return len(h.rec.lastSessions()) == 1 })
		time.Sleep(2500 * time.Millisecond) // a poll must not bring it back
		if got := h.app.Snapshot().Sessions; len(got) != 1 || got[0].ID != second.ID {
			t.Errorf("sessions after dismiss and a poll = %+v", got)
		}
	})

	t.Run("session ends on its own", func(t *testing.T) {
		if err := h.app.TermOpen(second.ID, 80, 24); err != nil {
			t.Fatal(err)
		}
		exits := h.rec.count("term:exit")
		if err := exec.Command("tmux", "-L", h.socket, "kill-session", "-t", second.ID).Run(); err != nil {
			t.Fatal(err)
		}
		eventually(t, "term:exit", func() bool { return h.rec.count("term:exit") > exits })
		eventually(t, "the row to turn ended", func() bool {
			s := h.rec.lastSessions()
			return len(s) == 1 && s[0].State == "ended" && s[0].EndedAt > 0 && s[0].Title == "renamed"
		})
		if h.app.Snapshot().Projects[0].Sessions != 0 {
			t.Error("an ended session counts as a running one")
		}

		// A new session does not take the ended one's number.
		third, err := h.app.NewSession("third", "")
		if err != nil || third.ID == second.ID {
			t.Errorf("new session = %+v, %v", third, err)
		}
		if err := h.app.DismissSession(second.ID); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPrefillIsTypedNotSubmitted(t *testing.T) {
	h := newHarness(t)
	s, err := h.app.NewSession("p", "typed prefill text")
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "prefill on the prompt", func() bool { return strings.Contains(h.pane(t, s.ID), "typed prefill text") })
	time.Sleep(300 * time.Millisecond)
	if pane := h.pane(t, s.ID); strings.Contains(pane, "not found") {
		t.Errorf("prefill was submitted:\n%s", pane)
	}
}

func TestIssues(t *testing.T) {
	h := newHarness(t)

	got, err := h.app.Issues(false)
	if err != nil {
		t.Fatal(err)
	}
	var lanes []string
	for _, is := range got {
		lanes = append(lanes, fmt.Sprintf("%d:%s:%s", is.Number, is.Lane, is.Type))
	}
	want := []string{"7:ready:bug", "8:plan:feature", "9:you:", "10:idea:chore", "11:inbox:", "12:idea:"}
	if !slices.Equal(lanes, want) {
		t.Errorf("issues = %v, want %v", lanes, want)
	}
	first := got[0]
	if first.Author != "ned" || !slices.Equal(first.Assignees, []string{"ned", "amy"}) || !slices.Equal(first.Labels, []string{"ready", "type:bug"}) ||
		first.CreatedAt != time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC).UnixMilli() || first.UpdatedAt <= first.CreatedAt {
		t.Errorf("issue 7 = %+v", first)
	}
	if raw, _ := json.Marshal(got[1]); !strings.Contains(string(raw), `"assignees":[]`) || !strings.Contains(string(raw), `"labels":["needs-plan","type:feature"]`) || !strings.Contains(string(raw), `"author":""`) {
		t.Errorf("issue 8 json = %s", raw)
	}
	if repo := h.app.Snapshot().Project.Repo; repo != "acme/widgets" {
		t.Errorf("repo = %q", repo)
	}
	if _, err := h.app.Issues(false); err != nil || h.gh.issueCalls() != 1 {
		t.Errorf("cached call: err=%v, gh issue list ran %d times", err, h.gh.issueCalls())
	}
	if _, err := h.app.Issues(true); err != nil || h.gh.issueCalls() != 2 {
		t.Errorf("refresh: err=%v, gh issue list ran %d times", err, h.gh.issueCalls())
	}

	s, err := h.app.StartIssue(7)
	if err != nil {
		t.Fatal(err)
	}
	if s.Issue != 7 || s.Title != "#7 Fix the thing" {
		t.Errorf("session = %+v", s)
	}
	eventually(t, "/ship typed", func() bool { return strings.Contains(h.pane(t, s.ID), "/ship 7") })
	again, err := h.app.StartIssue(7)
	if err != nil || again.ID != s.ID {
		t.Errorf("second start = %+v, %v", again, err)
	}
	got, _ = h.app.Issues(false)
	if got[0].SessionID != s.ID || got[1].SessionID != "" {
		t.Errorf("session ids = %q %q", got[0].SessionID, got[1].SessionID)
	}
	if _, err := h.app.StartIssue(999); err == nil {
		t.Error("starting an unknown issue succeeded")
	}
	plan, err := h.app.StartIssue(8)
	if err != nil {
		t.Fatal(err)
	}
	issuesEvent := func(number int) string {
		list, _ := h.rec.last("issues").([]Issue)
		for _, is := range list {
			if is.Number == number {
				return is.SessionID
			}
		}
		return "?"
	}
	eventually(t, "issues event with the session", func() bool { return issuesEvent(7) == s.ID })
	if err := h.app.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "issues event after the session ends", func() bool { return issuesEvent(7) == "" })
	eventually(t, "/investigate typed", func() bool { return strings.Contains(h.pane(t, plan.ID), "/investigate 8") })
}

func TestIssuesWithoutRepo(t *testing.T) {
	gh := &fakeGH{repo: errors.New("not a repo")}
	is := newIssues(gh.run)
	list, err := is.List(context.Background(), project.Project{Dir: "/x"}, false)
	if err != nil || list == nil || len(list) != 0 {
		t.Errorf("List = %v, %v", list, err)
	}
	if repo := is.Repo(context.Background(), "/x"); repo != "" {
		t.Errorf("repo = %q", repo)
	}
}

func TestCopyOSC52(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString([]byte("hello"))
	tests := []struct {
		name   string
		chunks []string
		want   []string
	}{
		{"bel", []string{"x\x1b]52;c;" + b64 + "\ay"}, []string{"hello"}},
		{"string terminator", []string{"\x1b]52;;" + b64 + "\x1b\\"}, []string{"hello"}},
		{"split across reads", []string{"ab\x1b]5", "2;c;aGVs", "bG8=\a"}, []string{"hello"}},
		{"two sequences", []string{"\x1b]52;c;" + b64 + "\a\x1b]52;c;" + b64 + "\a"}, []string{"hello", "hello"}},
		{"query is ignored", []string{"\x1b]52;c;?\a"}, nil},
		{"other osc", []string{"\x1b]0;title\a"}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			s := &stream{clip: func(text string) { got = append(got, text) }}
			for _, c := range tt.chunks {
				s.copyOSC52([]byte(c))
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("clipboard = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOpenURL(t *testing.T) {
	var opened []string
	a := &App{host: host{openURL: func(u string) { opened = append(opened, u) }}}
	tests := []struct {
		url string
		ok  bool
	}{{"https://github.com/x/y/issues/1", true}, {"file:///etc/passwd", false}, {"javascript:alert(1)", false}, {"", false}}
	for _, tt := range tests {
		if err := a.OpenURL(tt.url); (err == nil) != tt.ok {
			t.Errorf("OpenURL(%q) error = %v", tt.url, err)
		}
	}
	if len(opened) != 1 {
		t.Errorf("opened = %v", opened)
	}
}

func TestProjects(t *testing.T) {
	h := newHarness(t)
	root := t.TempDir()
	other, clash := filepath.Join(root, "a", "other"), filepath.Join(root, "b", "other")
	for _, d := range []string{other, clash} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	snap, err := h.app.AddProjectDir(other)
	if err != nil || snap.Project.Name != "other" || snap.Project.Dir != other {
		t.Fatalf("AddProjectDir = %+v, %v", snap.Project, err)
	}
	if again, err := h.app.AddProjectDir(other); err != nil || len(again.Projects) != 2 {
		t.Errorf("adding the same folder twice: %d projects, %v", len(again.Projects), err)
	}
	if snap, err = h.app.AddProjectDir(clash); err != nil || snap.Project.Name != "other-2" {
		t.Errorf("name clash gave %+v, %v", snap.Project, err)
	}
	if _, err := h.app.AddProjectDir(filepath.Join(root, "missing")); err == nil {
		t.Error("a folder that does not exist was accepted")
	}

	cfg, err := project.Load(h.conf)
	if err != nil || cfg.Agent != "bash" || len(cfg.Projects) != 3 || cfg.Projects[0].Commands["ready"] != "/ship {n}" {
		t.Fatalf("saved config = %+v, %v", cfg, err)
	}
	if got := readLastProject(h.state); got != "other-2" {
		t.Errorf("last project = %q", got)
	}

	t.Run("picker", func(t *testing.T) {
		before := h.app.Snapshot()
		if snap, err := h.app.AddProject(); err != nil || snap.Project.Name != before.Project.Name {
			t.Errorf("a cancelled picker changed the project: %+v, %v", snap.Project, err)
		}
		picked := filepath.Join(root, "c", "picked")
		if err := os.MkdirAll(picked, 0o700); err != nil {
			t.Fatal(err)
		}
		h.rec.mu.Lock()
		h.rec.pick = picked
		h.rec.mu.Unlock()
		if snap, err := h.app.AddProject(); err != nil || snap.Project.Name != "picked" {
			t.Errorf("picked project = %+v, %v", snap.Project, err)
		}
	})

	t.Run("counts", func(t *testing.T) {
		if _, err := h.app.SwitchProject("main"); err != nil {
			t.Fatal(err)
		}
		a, err := h.app.NewSession("a", "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := h.app.SwitchProject("other"); err != nil {
			t.Fatal(err)
		}
		b, err := h.app.NewSession("b", "")
		if err != nil {
			t.Fatal(err)
		}
		h.hook(t, a.ID, "Notification", `{"message":"needs you"}`)
		h.hook(t, b.ID, "PreToolUse", `{"tool_name":"Edit"}`)
		counts := func() string {
			var out []string
			list, _ := h.rec.last("projects").([]Project)
			for _, p := range list {
				if p.Sessions > 0 {
					out = append(out, fmt.Sprintf("%s:%d/%d/%d", p.Name, p.NeedsYou, p.Working, p.Sessions))
				}
			}
			return strings.Join(out, " ")
		}
		eventually(t, "counts across projects", func() bool { return counts() == "main:1/0/1 other:0/1/1" })
		var viaSnapshot []string
		for _, p := range h.app.Snapshot().Projects {
			if p.Sessions > 0 {
				viaSnapshot = append(viaSnapshot, fmt.Sprintf("%s:%d/%d/%d", p.Name, p.NeedsYou, p.Working, p.Sessions))
			}
		}
		if got := strings.Join(viaSnapshot, " "); got != "main:1/0/1 other:0/1/1" {
			t.Errorf("snapshot counts = %s", got)
		}
	})

	t.Run("last project at start", func(t *testing.T) {
		if _, err := h.app.SwitchProject("other-2"); err != nil {
			t.Fatal(err)
		}
		t.Setenv("AGENTOS_DIR", "/")
		cfg, err := loadConfig()
		if err != nil || cfg.project.Name != "other-2" {
			t.Errorf("from Finder: %+v, %v", cfg.project, err)
		}
		t.Setenv("AGENTOS_DIR", h.dir)
		if cfg, _ = loadConfig(); cfg.project.Name != "main" {
			t.Errorf("inside a known folder: %q", cfg.project.Name)
		}
		t.Setenv("AGENTOS_DIR", "/")
		if err := writeLastProject(h.state, "gone"); err != nil {
			t.Fatal(err)
		}
		if cfg, _ = loadConfig(); cfg.project.Name != "main" {
			t.Errorf("unknown last project gave %q, want the first one", cfg.project.Name)
		}
	})

	t.Run("remove", func(t *testing.T) {
		if _, err := h.app.SwitchProject("other"); err != nil {
			t.Fatal(err)
		}
		snap, err := h.app.RemoveProject("other")
		if err != nil || snap.Project.Name == "other" {
			t.Fatalf("RemoveProject = %+v, %v", snap.Project, err)
		}
		cfg, _ := project.Load(h.conf)
		for _, p := range cfg.Projects {
			if p.Name == "other" {
				t.Error("the removed project is still in the config")
			}
		}
		listed := false
		for _, p := range snap.Projects {
			listed = listed || (p.Name == "other" && p.Sessions == 1)
		}
		if !listed {
			t.Errorf("a project with a live session vanished: %+v", snap.Projects)
		}
		if _, err := h.app.RemoveProject("nope"); err == nil {
			t.Error("removing an unknown project succeeded")
		}
	})
}

func TestNotes(t *testing.T) {
	h := newHarness(t)
	if _, err := h.app.AddNote("  "); err == nil {
		t.Error("an empty note was accepted")
	}
	first, err := h.app.AddNote("first title\nbody line")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Millisecond)
	second, err := h.app.AddNote("second")
	if err != nil {
		t.Fatal(err)
	}
	ids := func(ns []Note) string {
		var out []string
		for _, n := range ns {
			out = append(out, n.Text)
		}
		return strings.Join(out, "|")
	}
	if got := ids(h.app.Snapshot().Notes); got != "second|first title\nbody line" {
		t.Errorf("order = %q", got)
	}
	if _, err := h.app.SetNoteDone(second.ID, true); err != nil {
		t.Fatal(err)
	}
	if n, err := h.app.UpdateNote(first.ID, "edited"); err != nil || n.Text != "edited" || n.UpdatedAt < first.UpdatedAt {
		t.Errorf("UpdateNote = %+v, %v", n, err)
	}
	if got := ids(h.app.Snapshot().Notes); got != "edited|second" {
		t.Errorf("done notes sort last: %q", got)
	}
	if events, _ := h.rec.last("notes").([]Note); ids(events) != "edited|second" {
		t.Errorf("notes event = %q", ids(events))
	}

	t.Setenv("AGENTOS_DIR", h.dir)
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	fresh := newApp(cfg, host{emit: func(string, any) {}, clipboard: func(string) {}, openURL: func(string) {}, pickDir: h.rec.picker}, h.gh.run)
	if got := ids(fresh.Snapshot().Notes); got != "edited|second" || !fresh.Snapshot().Notes[1].Archived {
		t.Errorf("after reload: %q", got)
	}

	if err := h.app.DeleteNote(second.ID); err != nil {
		t.Fatal(err)
	}
	if err := h.app.DeleteNote(second.ID); err == nil {
		t.Error("deleting a deleted note succeeded")
	}
	if _, err := h.app.UpdateNote("nope", "x"); err == nil {
		t.Error("updating an unknown note succeeded")
	}
	if got := ids(h.app.Snapshot().Notes); got != "edited" {
		t.Errorf("after delete: %q", got)
	}
	if got := h.app.Snapshot().Version; got == "" {
		t.Error("no version in the snapshot")
	}
}

func TestNoteToIssue(t *testing.T) {
	h := newHarness(t)
	h.gh.create = "Creating issue in acme/widgets\n\nhttps://github.com/acme/widgets/issues/55\n"
	if _, err := h.app.Issues(false); err != nil {
		t.Fatal(err)
	}
	n, err := h.app.AddNote("Fix login\n\nIt fails on Safari.\nSecond line.")
	if err != nil {
		t.Fatal(err)
	}
	if n, err = h.app.NoteToIssue(n.ID); err != nil {
		t.Fatal(err)
	}
	if n.Issue != 55 || n.IssueURL != "https://github.com/acme/widgets/issues/55" {
		t.Errorf("note = %+v", n)
	}
	var create string
	for _, c := range h.gh.callLog() {
		if strings.HasPrefix(c, "issue create") {
			create = c
		}
	}
	if want := "issue create --title Fix login --body ## Description\n\nFix login\n\nIt fails on Safari.\nSecond line. --assignee @me"; create != want {
		t.Errorf("gh call = %q, want %q", create, want)
	}
	if h.gh.issueCalls() != 2 {
		t.Errorf("issue list ran %d times, want a refresh", h.gh.issueCalls())
	}
	if h.rec.last("issues") == nil {
		t.Error("no issues event")
	}
	if got, _ := h.rec.last("notes").([]Note); len(got) != 1 || got[0].Issue != 55 {
		t.Errorf("notes event = %+v", got)
	}
	if _, err := h.app.NoteToIssue(n.ID); err == nil {
		t.Error("a note was filed twice")
	}

	h.gh.create = "something odd\n"
	odd, _ := h.app.AddNote("odd")
	if _, err := h.app.NoteToIssue(odd.ID); err == nil {
		t.Error("output without an address was accepted")
	}
	if got, _ := h.app.notes.Get("main", odd.ID); got.Issue != 0 {
		t.Error("a failed filing marked the note")
	}
}

func TestNoteToIssueWithoutRepo(t *testing.T) {
	h := newHarness(t)
	h.gh.repo = errors.New("not a repo")
	n, _ := h.app.AddNote("x")
	if _, err := h.app.NoteToIssue(n.ID); err == nil {
		t.Error("filed an issue without a repo")
	}
}

func TestNoteToSession(t *testing.T) {
	h := newHarness(t)
	n, _ := h.app.AddNote("remember this")
	s, err := h.app.NoteToSession(n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "remember this" {
		t.Errorf("title = %q", s.Title)
	}
	eventually(t, "note text typed", func() bool { return strings.Contains(h.pane(t, s.ID), "remember this") })
}

func TestNoteToSessionKeepsLineBreaksUnsent(t *testing.T) {
	h := newHarness(t)
	n, _ := h.app.AddNote("first line\nsecond line")
	s, err := h.app.NoteToSession(n.ID)
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "both lines typed", func() bool {
		pane := h.pane(t, s.ID)
		return strings.Contains(pane, "first line") && strings.Contains(pane, "second line")
	})
	time.Sleep(300 * time.Millisecond)
	if pane := h.pane(t, s.ID); strings.Contains(pane, "not found") {
		t.Errorf("the note was submitted:\n%s", pane)
	}
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

func TestSessionsSurviveWithoutALocale(t *testing.T) {
	for _, name := range []string{"LANG", "LC_ALL", "LC_CTYPE"} {
		t.Setenv(name, "")
	}
	h := newHarness(t)
	s, err := h.app.NewSession("naïve", "")
	if err != nil {
		t.Fatal(err)
	}
	h.app.sessions.Refresh()
	list := h.app.sessions.List()
	if len(list) != 1 || list[0].ID != s.ID || list[0].Title != "naïve" {
		t.Errorf("sessions after a refresh = %+v", list)
	}
}

func TestDataDirSplit(t *testing.T) {
	root := t.TempDir()
	home, synced := filepath.Join(root, "home"), filepath.Join(root, "cloud", "agentos")
	t.Setenv("HOME", home)
	t.Setenv("AGENTOS_DATA_DIR", "")
	state := filepath.Join(root, "state")
	conf := filepath.Join(root, "config.yaml")
	if err := os.WriteFile(conf, []byte("data_dir: "+synced+"\nagent: bash\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTOS_CONFIG", conf)
	t.Setenv("AGENTOS_STATE_DIR", state)
	t.Setenv("AGENTOS_DIR", root)
	t.Setenv("AGENTOS_TMUX_SOCKET", "aostest-unused")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	local := filepath.Join(home, ".local", "share", "agentos")
	if cfg.dataDir != synced || cfg.localDir != local {
		t.Fatalf("dirs = %q and %q", cfg.dataDir, cfg.localDir)
	}
	app := newApp(cfg, host{emit: func(string, any) {}, clipboard: func(string) {}, openURL: func(string) {}, pickDir: func() (string, error) { return "", nil }}, execRunner)
	if _, err := app.notes.Add("p", "a note"); err != nil {
		t.Fatal(err)
	}
	if err := app.sessions.waits.append("p", Wait{Kind: "finished"}); err != nil {
		t.Fatal(err)
	}
	if err := app.digests.Begin("p", time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := app.evidence.AddText("p/1", "x", "", "agent"); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		filepath.Join(synced, "p", "notes.json"), filepath.Join(synced, "p", "stats.jsonl"), filepath.Join(synced, "p", "digest.json"),
		filepath.Join(local, "p", "evidence", "1", "index.json"),
	} {
		if !exists(path) {
			t.Errorf("%s was not written", path)
		}
	}
	for _, path := range []string{filepath.Join(local, "p", "notes.json"), filepath.Join(synced, "p", "evidence"), filepath.Join(synced, "p", "browser")} {
		if exists(path) {
			t.Errorf("%s should not exist", path)
		}
	}
	if app.browsers.dataDir != local || app.life.dataDir != local {
		t.Error("the browser profile or PR tracking sits outside the local dir")
	}

	t.Setenv("AGENTOS_DATA_DIR", filepath.Join(root, "override"))
	if cfg, _ = loadConfig(); cfg.dataDir != filepath.Join(root, "override") || cfg.localDir != cfg.dataDir {
		t.Errorf("the override gave %q and %q", cfg.dataDir, cfg.localDir)
	}
}

func TestInboxAndIdeaCommands(t *testing.T) {
	h := newHarness(t)
	h.app.sessions.registry.mu.Lock()
	h.app.sessions.registry.cfg.Projects[0].Commands["inbox"] = "/investigate {n}"
	h.app.sessions.registry.mu.Unlock()
	h.app.sessions.mu.Lock()
	h.app.sessions.project = h.app.sessions.registry.cfg.Projects[0]
	h.app.sessions.mu.Unlock()

	s, err := h.app.StartIssue(11) // no labels: the inbox lane
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "/investigate typed", func() bool { return strings.Contains(h.pane(t, s.ID), "/investigate 11") })
	idea, err := h.app.StartIssue(10) // idea lane: no command by default
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(600 * time.Millisecond)
	if pane := h.pane(t, idea.ID); strings.Contains(pane, "/") && strings.Contains(pane, "investigate") {
		t.Errorf("an idea got a command:\n%s", pane)
	}
}

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

func TestIdleReminders(t *testing.T) {
	h := newHarness(t)
	s, err := h.app.NewSession("r", "")
	if err != nil {
		t.Fatal(err)
	}
	state := func() string { got, _ := h.session(s.ID); return string(got.State) }
	waits := func() int { st, _ := h.app.Stats(7); return st.Total }

	h.hook(t, s.ID, "UserPromptSubmit", `{"prompt":"go"}`)
	eventually(t, "working", func() bool { return state() == "working" })
	replied := h.rec.countAttention("replied")
	h.hook(t, s.ID, "Notification", `{"message":"Claude is waiting for your input","notification_type":"idle_prompt"}`)
	eventually(t, "a turn killed by an error to stop showing as working", func() bool { return state() == "idle" })
	if h.rec.countAttention("replied") != replied || waits() != 0 {
		t.Errorf("an idle reminder raised attention (%d) or opened a wait (%d)", h.rec.countAttention("replied")-replied, waits())
	}

	h.hook(t, s.ID, "Notification", `{"message":"needs permission","notification_type":"permission_prompt"}`)
	eventually(t, "waiting", func() bool { return state() == "waiting" })
	h.hook(t, s.ID, "Notification", `{"message":"Claude is waiting for your input","notification_type":"idle_prompt"}`)
	time.Sleep(300 * time.Millisecond)
	if state() != "waiting" {
		t.Errorf("an idle reminder changed a waiting session to %s", state())
	}

	h.hook(t, s.ID, "PostToolUse", `{"tool_name":"Edit"}`)
	eventually(t, "working again", func() bool { return state() == "working" })
	h.hook(t, s.ID, "Stop", `{}`)
	eventually(t, "a wait for the reply", func() bool { return waits() == 2 })
	st, _ := h.app.Stats(7)
	var kinds []string
	for _, c := range st.ByCause {
		kinds = append(kinds, c.Kind+"/"+c.Label)
	}
	slices.Sort(kinds)
	if !slices.Equal(kinds, []string{"idle/Reply landed", "permission/Permission"}) {
		t.Errorf("causes = %q", kinds)
	}
}

func TestEndedSessionsSurviveARestart(t *testing.T) {
	h := newHarness(t)
	s, err := h.app.NewSession("keep me", "")
	if err != nil {
		t.Fatal(err)
	}
	h.hook(t, s.ID, "SessionEnd", `{}`)
	eventually(t, "ended by its hook", func() bool { got, _ := h.session(s.ID); return got.State == "ended" })
	if err := h.app.KillSession(s.ID); err != nil && !strings.Contains(err.Error(), "ended") {
		t.Fatal(err)
	}
	eventually(t, "ended after the process is gone", func() bool { got, ok := h.session(s.ID); return ok && got.State == "ended" && got.EndedAt > 0 })

	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	fresh := newApp(cfg, host{emit: func(string, any) {}, clipboard: func(string) {}, openURL: func(string) {}, pickDir: h.rec.picker}, h.gh.run)
	fresh.digestFirst = time.Hour
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := fresh.start(ctx); err != nil {
		t.Fatal(err)
	}
	var got Session
	for _, v := range fresh.sessions.List() {
		if v.ID == s.ID {
			got = v
		}
	}
	if got.State != "ended" || got.Title != "keep me" || got.EndedAt == 0 {
		t.Errorf("after a restart = %+v", got)
	}
	if err := fresh.DismissSession(s.ID); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(h.state, "sessions", s.ID+".json")) {
		t.Error("the state file of a dismissed session is still there")
	}
}

func TestOldFinishedStateIsMigrated(t *testing.T) {
	h := newHarness(t)
	s, err := h.app.NewSession("old", "")
	if err != nil {
		t.Fatal(err)
	}
	old := `{"session":"` + s.ID + `","state":"finished","event":"Stop","at":"2026-10-01T12:00:00Z"}`
	file := filepath.Join(h.state, "sessions", s.ID+".json")
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	bus.MigrateStates(h.state)
	rec, err := bus.ReadState(h.state, s.ID)
	if err != nil || rec.State != session.Idle {
		t.Errorf("state = %q, %v", rec.State, err)
	}
	if data, _ := os.ReadFile(file); !strings.Contains(string(data), `"state":"idle"`) {
		t.Error("the file was not rewritten")
	}
}
