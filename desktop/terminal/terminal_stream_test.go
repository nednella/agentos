package terminal_test

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTerminalStream(t *testing.T) {
	h := newHarness(t)
	s, err := h.NewSession("term", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.TermOpen(s.ID, 100, 30); err != nil {
		t.Fatal(err)
	}
	eventually(t, "shell output", func() bool { return len(h.Rec.Output(s.ID)) > 0 })

	if err := h.TermWrite(s.ID, "echo $((40+2))\n"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "echo result", func() bool { return strings.Contains(h.Rec.Output(s.ID), "\r\n42\r\n") })

	if err := h.TermResize(s.ID, 77, 20); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if err := h.TermWrite(s.ID, "stty size\n"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "resized pty", func() bool { return strings.Contains(h.Rec.Output(s.ID), "\r\n20 77\r\n") })

	h.TermClose(s.ID)
	if err := h.TermWrite(s.ID, "x"); err == nil {
		t.Error("writing to a closed terminal succeeded")
	}
	if err := h.TermOpen("not a session", 80, 24); err == nil {
		t.Error("a name that is no session was attached")
	}
}

func TestClosingATerminalKeepsTheAgentRunning(t *testing.T) {
	h := newHarness(t)
	s, _ := h.NewSession("keep", "")
	if err := h.TermOpen(s.ID, 80, 24); err != nil {
		t.Fatal(err)
	}
	h.TermClose(s.ID)
	time.Sleep(300 * time.Millisecond)
	if err := exec.Command("tmux", "-L", h.Socket, "has-session", "-t", s.ID).Run(); err != nil {
		t.Errorf("the session ended with its terminal: %v", err)
	}
	if h.Rec.Count("term:exit") != 0 {
		t.Error("a terminal we closed ourselves reported an exit")
	}
}

func TestTerminalReportsASessionThatEnds(t *testing.T) {
	h := newHarness(t)
	s, _ := h.NewSession("end", "")
	if err := h.TermOpen(s.ID, 80, 24); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("tmux", "-L", h.Socket, "kill-session", "-t", s.ID).Run(); err != nil {
		t.Fatal(err)
	}
	eventually(t, "term:exit", func() bool { return h.Rec.Count("term:exit") == 1 })
}

func TestReopeningRedrawsTheScreen(t *testing.T) {
	h := newHarness(t)
	s, _ := h.NewSession("again", "")
	if err := h.TermOpen(s.ID, 80, 24); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the first draw", func() bool { return len(h.Rec.Output(s.ID)) > 0 })
	if err := h.TermWrite(s.ID, "echo before-reopen\n"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the echo", func() bool { return strings.Contains(h.Rec.Output(s.ID), "before-reopen") })
	n := len(h.Rec.Output(s.ID))
	if err := h.TermOpen(s.ID, 80, 24); err != nil {
		t.Fatal(err)
	}
	eventually(t, "a full redraw that still shows the screen", func() bool {
		out := h.Rec.Output(s.ID)
		return len(out) > n && strings.Count(out, "before-reopen") >= 2
	})
}

func TestHeavyOutputIsMerged(t *testing.T) {
	h := newHarness(t)
	s, _ := h.NewSession("heavy", "")
	if err := h.TermOpen(s.ID, 120, 30); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the prompt", func() bool { return len(h.Rec.Output(s.ID)) > 0 })
	before := h.Rec.Count("term:data")
	if err := h.TermWrite(s.ID, "seq 1 30000; echo done-seq\n"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the end of the output", func() bool { return strings.Contains(h.Rec.Output(s.ID), "done-seq") })
	events := h.Rec.Count("term:data") - before
	if bytes := len(h.Rec.Output(s.ID)); events > bytes/200 {
		t.Errorf("%d events for %d bytes: output is not merged", events, bytes)
	}
}

func TestClipboardFromTheAgent(t *testing.T) {
	h := newHarness(t)
	s, _ := h.NewSession("clip", "")
	if err := h.TermOpen(s.ID, 100, 30); err != nil {
		t.Fatal(err)
	}
	eventually(t, "shell output", func() bool { return len(h.Rec.Output(s.ID)) > 0 })
	cmd := `printf '\033]52;c;%s\a' $(printf copied-text | base64)` + "\n"
	if err := h.TermWrite(s.ID, cmd); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the clipboard", func() bool { return h.Rec.HasClip("copied-text") })
	if !strings.Contains(h.Rec.Output(s.ID), "\x1b]52;") {
		t.Error("the OSC 52 bytes were not forwarded")
	}
}

func TestTerminalAttachesToTheShell(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("HOME", t.TempDir())
	h := newHarness(t)
	shell, err := h.ShellOpen()
	if err != nil {
		t.Fatal(err)
	}
	if err := h.TermOpen(shell.ID, 100, 30); err != nil {
		t.Fatal(err)
	}
	eventually(t, "shell output", func() bool { return len(h.Rec.Output(shell.ID)) > 0 })
	if err := h.TermWrite(shell.ID, "echo \"$AGENTOS_PROJECT|${AGENTOS_SESSION-unset}|${AGENTOS_SOCKET:+set}|$(pwd -P)\"\n"); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(h.Dir)
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "the shell's environment", func() bool {
		return strings.Contains(h.Rec.Output(shell.ID), "\r\nmain|unset|set|"+want+"\r\n")
	})
	h.TermClose(shell.ID)
}

func TestOpeningAnOpenTerminalKeepsItsStream(t *testing.T) {
	h := newHarness(t)
	s, _ := h.NewSession("keep the stream", "")
	if err := h.TermOpen(s.ID, 80, 24); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the first draw", func() bool { return len(h.Rec.Output(s.ID)) > 0 })
	clients := func() string {
		out, err := exec.Command("tmux", "-L", h.Socket, "list-clients", "-F", "#{client_pid} #{client_width}x#{client_height}").Output()
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(out))
	}
	before := clients()
	if strings.Count(before, "\n") != 0 || before == "" {
		t.Fatalf("clients = %q, want one", before)
	}
	pid, _, _ := strings.Cut(before, " ")

	if err := h.TermOpen(s.ID, 90, 26); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the same client at the new size", func() bool { return clients() == pid+" 90x26" })
	if h.Rec.Count("term:exit") != 0 {
		t.Error("reopening ended the stream")
	}
	if err := h.TermWrite(s.ID, "echo still-here\n"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the stream to carry on", func() bool { return strings.Contains(h.Rec.Output(s.ID), "still-here") })
}
