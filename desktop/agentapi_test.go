package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/session"
)

type digestCall struct {
	dir, name string
	env       []string
	args      []string
}

type fakeClaude struct {
	mu    sync.Mutex
	calls []digestCall
	run   func(call digestCall)
	err   error
	block chan struct{}
}

func (f *fakeClaude) runner(ctx context.Context, dir string, env []string, name string, args ...string) ([]byte, error) {
	call := digestCall{dir: dir, name: name, env: env, args: args}
	f.mu.Lock()
	f.calls = append(f.calls, call)
	f.mu.Unlock()
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.run != nil {
		f.run(call)
	}
	return nil, f.err
}

func (f *fakeClaude) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func envValue(env []string, key string) (string, bool) {
	for _, kv := range slices.Backward(env) {
		if v, ok := strings.CutPrefix(kv, key+"="); ok {
			return v, true
		}
	}
	return "", false
}

func (h *harness) lastDigest() Digest {
	d, _ := h.rec.last("digest").(Digest)
	return d
}

func TestDigestRun(t *testing.T) {
	claude := &fakeClaude{}
	h := newHarnessWith(t, func(a *App) { a.runEnv = claude.runner })
	t.Setenv("AGENTOS_SESSION", "main/9") // a digest run must not inherit a session
	claude.run = func(call digestCall) {
		// What claude would do: run the CLI the prompt names, as a child of the run.
		sess, _ := envValue(call.env, "AGENTOS_SESSION")
		if sess != "" {
			t.Errorf("AGENTOS_SESSION leaked into the digest run: %q", sess)
		}
		for i, item := range [][4]string{
			{"Wails 2.15 released", "you build on Wails", "https://example.com/wails", "wails"},
			{"Duplicate", "same link", "https://example.com/wails", "wails"},
			{"Claude Code hooks", "the guard uses hooks", "https://example.com/hooks", "Claude Code"},
		} {
			cmd := []string{"digest", "add", "--title", item[0], "--why", item[1], "--url", item[2], "--source", item[3]}
			out, err := h.cliEnv(t, call.env, cmd...)
			if err != nil {
				t.Errorf("digest add %d: %v", i, err)
			}
			if i == 1 && out != "already in the digest" {
				t.Errorf("a duplicate printed %q", out)
			}
		}
	}

	if err := h.app.RunDigest(); err != nil {
		t.Fatal(err)
	}
	if err := h.app.RunDigest(); err == nil {
		t.Error("a second run started while one was going")
	}
	eventually(t, "the digest to finish", func() bool { d := h.lastDigest(); return !d.Running && d.LastRunAt > 0 })

	call := claude.calls[0]
	if call.name != "claude" || call.dir != h.dir || len(call.args) != 4 || call.args[0] != "-p" || call.args[2] != "--allowedTools" ||
		call.args[3] != "WebSearch WebFetch Read Glob Grep Bash(agentos digest add:*)" {
		t.Errorf("command = %s %q in %s", call.name, call.args, call.dir)
	}
	if !strings.Contains(call.args[1], "agentos digest add --title") || !strings.Contains(call.args[1], "at most five") {
		t.Errorf("prompt = %q", call.args[1])
	}
	if key, _ := envValue(call.env, "AGENTOS_DIGEST_PROJECT"); key != "main" {
		t.Errorf("AGENTOS_DIGEST_PROJECT = %q", key)
	}

	d := h.app.Digest()
	if len(d.Items) != 2 || d.Items[0].Title != "Wails 2.15 released" || d.Items[1].Source != "Claude Code" || d.Error != "" || d.NextRunAt <= d.LastRunAt {
		t.Fatalf("digest = %+v", d)
	}
	note, err := h.app.DigestToNote(d.Items[0].ID)
	if err != nil || !strings.HasPrefix(note.Text, "Wails 2.15 released\n\nyou build on Wails\nhttps://example.com/wails") {
		t.Fatalf("DigestToNote = %+v, %v", note, err)
	}
	if d = h.app.Digest(); d.Items[0].NoteID != note.ID {
		t.Errorf("the item does not know its note: %+v", d.Items[0])
	}
	if err := h.app.DismissDigestItem(d.Items[1].ID); err != nil || len(h.app.Digest().Items) != 1 {
		t.Errorf("dismiss: %v", err)
	}
	if err := h.app.DismissDigestItem("nope"); err == nil {
		t.Error("dismissed an unknown item")
	}

	if out, err := h.cliEnv(t, os.Environ(), "digest", "add", "--title", "x", "--url", "u"); err == nil {
		t.Errorf("digest add outside a run succeeded: %q", out)
	}
	h.app.digests.mu.Lock()
	h.app.digests.running["main"] = 0
	h.app.digests.mu.Unlock()
	for i := range 7 {
		_, err := h.app.digests.Add("main", DigestItem{Title: "t", URL: "https://example.com/" + string(rune('a'+i))})
		if (err != nil) != (i >= digestPerRun) {
			t.Errorf("item %d: error %v", i, err)
		}
	}
}

func TestDigestFailures(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"claude missing", fmt.Errorf("claude: %w", exec.ErrNotFound), "claude was not found"},
		{"claude fails", context.Canceled, "context canceled"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claude := &fakeClaude{err: tt.err}
			h := newHarnessWith(t, func(a *App) { a.runEnv = claude.runner })
			if err := h.app.RunDigest(); err != nil {
				t.Fatal(err)
			}
			eventually(t, "the failure", func() bool { d := h.lastDigest(); return !d.Running && d.Error != "" })
			if d := h.lastDigest(); !strings.Contains(d.Error, tt.want) || d.LastRunAt != 0 {
				t.Errorf("digest = %+v", d)
			}
		})
	}
}

func TestDigestAutomaticRuns(t *testing.T) {
	claude := &fakeClaude{}
	h := newHarnessWith(t, func(a *App) {
		a.runEnv = claude.runner
		a.digestFirst, a.digestTick = 50*time.Millisecond, 50*time.Millisecond
	})

	eventually(t, "the automatic run", func() bool { return claude.callCount() == 1 && !h.app.Digest().Running })
	time.Sleep(300 * time.Millisecond)
	if claude.callCount() != 1 {
		t.Errorf("ran %d times; a run in the last week should stop the next", claude.callCount())
	}
	if h.app.Digest().NextRunAt == 0 {
		t.Error("no next run time")
	}

	t.Run("off and stale", func(t *testing.T) {
		h.app.sessions.mu.Lock()
		h.app.sessions.project.Digest = "off"
		h.app.sessions.mu.Unlock()
		h.app.digests.mu.Lock()
		f := h.app.digests.read("main")
		f.LastRunAt = time.Now().Add(-8 * 24 * time.Hour).UnixMilli()
		f.LastAttemptAt = f.LastRunAt
		_ = h.app.digests.write("main", f)
		h.app.digests.mu.Unlock()
		time.Sleep(300 * time.Millisecond)
		if claude.callCount() != 1 {
			t.Error("ran although the digest is off")
		}
		if h.app.Digest().NextRunAt != 0 {
			t.Error("a next run is shown although the digest is off")
		}
		h.app.sessions.mu.Lock()
		h.app.sessions.project.Digest = "weekly"
		h.app.sessions.mu.Unlock()
		eventually(t, "a run of a stale digest", func() bool { return claude.callCount() == 2 })
	})
}

func TestDueAfterFailure(t *testing.T) {
	d := newDigests(t.TempDir())
	now := time.Now()
	if !d.Due("p", now) {
		t.Error("a project that never ran is not due")
	}
	if err := d.Begin("p", now); err != nil {
		t.Fatal(err)
	}
	d.Finish("p", now, "boom")
	if d.Due("p", now.Add(time.Hour)) {
		t.Error("a failed run is retried within hours")
	}
	if !d.Due("p", now.Add(7*time.Hour)) {
		t.Error("a failed run is never retried")
	}
}

func (h *harness) cliEnv(t *testing.T, env []string, args ...string) (string, error) {
	t.Helper()
	return h.cliWith(t, env, args...)
}

func TestCLIsAgainstTheApp(t *testing.T) {
	h := newHarness(t)
	s, err := h.app.NewSession("cli", "")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("note", func(t *testing.T) {
		out := h.mustCLI(t, s.ID, "note", "check the", "login page")
		if !strings.Contains(out, "note saved") {
			t.Errorf("note printed %q", out)
		}
		notes := h.app.Snapshot().Notes
		if len(notes) != 1 || notes[0].Text != "check the login page" {
			t.Errorf("notes = %+v", notes)
		}
		if h.rec.last("notes") == nil {
			t.Error("no notes event")
		}
		if _, err := h.cli(t, s.ID, "note", "  "); err == nil {
			t.Error("an empty note was accepted")
		}
	})

	t.Run("stats", func(t *testing.T) {
		h.hook(t, s.ID, "PreToolUse", `{"tool_name":"Bash","tool_input":{"command":"yarn test api"}}`)
		h.hook(t, s.ID, "Notification", `{"message":"needs permission","notification_type":"permission_prompt"}`)
		eventually(t, "a wait", func() bool { st, _ := h.app.Stats(7); return st.Total == 1 })
		out := h.mustCLI(t, s.ID, "stats", "--days", "30")
		if !strings.Contains(out, "over the last 30 days: 1") || !strings.Contains(out, "Bash: yarn test api") {
			t.Errorf("stats = %q", out)
		}
		var st Stats
		if err := json.Unmarshal([]byte(h.mustCLI(t, s.ID, "stats", "--json")), &st); err != nil || st.Days != 7 || st.Total != 1 {
			t.Errorf("stats json = %+v, %v", st, err)
		}
		if _, err := h.cli(t, s.ID, "stats", "--days", "0"); err == nil {
			t.Error("--days 0 was accepted")
		}
	})

	t.Run("show", func(t *testing.T) {
		dir := t.TempDir()
		img := filepath.Join(dir, "shot.png")
		note := filepath.Join(dir, "notes.txt")
		binary := filepath.Join(dir, "blob.bin")
		for path, data := range map[string][]byte{img: onePixelPNG, note: []byte("all green\nno regressions\n"), binary: {0xff, 0xfe, 0x00, 0x01}} {
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		attention := h.rec.count("attention")
		h.mustCLI(t, s.ID, "show", img, "--caption", "pixel")
		h.mustCLI(t, s.ID, "show", note)
		h.mustCLI(t, s.ID, "show", "--text", "ran 40 tests, all pass", "--caption", "tests")
		items := h.app.Evidence(s.ID)
		if len(items) != 3 {
			t.Fatalf("evidence = %+v", items)
		}
		if items[0].Kind != "image" || items[0].Caption != "pixel" || items[0].Source != "agent" || !strings.HasPrefix(items[0].URL, "/media/evidence/main/") {
			t.Errorf("image = %+v", items[0])
		}
		if items[1].Kind != "text" || items[1].Text != "all green\nno regressions\n" || items[1].Caption != "notes.txt" || items[1].URL != "" {
			t.Errorf("text file = %+v", items[1])
		}
		if items[2].Text != "ran 40 tests, all pass" || items[2].Caption != "tests" {
			t.Errorf("text card = %+v", items[2])
		}
		if got := h.rec.count("attention"); got != attention+3 {
			t.Errorf("attention events = %d, want %d", got, attention+3)
		}
		for _, bad := range [][]string{{"show", binary}, {"show", filepath.Join(dir, "missing.png")}, {"show"}, {"show", img, "--text", "x"}} {
			if out, err := h.cli(t, s.ID, bad...); err == nil {
				t.Errorf("agentos %v succeeded: %q", bad, out)
			}
		}
		if _, err := h.cli(t, "", "show", "--text", "x"); err == nil {
			t.Error("show without a session succeeded")
		}
		if got, _ := h.session(s.ID); got.Evidence != 3 {
			t.Errorf("session evidence = %d", got.Evidence)
		}
		if err := h.app.KillSession(s.ID); err != nil {
			t.Fatal(err)
		}
		if len(h.app.Evidence(s.ID)) != 3 {
			t.Error("evidence went with the ended session; it should stay until dismissed")
		}
		eventually(t, "the session to show as ended", func() bool { got, _ := h.session(s.ID); return got.State == "ended" })
		if err := h.app.DismissSession(s.ID); err != nil {
			t.Fatal(err)
		}
		eventually(t, "evidence removed with the dismissed session", func() bool { return len(h.app.Evidence(s.ID)) == 0 })
	})

	t.Run("app not running", func(t *testing.T) {
		if err := os.Remove(filepath.Join(h.state, "control.sock")); err != nil {
			t.Fatal(err)
		}
		if _, err := h.cli(t, s.ID, "note", "x"); err == nil || !strings.Contains(err.Error(), "not running") {
			t.Errorf("error = %v", err)
		}
	})
}

func TestHarnessCheck(t *testing.T) {
	h := newHarness(t)
	h.app.sessions.prefillFor = 200 * time.Millisecond
	s, err := h.app.HarnessCheck()
	if err != nil || s.Title != "Harness check" {
		t.Fatalf("HarnessCheck = %+v, %v", s, err)
	}
	eventually(t, "the prompt typed in", func() bool {
		pane := h.pane(t, s.ID)
		return strings.Contains(pane, "Review this project") && strings.Contains(pane, "agentos note")
	})
	time.Sleep(300 * time.Millisecond)
	if strings.Contains(h.pane(t, s.ID), "not found") {
		t.Error("the prompt was submitted")
	}
	for _, want := range []string{"agentos stats --days 30", "claude --help", "https://docs.claude.com/en/docs/claude-code", "at most seven", "agentos note", "Change no files"} {
		if !strings.Contains(harnessPrompt, want) {
			t.Errorf("the prompt lacks %q", want)
		}
	}
}

func TestClaudeGetsTheBrowserPrompt(t *testing.T) {
	off := false
	tests := []struct {
		name    string
		agent   agent.Agent
		proj    project.Project
		browser bool
		want    bool
	}{
		{"claude with a browser", agent.Claude{Exe: "/x"}, project.Project{Name: "p"}, true, true},
		{"project turns it off", agent.Claude{Exe: "/x"}, project.Project{Name: "p", Browser: &off}, true, false},
		{"no browser installed", agent.Claude{Exe: "/x"}, project.Project{Name: "p"}, false, false},
		{"plain agent", agent.Plain{Argv: []string{"claude"}}, project.Project{Name: "p"}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sessions{agent: tt.agent, browsers: &Browsers{}}
			if tt.browser {
				s.browsers.binary = "/bin/true"
			}
			argv := s.commandFor(session.Name{Project: "p", N: 1}, tt.proj)
			i := slices.Index(argv, "--append-system-prompt")
			if (i >= 0) != tt.want {
				t.Fatalf("argv = %q", argv)
			}
			if tt.want && (argv[i+1] != agent.BrowserPrompt || !strings.Contains(argv[i+1], "agentos browser help") || !strings.Contains(argv[i+1], "agentos show")) {
				t.Errorf("prompt = %q", argv[i+1])
			}
		})
	}
}

func TestNoBrowserFound(t *testing.T) {
	b := newBrowsers(t.TempDir(), func(string, any) {})
	b.binary = ""
	if b.Available() {
		t.Error("available without a browser")
	}
	st, err := b.Open(context.Background(), "p/1", "")
	if err == nil || !strings.Contains(st.Error, "no Brave") {
		t.Errorf("Open = %+v, %v", st, err)
	}
	if s := b.State("p/1"); s.Open || s.ID != "p/1" {
		t.Errorf("state = %+v", s)
	}
}

func TestWebAddress(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"example.com", "https://example.com", true},
		{"example.com/a?b=1", "https://example.com/a?b=1", true},
		{"https://example.com", "https://example.com", true},
		{"localhost:3000", "http://localhost:3000", true},
		{"127.0.0.1:8080/x", "http://127.0.0.1:8080/x", true},
		{"about:blank", "about:blank", true},
		{"  spaced.dev  ", "https://spaced.dev", true},
		{"javascript:alert(1)", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, err := webAddress(tt.in)
		if (err == nil) != tt.ok || got != tt.want {
			t.Errorf("webAddress(%q) = %q, %v", tt.in, got, err)
		}
	}
}

func TestEvidenceStore(t *testing.T) {
	e := newEvidenceStore(t.TempDir())
	id := "main/3"
	img, err := e.AddImage(id, onePixelPNG, "c", "user")
	if err != nil || !strings.HasPrefix(img.URL, "/media/evidence/main/3/") {
		t.Fatalf("AddImage = %+v, %v", img, err)
	}
	if _, err := e.AddImage(id, []byte("<svg/>"), "", "user"); err == nil {
		t.Error("a non-image was filed as an image")
	}
	if _, err := e.AddText(id, "words", "", "agent"); err != nil {
		t.Fatal(err)
	}
	again := newEvidenceStore(e.dir)
	if got := again.List(id); len(got) != 2 || got[0].ID != img.ID || got[1].Text != "words" {
		t.Errorf("after reload: %+v", got)
	}
	if got := again.List("main/4"); len(got) != 0 {
		t.Errorf("another session's evidence: %+v", got)
	}
	if err := again.Delete(id, img.ID); err != nil || again.Count(id) != 1 || exists(mediaFile(map[string]string{evidenceFolder: e.dir}, img.URL)) {
		t.Errorf("Delete: %v", err)
	}
	if n := again.Purge(id); n != 1 || again.Count(id) != 0 || exists(filepath.Join(e.dir, "evidence", "main", "3")) {
		t.Errorf("Purge removed %d", n)
	}
	if got := again.List("not a session"); len(got) != 0 {
		t.Errorf("a bad id gave %+v", got)
	}
}
