package sessions_test

import (
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/internal/bus"
)

// shellHarness starts an app whose shell is a plain sh run from an empty home, so no profile of the user runs.
func shellHarness(t *testing.T) *apptest.Harness {
	t.Helper()
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("HOME", t.TempDir())
	return newHarness(t)
}

func tmuxHas(h *apptest.Harness, name string) bool {
	return exec.Command("tmux", "-L", h.Socket, "has-session", "-t", "="+name).Run() == nil
}

func TestShellSession(t *testing.T) {
	h := shellHarness(t)
	if got := h.Snapshot().Shells; len(got) != 0 {
		t.Fatalf("shells before one is opened = %q", got)
	}
	shell, err := h.ShellOpen()
	if err != nil || shell.ID != "main/shell" {
		t.Fatalf("ShellOpen = %+v, %v", shell, err)
	}
	if got := h.Snapshot().Shells; !slices.Equal(got, []string{shell.ID}) {
		t.Errorf("Snapshot.Shells = %q", got)
	}
	if again, err := h.ShellOpen(); err != nil || again != shell {
		t.Errorf("second ShellOpen = %+v, %v", again, err)
	}
	if !tmuxHas(h, shell.ID) {
		t.Fatal("no tmux session for the shell")
	}

	agent, err := h.NewSession("agent", "")
	if err != nil {
		t.Fatal(err)
	}
	if agent.N != 1 {
		t.Errorf("the first agent is session %d: the shell took a number", agent.N)
	}
	h.Sessions().Refresh()
	if list := h.Sessions().List(); len(list) != 1 || list[0].ID != agent.ID {
		t.Errorf("session rows = %+v", list)
	}
	if got := h.Snapshot().Project.Sessions; got != 1 {
		t.Errorf("project counts %d sessions", got)
	}
	if records := bus.ReadAll(h.State); len(records) != 1 || records[agent.ID].Title != "agent" {
		t.Errorf("state = %v, want only the agent's", records)
	}
	if err := h.KillSession(shell.ID); err == nil {
		t.Error("KillSession stopped the shell")
	}
	if err := h.RenameSession(shell.ID, "x"); err == nil {
		t.Error("RenameSession renamed the shell")
	}
	if err := h.Cleanup(shell.ID, true); err == nil {
		t.Error("Cleanup accepted the shell")
	}
	if !tmuxHas(h, shell.ID) {
		t.Error("the shell is gone")
	}
}

func TestShellFollowsTheProject(t *testing.T) {
	h := shellHarness(t)
	if _, err := h.AddProjectDir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	other := h.Snapshot().Project.Name
	if _, err := h.SwitchProject("main"); err != nil {
		t.Fatal(err)
	}
	shell, err := h.ShellOpen()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.SwitchProject(other); err != nil {
		t.Fatal(err)
	}
	if got := h.Snapshot().Shells; len(got) != 0 {
		t.Errorf("the other project shows the shells %q", got)
	}
	if _, err := h.SwitchProject("main"); err != nil {
		t.Fatal(err)
	}
	if got := h.Snapshot().Shells; !slices.Equal(got, []string{shell.ID}) {
		t.Errorf("shells after switching back = %q", got)
	}
}

func TestShellSurvivesARestart(t *testing.T) {
	h := shellHarness(t)
	shell, err := h.ShellOpen()
	if err != nil {
		t.Fatal(err)
	}
	second := h.Restart(t)
	if got := second.Snapshot().Shells; !slices.Equal(got, []string{shell.ID}) {
		t.Errorf("shells after a restart = %q", got)
	}
	if list := second.Sessions().List(); len(list) != 0 {
		t.Errorf("the shell came back as a row: %+v", list)
	}
}

func TestConcurrentShellOpensShareOneShell(t *testing.T) {
	h := shellHarness(t)
	const calls = 4
	ids := make(chan string, calls)
	errs := make(chan error, calls)
	for range calls {
		go func() {
			shell, err := h.ShellOpen()
			ids <- shell.ID
			errs <- err
		}()
	}
	for range calls {
		if err := <-errs; err != nil {
			t.Errorf("ShellOpen: %v", err)
		}
		if id := <-ids; id != "main/shell" {
			t.Errorf("ShellOpen = %q", id)
		}
	}
}

func TestShellTabs(t *testing.T) {
	h := shellHarness(t)
	first, err := h.ShellOpen()
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.ShellNew()
	if err != nil || second.ID != "main/shell-2" {
		t.Fatalf("ShellNew = %+v, %v", second, err)
	}
	third, err := h.ShellNew()
	if err != nil || third.ID != "main/shell-3" {
		t.Fatalf("ShellNew = %+v, %v", third, err)
	}
	want := []string{first.ID, second.ID, third.ID}
	if got := h.Snapshot().Shells; !slices.Equal(got, want) {
		t.Errorf("Shells = %q, want %q", got, want)
	}
	if again, err := h.ShellOpen(); err != nil || again != first {
		t.Errorf("ShellOpen with shells = %+v, %v", again, err)
	}

	if err := h.ShellClose(second.ID); err != nil {
		t.Fatal(err)
	}
	if tmuxHas(h, second.ID) {
		t.Error("the closed shell still runs")
	}
	if got := h.Snapshot().Shells; !slices.Equal(got, []string{first.ID, third.ID}) {
		t.Errorf("Shells after close = %q", got)
	}
	if again, err := h.ShellNew(); err != nil || again != second {
		t.Errorf("ShellNew reuses the free number: %+v, %v", again, err)
	}

	agent, err := h.NewSession("agent", "")
	if err != nil || agent.N != 1 {
		t.Errorf("an agent after shells = %+v, %v", agent, err)
	}
	if err := h.ShellClose(agent.ID); err == nil {
		t.Error("ShellClose stopped an agent")
	}
}

// panePath is the folder the shell in the tmux session works in.
func panePath(t *testing.T, h *apptest.Harness, name string) string {
	t.Helper()
	out, err := exec.Command("tmux", "-L", h.Socket, "display-message", "-p", "-t", "="+name+":", "#{pane_current_path}").Output()
	if err != nil {
		t.Fatal(err)
	}
	return realPath(t, strings.TrimSpace(string(out)))
}

func realPath(t *testing.T, dir string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return real
}

func TestSessionShell(t *testing.T) {
	h := shellHarness(t)
	chat, err := h.NewChat("ask")
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := h.Session(chat.ID); got.Shell != "" {
		t.Fatalf("shell before one is opened = %q", got.Shell)
	}
	shell, err := h.SessionShellOpen(chat.ID)
	if err != nil || shell.ID != chat.ID+"-shell" {
		t.Fatalf("SessionShellOpen = %+v, %v", shell, err)
	}
	if again, err := h.SessionShellOpen(chat.ID); err != nil || again != shell {
		t.Errorf("second SessionShellOpen = %+v, %v", again, err)
	}
	if got := panePath(t, h, shell.ID); got != realPath(t, h.Dir) {
		t.Errorf("the shell of a session with no worktree works in %s, want the project folder %s", got, h.Dir)
	}
	if got, _ := h.Session(chat.ID); got.Shell != shell.ID {
		t.Errorf("row shell = %q", got.Shell)
	}
	h.Sessions().Refresh()
	if list := h.Sessions().List(); len(list) != 1 || list[0].ID != chat.ID {
		t.Errorf("session rows = %+v", list)
	}
	if got := h.Snapshot().Shells; len(got) != 0 {
		t.Errorf("the drawer shows the session's shell: %q", got)
	}
	if drawer, err := h.ShellOpen(); err != nil || drawer.ID != "main/shell" {
		t.Errorf("ShellOpen next to a session's shell = %+v, %v", drawer, err)
	}
	if _, err := h.SessionShellOpen(shell.ID); err == nil {
		t.Error("a shell got a shell")
	}
	if _, err := h.SessionShellOpen("main/nosuch12"); err == nil {
		t.Error("a missing session got a shell")
	}

	if err := h.KillSession(chat.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the session ended", func() bool { got, _ := h.Session(chat.ID); return got.State == "ended" })
	if got, _ := h.Session(chat.ID); !tmuxHas(h, shell.ID) || got.Shell != shell.ID {
		t.Fatalf("after the agent ended: shell runs %v, row shell %q", tmuxHas(h, shell.ID), got.Shell)
	}
	if err := h.DismissSession(chat.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the shell to stop", func() bool { return !tmuxHas(h, shell.ID) })
	if !tmuxHas(h, "main/shell") {
		t.Error("dismissing the row stopped the drawer's shell")
	}
}

func TestSessionShellStartsInTheWorktree(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	h := newHarness(t)
	repoFixture(t, h)
	wt := issueWorktree(t, h)
	s := issueSession(h, t, 7)
	h.GH.SetPR(prJSON("OPEN", false, "[]", 0, 0))
	h.RefreshPRs()
	eventually(t, "the worktree", func() bool { got, _ := h.Session(s.ID); return got.Worktree != "" })
	shell, err := h.SessionShellOpen(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := panePath(t, h, shell.ID); got != realPath(t, wt) {
		t.Errorf("the shell works in %s, want the worktree %s", got, wt)
	}
}

func TestSessionShellSurvivesARestart(t *testing.T) {
	h := shellHarness(t)
	s, err := h.NewSession("agent", "")
	if err != nil {
		t.Fatal(err)
	}
	shell, err := h.SessionShellOpen(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	second := h.Restart(t)
	if got, _ := second.Session(s.ID); got.Shell != shell.ID {
		t.Errorf("row shell after a restart = %q, want %q", got.Shell, shell.ID)
	}
	if again, err := second.SessionShellOpen(s.ID); err != nil || again != shell {
		t.Errorf("SessionShellOpen after a restart = %+v, %v", again, err)
	}
	if got := second.Snapshot().Shells; len(got) != 0 {
		t.Errorf("the drawer shows the session's shell after a restart: %q", got)
	}
}

func TestShellExitClearsTheRow(t *testing.T) {
	h := shellHarness(t)
	s, err := h.NewSession("agent", "")
	if err != nil {
		t.Fatal(err)
	}
	shell, err := h.SessionShellOpen(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("tmux", "-L", h.Socket, "send-keys", "-t", "="+shell.ID+":", "exit", "Enter").Run(); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the shell to end", func() bool { return !tmuxHas(h, shell.ID) })
	h.Sessions().Refresh()
	if got, _ := h.Session(s.ID); got.Shell != "" {
		t.Errorf("row shell after exit = %q", got.Shell)
	}
	if again, err := h.SessionShellOpen(s.ID); err != nil || again != shell || !tmuxHas(h, shell.ID) {
		t.Errorf("SessionShellOpen after exit = %+v, %v", again, err)
	}
}
