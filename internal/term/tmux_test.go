package term

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

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

	if infos, err := list(tmux, ctx); err != nil || len(infos) != 0 {
		t.Fatalf("List without a server = %v, %v", infos, err)
	}

	name := session.Name{Project: "demo", Token: "a1"}
	dir := t.TempDir()
	if err := tmux.NewSession(ctx, name, "first agent", dir, []string{"AGENTOS_SESSION=demo/a1"}, []string{"sleep", "60"}, 100, 30); err != nil {
		t.Fatal(err)
	}
	other := session.Name{Project: "demo", Token: "b2"}
	if err := tmux.NewSession(ctx, other, "tabs\tand ünïcode", dir, nil, []string{"sleep", "60"}, 100, 30); err != nil {
		t.Fatal(err)
	}

	if err := tmux.SetIssue(ctx, name, 394); err != nil {
		t.Fatal(err)
	}
	if err := tmux.SetModel(ctx, name, "opus", "high"); err != nil {
		t.Fatal(err)
	}
	infos, err := list(tmux, ctx)
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
	if got := byName[name]; got.Model != "opus" || got.Effort != "high" {
		t.Errorf("first model = %q, effort = %q", got.Model, got.Effort)
	}
	if got := byName[other]; got.Model != "" || got.Effort != "" {
		t.Errorf("second model = %q, effort = %q, want none", got.Model, got.Effort)
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
	if infos, _ = list(tmux, ctx); !strings.Contains(fmt.Sprint(infos), "renamed") {
		t.Errorf("rename did not show: %v", infos)
	}

	if err := tmux.Kill(ctx, name); err != nil {
		t.Fatal(err)
	}
	if infos, _ = list(tmux, ctx); len(infos) != 1 || infos[0].Name != other {
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
	if infos, err := list(tmux, ctx); err != nil || len(infos) != 0 {
		t.Errorf("List = %v, %v", infos, err)
	}
}

func TestShellSessionsAreListedApart(t *testing.T) {
	tmux := newTestTmux(t)
	ctx := context.Background()
	agent, shell := session.Name{Project: "demo", Token: "a1"}, session.Name{Project: "demo", Shell: 1}
	if tmux.Has(ctx, shell) {
		t.Fatal("Has without a server is true")
	}
	for _, n := range []session.Name{agent, shell} {
		if err := tmux.NewSession(ctx, n, "title", t.TempDir(), nil, []string{"sleep", "60"}, 100, 30); err != nil {
			t.Fatal(err)
		}
	}
	infos, shells, err := tmux.ListAll(ctx)
	if err != nil || len(infos) != 1 || infos[0].Name != agent || len(shells) != 1 || shells[0] != shell {
		t.Fatalf("ListAll = %v, %v, %v", infos, shells, err)
	}
	if infos, err = list(tmux, ctx); err != nil || len(infos) != 1 {
		t.Errorf("List = %v, %v", infos, err)
	}
	if !tmux.Has(ctx, shell) || !tmux.Has(ctx, agent) || tmux.Has(ctx, session.Name{Project: "demo", Token: "b2"}) || tmux.Has(ctx, session.Name{Project: "dem", Shell: 1}) {
		t.Error("Has gave a wrong answer")
	}
}

func list(t *Tmux, ctx context.Context) ([]Info, error) {
	infos, _, err := t.ListAll(ctx)
	return infos, err
}

func TestTargetsMatchTheWholeName(t *testing.T) {
	tmux := newTestTmux(t)
	ctx := context.Background()
	one, ten := session.Name{Project: "p", Token: "a1"}, session.Name{Project: "p", Token: "a10"}
	for _, n := range []session.Name{one, ten} {
		if err := tmux.NewSession(ctx, n, "title", t.TempDir(), nil, []string{"sleep", "60"}, 100, 30); err != nil {
			t.Fatal(err)
		}
	}
	if err := tmux.Rename(ctx, one, "renamed"); err != nil {
		t.Fatal(err)
	}
	if err := tmux.SetIssue(ctx, one, 7); err != nil {
		t.Fatal(err)
	}
	infos, _ := list(tmux, ctx)
	for _, in := range infos {
		if in.Name == ten && (in.Title != "title" || in.Issue != "") {
			t.Errorf("p/a10 changed with p/1: %+v", in)
		}
	}
	if err := tmux.Kill(ctx, one); err != nil {
		t.Fatal(err)
	}
	if !tmux.Has(ctx, ten) || tmux.Has(ctx, one) {
		t.Error("killing p/a1 did not leave exactly p/a10")
	}
	if err := tmux.Kill(ctx, one); err == nil {
		t.Error("killing p/a1 again hit p/a10")
	}
}

func TestTrailingSemicolonSurvives(t *testing.T) {
	tmux := newTestTmux(t)
	ctx := context.Background()
	name := session.Name{Project: "demo", Token: "a1"}
	if err := tmux.NewSession(ctx, name, "fix a;", t.TempDir(), nil, []string{"cat"}, 100, 30); err != nil {
		t.Fatal(err)
	}
	if err := tmux.Rename(ctx, name, "then b;"); err != nil {
		t.Fatal(err)
	}
	infos, _ := list(tmux, ctx)
	if len(infos) != 1 || infos[0].Title != "then b;" {
		t.Fatalf("title = %+v", infos)
	}
	for _, text := range []string{"one line;", "two\nlines;"} {
		if err := tmux.Type(ctx, name, text); err != nil {
			t.Fatalf("Type(%q): %v", text, err)
		}
	}
	time.Sleep(300 * time.Millisecond)
	out, err := tmux.run(ctx, "capture-pane", "-p", "-t", target(name))
	if err != nil || !strings.Contains(out, "one line;") || !strings.Contains(out, "lines;") {
		t.Errorf("pane = %q, %v", out, err)
	}
}

func TestListFailureIsNotNoSessions(t *testing.T) {
	for msg, want := range map[string]bool{
		"tmux list-sessions: exit status 1: no server running on /tmp/x":                            true,
		"tmux list-sessions: exit status 1: error connecting to /tmp/x (No such file or directory)": true,
		"tmux list-sessions: exit status 1: error connecting to /tmp/x (Permission denied)":         false,
		"tmux list-sessions: exit status 1: protocol version mismatch":                              false,
	} {
		if got := noServer(msg); got != want {
			t.Errorf("noServer(%q) = %v", msg, got)
		}
	}
}

func TestCleanEnvDropsInheritedSession(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1/default,1,0")
	t.Setenv("AGENTOS_SESSION", "other/abc")
	t.Setenv("AGENTOS_ISSUE", "7")
	t.Setenv("AGENTOS_PROJECT", "other")
	t.Setenv("AGENTOS_KEEP", "yes")
	env := strings.Join(cleanEnv(), "\n")
	for _, name := range inherited {
		if strings.Contains(env, name+"=") {
			t.Errorf("%s was passed on", name)
		}
	}
	if !strings.Contains(env, "AGENTOS_KEEP=yes") {
		t.Error("an unrelated variable was dropped")
	}
}
