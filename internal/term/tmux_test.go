package term

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
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

// paneScript leaves the children a session's agent can leave: one that ignores SIGHUP and SIGTERM, one in a
// process group of its own, one that moves to a session of its own after Kill has looked, and one whose parent has
// exited. All of them ignore SIGHUP and SIGTERM, as the trap is inherited. It writes their pids and its own to the
// file in $1.
const paneScript = `trap "" HUP TERM
sleep 300 & echo $! >> "$1"
set -m; sleep 301 & echo $! >> "$1"; set +m
perl -e 'use POSIX; select(undef, undef, undef, 0.5); setsid(); sleep 302' & echo $! >> "$1"
(sleep 303 & echo $! >> "$1")
echo $$ >> "$1"
wait`

func TestKillStopsEveryProcessUnderThePane(t *testing.T) {
	tmux := newTestTmux(t)
	ctx := context.Background()
	pidFile := filepath.Join(t.TempDir(), "pids")
	name := session.Name{Project: "demo", Token: "k1"}
	if err := tmux.NewSession(ctx, name, "agent", t.TempDir(), nil, []string{"sh", "-c", paneScript, "sh", pidFile}, 100, 30); err != nil {
		t.Fatal(err)
	}
	bystander := exec.Command("sleep", "60")
	if err := bystander.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = bystander.Process.Kill(); _ = bystander.Wait() }()
	var pids []int
	for range 100 {
		data, _ := os.ReadFile(pidFile)
		if pids = pidsIn(string(data)); len(pids) == 5 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(pids) != 5 {
		t.Fatalf("the pane wrote pids %v", pids)
	}
	t.Cleanup(func() {
		for _, pid := range pids {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	})

	if err := tmux.Kill(ctx, name); err != nil {
		t.Fatal(err)
	}
	for _, pid := range pids {
		if !gone(pid) {
			t.Errorf("process %d outlived the session", pid)
		}
	}
	if err := syscall.Kill(bystander.Process.Pid, 0); err != nil {
		t.Errorf("a process outside the session was stopped: %v", err)
	}
}

func pidsIn(s string) []int {
	var pids []int
	for _, f := range strings.Fields(s) {
		if pid, err := strconv.Atoi(f); err == nil {
			pids = append(pids, pid)
		}
	}
	return pids
}

// gone waits for pid to stop existing; once its parent is gone too, launchd reaps it.
func gone(pid int) bool {
	for range 50 {
		if syscall.Kill(pid, 0) != nil {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
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

func TestListGivesThePanesProcessUnderTheServer(t *testing.T) {
	tmux := newTestTmux(t)
	ctx := context.Background()
	if pid, err := tmux.ServerPID(ctx); err != nil || pid != 0 {
		t.Fatalf("ServerPID without a server = %d, %v", pid, err)
	}
	name := session.Name{Project: "demo", Token: "a1"}
	if err := tmux.NewSession(ctx, name, "title", t.TempDir(), nil, []string{"sleep", "60"}, 100, 30); err != nil {
		t.Fatal(err)
	}
	infos, err := list(tmux, ctx)
	if err != nil || len(infos) != 1 {
		t.Fatalf("List = %v, %v", infos, err)
	}
	server, err := tmux.ServerPID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	parent, err := exec.Command("ps", "-o", "ppid=", "-p", strconv.Itoa(infos[0].PID)).Output()
	if err != nil || strings.TrimSpace(string(parent)) != strconv.Itoa(server) {
		t.Errorf("pane %d has parent %q, want the server %d (%v)", infos[0].PID, parent, server, err)
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
