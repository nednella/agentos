package main

import (
	"encoding/base64"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestTerminalStream(t *testing.T) {
	h := newHarness(t)
	s, err := h.app.NewSession("term", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.app.TermOpen(s.ID, 100, 30); err != nil {
		t.Fatal(err)
	}
	eventually(t, "shell output", func() bool { return len(h.rec.output(s.ID)) > 0 })

	if err := h.app.TermWrite(s.ID, "echo $((40+2))\n"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "echo result", func() bool { return strings.Contains(h.rec.output(s.ID), "\r\n42\r\n") })

	if err := h.app.TermResize(s.ID, 77, 20); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if err := h.app.TermWrite(s.ID, "stty size\n"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "resized pty", func() bool { return strings.Contains(h.rec.output(s.ID), "\r\n20 77\r\n") })

	h.app.TermClose(s.ID)
	if err := h.app.TermWrite(s.ID, "x"); err == nil {
		t.Error("writing to a closed terminal succeeded")
	}
	if err := h.app.TermOpen("not a session", 80, 24); err == nil {
		t.Error("a name that is no session was attached")
	}
}

func TestClosingATerminalKeepsTheAgentRunning(t *testing.T) {
	h := newHarness(t)
	s, _ := h.app.NewSession("keep", "")
	if err := h.app.TermOpen(s.ID, 80, 24); err != nil {
		t.Fatal(err)
	}
	h.app.TermClose(s.ID)
	time.Sleep(300 * time.Millisecond)
	if err := exec.Command("tmux", "-L", h.socket, "has-session", "-t", s.ID).Run(); err != nil {
		t.Errorf("the session ended with its terminal: %v", err)
	}
	if h.rec.count("term:exit") != 0 {
		t.Error("a terminal we closed ourselves reported an exit")
	}
}

func TestTerminalReportsASessionThatEnds(t *testing.T) {
	h := newHarness(t)
	s, _ := h.app.NewSession("end", "")
	if err := h.app.TermOpen(s.ID, 80, 24); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("tmux", "-L", h.socket, "kill-session", "-t", s.ID).Run(); err != nil {
		t.Fatal(err)
	}
	eventually(t, "term:exit", func() bool { return h.rec.count("term:exit") == 1 })
}

func TestReopeningRedrawsTheScreen(t *testing.T) {
	h := newHarness(t)
	s, _ := h.app.NewSession("again", "")
	if err := h.app.TermOpen(s.ID, 80, 24); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the first draw", func() bool { return len(h.rec.output(s.ID)) > 0 })
	if err := h.app.TermWrite(s.ID, "echo before-reopen\n"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the echo", func() bool { return strings.Contains(h.rec.output(s.ID), "before-reopen") })
	n := len(h.rec.output(s.ID))
	if err := h.app.TermOpen(s.ID, 80, 24); err != nil {
		t.Fatal(err)
	}
	eventually(t, "a full redraw that still shows the screen", func() bool {
		out := h.rec.output(s.ID)
		return len(out) > n && strings.Count(out, "before-reopen") >= 2
	})
}

func TestHeavyOutputIsMerged(t *testing.T) {
	h := newHarness(t)
	s, _ := h.app.NewSession("heavy", "")
	if err := h.app.TermOpen(s.ID, 120, 30); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the prompt", func() bool { return len(h.rec.output(s.ID)) > 0 })
	before := h.rec.count("term:data")
	if err := h.app.TermWrite(s.ID, "seq 1 30000; echo done-seq\n"); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the end of the output", func() bool { return strings.Contains(h.rec.output(s.ID), "done-seq") })
	events := h.rec.count("term:data") - before
	if bytes := len(h.rec.output(s.ID)); events > bytes/200 {
		t.Errorf("%d events for %d bytes: output is not merged", events, bytes)
	}
}

func TestClipboardFromTheAgent(t *testing.T) {
	h := newHarness(t)
	s, _ := h.app.NewSession("clip", "")
	if err := h.app.TermOpen(s.ID, 100, 30); err != nil {
		t.Fatal(err)
	}
	eventually(t, "shell output", func() bool { return len(h.rec.output(s.ID)) > 0 })
	cmd := `printf '\033]52;c;%s\a' $(printf copied-text | base64)` + "\n"
	if err := h.app.TermWrite(s.ID, cmd); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the clipboard", func() bool {
		h.rec.mu.Lock()
		defer h.rec.mu.Unlock()
		return slices.Contains(h.rec.clips, "copied-text")
	})
	if !strings.Contains(h.rec.output(s.ID), "\x1b]52;") {
		t.Error("the OSC 52 bytes were not forwarded")
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
