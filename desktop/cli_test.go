package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func (r *recorder) uiCommands() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, e := range r.events {
		if m, ok := e.payload.(map[string]any); ok && e.name == "ui:command" {
			out = append(out, fmt.Sprintf("%s %s", m["name"], strings.Join(m["args"].([]string), " ")))
		}
	}
	return out
}

func cliHarness(t *testing.T) *harness {
	t.Helper()
	claude := &fakeClaude{}
	return newHarnessWith(t, func(a *App) { a.runEnv = claude.runner })
}

func TestShellSession(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	h := newHarness(t)
	if id := h.app.Snapshot().Shell; id != "" {
		t.Fatalf("a shell before it was opened: %q", id)
	}
	shell, err := h.app.ShellOpen()
	if err != nil || shell.ID != "main/shell" {
		t.Fatalf("ShellOpen = %+v, %v", shell, err)
	}
	if again, err := h.app.ShellOpen(); err != nil || again.ID != shell.ID {
		t.Errorf("a second ShellOpen = %+v, %v", again, err)
	}
	if h.app.Snapshot().Shell != shell.ID {
		t.Error("Snapshot has no shell")
	}

	// It is not a session row.
	agent, err := h.app.NewSession("agent", "")
	if err != nil {
		t.Fatal(err)
	}
	h.app.sessions.Refresh()
	if list := h.app.sessions.List(); len(list) != 1 || list[0].ID != agent.ID {
		t.Errorf("sessions = %+v", list)
	}
	if got := h.app.Snapshot().Projects[0].Sessions; got != 1 {
		t.Errorf("the project counts %d sessions", got)
	}
	if err := h.app.KillSession(shell.ID); err == nil {
		t.Error("the shell was killed as a session")
	}
	if err := h.app.RenameSession(shell.ID, "x"); err == nil {
		t.Error("the shell was renamed as a session")
	}

	// It runs in the project, with the project in its environment, and reports nothing.
	if err := h.app.TermOpen(shell.ID, 100, 30); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the shell's prompt", func() bool { return len(h.rec.output(shell.ID)) > 0 })
	if err := h.app.TermWrite(shell.ID, "echo p=$AGENTOS_PROJECT s=[$AGENTOS_SESSION] d=$(pwd -P)\n"); err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(h.dir)
	eventually(t, "the shell's environment", func() bool {
		return strings.Contains(h.rec.output(shell.ID), "p=main s=[] d="+real)
	})
	if _, err := os.Stat(filepath.Join(h.state, "sessions", "main", "shell.json")); err == nil {
		t.Error("the shell left a state file")
	}

	t.Run("survives a restart", func(t *testing.T) {
		cfg, err := loadConfig()
		if err != nil {
			t.Fatal(err)
		}
		fresh := newApp(cfg, host{emit: func(string, any) {}, clipboard: func(string) {}, openURL: func(string) {}, pickDir: h.rec.picker}, h.gh.run)
		fresh.digestFirst = time.Hour
		fresh.sessions.Refresh()
		if fresh.Snapshot().Shell != shell.ID {
			t.Error("the restarted app does not know the shell")
		}
	})

	t.Run("agentos kill leaves it alone unless --all", func(t *testing.T) {
		killer := func(args ...string) {
			cmd := exec.Command(cliPath(t), args...)
			cmd.Dir = h.dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("agentos %v: %v: %s", args, err, out)
			}
		}
		has := func(id string) bool {
			return exec.Command("tmux", "-L", h.socket, "has-session", "-t", "="+id).Run() == nil
		}
		killer("kill")
		if has(agent.ID) || !has(shell.ID) {
			t.Errorf("after kill: agent %v, shell %v", has(agent.ID), has(shell.ID))
		}
		killer("kill", "--all")
		if has(shell.ID) {
			t.Error("kill --all left the shell")
		}
	})
}

func TestCLICommands(t *testing.T) {
	h := cliHarness(t)
	// A session to talk about, and the project's issues from the fake gh.
	first, err := h.app.NewSession("first", "")
	if err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) (string, error) { return h.cli(t, first.ID, args...) }
	must := func(args ...string) string {
		t.Helper()
		out, err := run(args...)
		if err != nil {
			t.Fatalf("agentos %s: %v", strings.Join(args, " "), err)
		}
		return out
	}

	t.Run("issue", func(t *testing.T) {
		if out := must("issue", "7", "8"); out != "started 2 sessions: #7 (2), #8 (3)" {
			t.Errorf("issue 7 8 = %q", out)
		}
		if out := must("issue", "7"); out != "already running: #7 (2)" {
			t.Errorf("a second issue 7 = %q", out)
		}
		if out, err := run("issue", "999"); err == nil || !strings.Contains(err.Error(), "failed: #999") {
			t.Errorf("issue 999 = %q, %v", out, err)
		}
		if out := must("issue", "7", "999"); !strings.Contains(out, "already running: #7 (2)") || !strings.Contains(out, "failed: #999") {
			t.Errorf("a mixed issue = %q", out)
		}
		if _, err := run("issue"); err == nil {
			t.Error("issue without a number succeeded")
		}
	})

	t.Run("new and kill", func(t *testing.T) {
		if out := must("new", "spike", "it"); out != "started session 4: spike it" {
			t.Errorf("new = %q", out)
		}
		if out := must("kill", "4"); out != "killed session 4: spike it" {
			t.Errorf("kill = %q", out)
		}
		eventually(t, "the killed session to end", func() bool {
			s := h.rec.lastSessions()
			return slices.ContainsFunc(s, func(v Session) bool { return v.N == 4 && v.State == "ended" })
		})
		for _, bad := range []string{"4", "99", "x"} {
			if _, err := run("kill", bad); err == nil {
				t.Errorf("kill %s succeeded", bad)
			}
		}
	})

	t.Run("view commands emit ui:command", func(t *testing.T) {
		tests := []struct {
			args []string
			want string
		}{
			{[]string{"queue"}, "queue "},
			{[]string{"notes"}, "notes "},
			{[]string{"evidence"}, "evidence "},
			{[]string{"term"}, "term "},
			{[]string{"next"}, "next "},
			{[]string{"filter", "@me", "type:bug"}, "filter @me type:bug"},
			{[]string{"filter"}, "filter "},
			{[]string{"browser"}, "browser "},
			{[]string{"stats", "--open"}, "stats "},
			{[]string{"digest"}, "digest "},
			{[]string{"open", "1"}, "open 1"},
			{[]string{"open", "zzz-nothing"}, ""},
		}
		for _, tt := range tests {
			before := len(h.rec.uiCommands())
			out, err := run(tt.args...)
			if tt.want == "" {
				if err == nil {
					t.Errorf("agentos %v succeeded: %q", tt.args, out)
				}
				continue
			}
			got := h.rec.uiCommands()
			if err != nil || out != "ok" || len(got) != before+1 || got[len(got)-1] != tt.want {
				t.Errorf("agentos %v = %q, %v; events %q", tt.args, out, err, got[before:])
			}
		}
		if _, err := run("open", "99"); err == nil {
			t.Error("open of a missing session succeeded")
		}
	})

	t.Run("open by title", func(t *testing.T) {
		if _, err := h.app.NewSession("unique-title", ""); err != nil {
			t.Fatal(err)
		}
		before := len(h.rec.uiCommands())
		if out := must("open", "unique"); out != "ok" || h.rec.uiCommands()[before] != "open unique" {
			t.Errorf("open by title = %q", out)
		}
	})

	t.Run("refresh, pr, cleanup, harness, digest", func(t *testing.T) {
		if out := must("refresh"); !strings.HasPrefix(out, "refreshed 6 issues") {
			t.Errorf("refresh = %q", out)
		}
		if out := must("pr"); out != "no pull requests" {
			t.Errorf("pr = %q", out)
		}
		h.gh.setPR(prJSON("OPEN", true, `[{"status":"COMPLETED","conclusion":"FAILURE"}]`, 0, 0))
		h.app.RefreshPRs()
		if out := must("pr", "2"); !strings.Contains(out, "2: #12 draft, checks failing, 0 comments") {
			t.Errorf("pr 2 = %q", out)
		}
		if out := must("pr"); !strings.Contains(out, "#12 draft") {
			t.Errorf("pr = %q", out)
		}
		if out := must("cleanup"); out != "nothing to clean up" {
			t.Errorf("cleanup = %q", out)
		}
		if out := must("harness"); !strings.HasPrefix(out, "started session ") || !strings.HasSuffix(out, ": Harness check") {
			t.Errorf("harness = %q", out)
		}
		if out := must("digest", "--run"); out != "digest started" {
			t.Errorf("digest --run = %q", out)
		}
		eventually(t, "the digest run to end", func() bool { return !h.app.Digest().Running })
	})

	t.Run("projects", func(t *testing.T) {
		other := filepath.Join(t.TempDir(), "other")
		if err := os.MkdirAll(other, 0o700); err != nil {
			t.Fatal(err)
		}
		if out := must("project"); !strings.Contains(out, "* main") {
			t.Errorf("project = %q", out)
		}
		cmd := exec.Command(cliPath(t), "project", "add", other)
		cmd.Env = append(os.Environ(), "AGENTOS_SESSION="+first.ID, "AGENTOS_SOCKET="+filepath.Join(h.state, "agentos.sock"))
		if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "added project other") {
			t.Fatalf("project add = %q, %v", out, err)
		}
		if out := must("project", "main"); out != "project main" {
			t.Errorf("project main = %q", out)
		}

		// A command from the other project's shell works on that project.
		out, err := h.cliWith(t, append(os.Environ(), "AGENTOS_PROJECT=other"), "new", "there")
		if err != nil || out != "started session 1: there" {
			t.Fatalf("new from another project = %q, %v", out, err)
		}
		if cur := h.app.sessions.Current().Name; cur != "other" {
			t.Errorf("the app stayed on %q", cur)
		}
		if _, err := h.cliWith(t, append(os.Environ(), "AGENTOS_PROJECT=other"), "project", "remove", "other"); err != nil {
			t.Error(err)
		}
		if _, err := run("project", "nope"); err == nil {
			t.Error("a switch to an unknown project succeeded")
		}
	})

	t.Run("the app is not running", func(t *testing.T) {
		if err := os.Remove(filepath.Join(h.state, "control.sock")); err != nil {
			t.Fatal(err)
		}
		for _, args := range [][]string{{"queue"}, {"issue", "7"}, {"kill", "2"}, {"new"}} {
			if out, err := run(args...); err == nil || !strings.Contains(err.Error(), "agentos is not running") {
				t.Errorf("agentos %v = %q, %v", args, out, err)
			}
		}
	})
}

func TestHelpListsEverything(t *testing.T) {
	cmd := exec.Command(cliPath(t), "help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Work:", "Views:", "Projects:", "From inside a session:", "issue", "open", "new", "kill", "project", "next", "filter",
		"queue", "notes", "stats", "digest", "evidence", "browser", "term", "refresh", "pr", "cleanup", "harness", "show", "note"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("help lacks %q", want)
		}
	}
}
