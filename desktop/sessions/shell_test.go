package sessions_test

import (
	"os/exec"
	"slices"
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
	if records := bus.ReadAll(h.State); len(records) != 0 {
		t.Errorf("the shell or agent left state: %v", records)
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
