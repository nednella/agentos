package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/session"
	"github.com/nednella/agentos/internal/term"
)

// killSetup points the command at a private tmux socket, state dir and config,
// starts three sessions in two projects, and works from the first project's folder.
func killSetup(t *testing.T) (*term.Tmux, string) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux is not installed")
	}
	state, err := os.MkdirTemp("", "aoskill")
	if err != nil {
		t.Fatal(err)
	}
	socket := fmt.Sprintf("aostest-kill-%d", os.Getpid())
	t.Cleanup(func() {
		_ = exec.Command("tmux", "-L", socket, "kill-server").Run()
		os.RemoveAll(state)
	})
	demo, other := t.TempDir(), t.TempDir()
	config := filepath.Join(state, "config.yaml")
	body := fmt.Sprintf("projects:\n  - {name: demo, dir: %s}\n  - {name: other, dir: %s}\n", demo, other)
	if err := os.WriteFile(config, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTOS_TMUX_SOCKET", socket)
	t.Setenv("AGENTOS_STATE_DIR", state)
	t.Setenv("AGENTOS_CONFIG", config)

	tmux, err := term.NewTmux(socket, state)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []struct {
		name  session.Name
		title string
		dir   string
	}{{session.Name{Project: "demo", N: 1}, "one", demo}, {session.Name{Project: "demo", N: 2}, "two", demo}, {session.Name{Project: "other", N: 1}, "three", other}} {
		if err := tmux.NewSession(context.Background(), s.name, s.title, s.dir, nil, []string{"sleep", "60"}, 80, 24); err != nil {
			t.Fatal(err)
		}
		if err := bus.WriteState(state, session.Record{Session: s.name.String(), State: session.Idle}); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(demo)
	return tmux, state
}

func runKill(t *testing.T, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetArgs(append([]string{"kill"}, args...))
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func names(t *testing.T, tmux *term.Tmux) string {
	t.Helper()
	infos, _, err := tmux.ListAll(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, in := range infos {
		out = append(out, in.Name.String())
	}
	return strings.Join(out, " ")
}

func TestKillStopsOnlyThisProject(t *testing.T) {
	tmux, state := killSetup(t)
	out := runKill(t)
	if !strings.Contains(out, "stopping demo/1 (one)") || !strings.Contains(out, "stopping demo/2 (two)") || strings.Contains(out, "other/1") {
		t.Errorf("output = %q", out)
	}
	if got := names(t, tmux); got != "other/1" {
		t.Errorf("sessions left: %q", got)
	}
	if left := bus.ReadAll(state); len(left) != 1 || left["other/1"].Session == "" {
		t.Errorf("state files left: %v", left)
	}
	if out := runKill(t); !strings.Contains(out, "no agents to stop") {
		t.Errorf("a second kill printed %q", out)
	}
}

func TestKillAllStopsEveryProject(t *testing.T) {
	tmux, state := killSetup(t)
	runKill(t, "--all")
	if got := names(t, tmux); got != "" {
		t.Errorf("sessions left: %q", got)
	}
	if left := bus.ReadAll(state); len(left) != 0 {
		t.Errorf("state files left: %v", left)
	}
}

func TestKillLeavesShellsUnlessAll(t *testing.T) {
	tmux, _ := killSetup(t)
	shell := session.Name{Project: "demo"}
	if err := tmux.NewSession(context.Background(), shell, "shell", t.TempDir(), nil, []string{"sleep", "60"}, 80, 24); err != nil {
		t.Fatal(err)
	}
	if out := runKill(t); strings.Contains(out, "shell") || !tmux.Has(context.Background(), shell) {
		t.Errorf("kill touched the shell: %q", out)
	}
	if out := runKill(t, "--all"); !strings.Contains(out, "stopping demo/shell") || tmux.Has(context.Background(), shell) {
		t.Errorf("kill --all left the shell: %q", out)
	}
}
