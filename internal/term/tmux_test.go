package term

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/nednella/agentos/internal/session"
)

// newTestTmux starts nothing: it only points at a private socket, which the
// cleanup stops. A test must never touch the default socket.
func newTestTmux(t *testing.T) *Tmux {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux is not installed")
	}
	dir, err := os.MkdirTemp("", "aosterm")
	if err != nil {
		t.Fatal(err)
	}
	socket := fmt.Sprintf("aostest-term-%d", os.Getpid())
	tmux, err := NewTmux(socket, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = exec.Command("tmux", "-L", socket, "kill-server").Run()
		os.RemoveAll(dir)
	})
	return tmux
}

func TestSessionLifecycle(t *testing.T) {
	tmux := newTestTmux(t)
	ctx := context.Background()

	if infos, err := tmux.List(ctx); err != nil || len(infos) != 0 {
		t.Fatalf("List without a server = %v, %v", infos, err)
	}

	name := session.Name{Project: "demo", N: 1}
	dir := t.TempDir()
	if err := tmux.NewSession(ctx, name, "first agent", dir, []string{"AGENTOS_SESSION=demo/1"}, []string{"sleep", "60"}, 100, 30); err != nil {
		t.Fatal(err)
	}
	other := session.Name{Project: "demo", N: 2}
	if err := tmux.NewSession(ctx, other, "tabs\tand ünïcode", dir, nil, []string{"sleep", "60"}, 100, 30); err != nil {
		t.Fatal(err)
	}

	if err := tmux.SetIssue(ctx, name, 394); err != nil {
		t.Fatal(err)
	}
	infos, err := tmux.List(ctx)
	if err != nil || len(infos) != 2 {
		t.Fatalf("List = %v, %v", infos, err)
	}
	byName := map[session.Name]Info{}
	for _, in := range infos {
		byName[in.Name] = in
	}
	if got := byName[other].Title; got != "tabs and ünïcode" {
		t.Errorf("second title = %q", got)
	}
	if byName[name].Issue != "394" || byName[other].Issue != "" {
		t.Errorf("issues = %q and %q", byName[name].Issue, byName[other].Issue)
	}
	if got := byName[name]; got.Title != "first agent" || !strings.HasSuffix(got.Path, dir[strings.LastIndex(dir, "/"):]) || got.Created.IsZero() {
		t.Errorf("first session = %+v", got)
	}

	if err := tmux.Rename(ctx, name, "renamed"); err != nil {
		t.Fatal(err)
	}
	if infos, _ = tmux.List(ctx); !strings.Contains(fmt.Sprint(infos), "renamed") {
		t.Errorf("rename did not show: %v", infos)
	}

	if err := tmux.Kill(ctx, name); err != nil {
		t.Fatal(err)
	}
	if infos, _ = tmux.List(ctx); len(infos) != 1 || infos[0].Name != other {
		t.Errorf("after kill: %v", infos)
	}
	if err := tmux.Kill(ctx, name); err == nil {
		t.Error("killing a session twice succeeded")
	}
}

func TestListSkipsSessionsAgentosDidNotMake(t *testing.T) {
	tmux := newTestTmux(t)
	ctx := context.Background()
	if _, err := tmux.run(ctx, "new-session", "-d", "-s", "not-ours", "sleep", "60"); err != nil {
		t.Fatal(err)
	}
	if infos, err := tmux.List(ctx); err != nil || len(infos) != 0 {
		t.Errorf("List = %v, %v", infos, err)
	}
}
