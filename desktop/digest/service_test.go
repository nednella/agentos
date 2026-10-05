package digest_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/digest"
	"github.com/nednella/agentos/desktop/internal/apptest"
	ctl "github.com/nednella/agentos/internal/control"
)

func envValue(env []string, key string) string {
	for i := len(env) - 1; i >= 0; i-- {
		if v, ok := strings.CutPrefix(env[i], key+"="); ok {
			return v
		}
	}
	return ""
}

func lastDigest(h *apptest.Harness) digest.Digest {
	d, _ := h.Rec.Last("digest").(digest.Digest)
	return d
}

func TestDigestRun(t *testing.T) {
	h := newHarness(t)
	t.Setenv("AGENTOS_SESSION", "main/9") // a run must not inherit a session
	t.Setenv("DEPLOY_TOKEN", "secret")
	t.Setenv("ANTHROPIC_API_KEY", "claude-needs-this")
	if err := os.WriteFile(filepath.Join(h.Dir, "package.json"), []byte(`{"dependencies":{"wails-ish":"1"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	h.Claude.SetDo(func(call apptest.DigestCall) {
		for _, leaked := range []string{"AGENTOS_SESSION", "DEPLOY_TOKEN"} {
			if envValue(call.Env, leaked) != "" {
				t.Errorf("%s leaked into the digest run", leaked)
			}
		}
		if entries, err := os.ReadDir(call.Dir); err != nil || len(entries) != 0 {
			t.Errorf("the run's folder %s is not empty: %v, %v", call.Dir, entries, err)
		}
		for i, item := range [][4]string{
			{"Wails 2.15 released", "you build on Wails", "https://example.com/wails", "wails"},
			{"Duplicate", "same link", "https://example.com/wails", "wails"},
			{"Claude Code hooks", "hooks can match commands", "https://example.com/hooks", "Claude Code"},
		} {
			resp := h.Ask(t, ctl.Request{Cmd: "digest-add", Project: envValue(call.Env, "AGENTOS_DIGEST_PROJECT"),
				Opts: map[string]string{"title": item[0], "why": item[1], "url": item[2], "source": item[3]}})
			if !resp.OK || (i == 1 && resp.Out != "already in the digest") {
				t.Errorf("digest-add %d = %+v", i, resp)
			}
		}
	})
	if err := h.RunDigest(); err != nil {
		t.Fatal(err)
	}
	if err := h.RunDigest(); err == nil {
		t.Error("a second run started while one was going")
	}
	eventually(t, "the digest to finish", func() bool { d := lastDigest(h); return !d.Running && d.LastRunAt > 0 })

	call := h.Claude.Calls()[0]
	if call.Dir == h.Dir || !strings.Contains(call.Dir, "agentos-digest") || exists(call.Dir) {
		t.Errorf("the run's folder = %s (the project is %s): want a fresh folder that is removed afterwards", call.Dir, h.Dir)
	}
	if envValue(call.Env, "ANTHROPIC_API_KEY") != "claude-needs-this" || envValue(call.Env, "PATH") == "" || envValue(call.Env, "HOME") == "" {
		t.Errorf("claude's own environment was dropped: %v", call.Env)
	}
	if !strings.Contains(call.Args[2], "wails-ish") {
		t.Errorf("the prompt does not list the project's dependencies:\n%s", call.Args[2])
	}
	if len(call.Args) != 5 || call.Args[0] != "claude" || call.Args[1] != "-p" || call.Args[3] != "--allowedTools" ||
		call.Args[4] != "WebSearch WebFetch Bash(agentos digest add:*)" || !strings.Contains(call.Args[2], "agentos digest add --title") {
		t.Errorf("command = %q in %s", call.Args, call.Dir)
	}
	if envValue(call.Env, "AGENTOS_DIGEST_PROJECT") != "main" || envValue(call.Env, "AGENTOS_SOCKET") == "" {
		t.Errorf("env = %v", call.Env)
	}
	if got := lastDigest(h).Project; got != "main" {
		t.Errorf("digest event project = %q", got)
	}
	d := h.Digest()
	if len(d.Items) != 2 || d.Items[0].Title != "Wails 2.15 released" || d.Items[1].Source != "Claude Code" || d.Error != "" || d.NextRunAt <= d.LastRunAt {
		t.Fatalf("digest = %+v", d)
	}
	note, err := h.DigestToNote(d.Items[0].ID)
	if err != nil || !strings.HasPrefix(note.Text, "Wails 2.15 released\n\nyou build on Wails\nhttps://example.com/wails") {
		t.Fatalf("DigestToNote = %+v, %v", note, err)
	}
	if d = h.Digest(); d.Items[0].NoteID != note.ID {
		t.Errorf("the item does not know its note: %+v", d.Items[0])
	}
	if err := h.DismissDigestItem(d.Items[1].ID); err != nil || len(h.Digest().Items) != 1 {
		t.Errorf("dismiss: %v", err)
	}
	if err := h.DismissDigestItem("nope"); err == nil {
		t.Error("dismissed an unknown item")
	}
	if resp := h.Ask(t, ctl.Request{Cmd: "digest-add", Project: "main", Opts: map[string]string{"title": "x", "url": "https://example.com"}}); resp.OK {
		t.Error("digest-add outside a run succeeded")
	}
	if resp := h.Ask(t, ctl.Request{Cmd: "digest-add", Opts: map[string]string{"title": "x", "url": "https://example.com"}}); resp.OK {
		t.Error("digest-add without a project succeeded")
	}
}

func TestDigestAddOnlyTakesWebLinks(t *testing.T) {
	h := newHarness(t)
	h.Claude.SetDo(func(call apptest.DigestCall) {
		for url, ok := range map[string]bool{
			"https://example.com/a": true, "http://example.com/b": true,
			"javascript:alert(1)": false, "file:///etc/passwd": false, "ftp://example.com/x": false, "example.com/c": false, "https://": false,
		} {
			resp := h.Ask(t, ctl.Request{Cmd: "digest-add", Project: envValue(call.Env, "AGENTOS_DIGEST_PROJECT"), Opts: map[string]string{"title": "t", "url": url}})
			if resp.OK != ok {
				t.Errorf("digest-add %q: ok %v, want %v (%s)", url, resp.OK, ok, resp.Error)
			}
			if !ok && !strings.Contains(resp.Error, "http or https") {
				t.Errorf("digest-add %q: error %q does not say why", url, resp.Error)
			}
		}
	})
	if err := h.RunDigest(); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the digest to finish", func() bool { d := lastDigest(h); return !d.Running && d.LastRunAt > 0 })
	if n := len(h.Digest().Items); n != 2 {
		t.Errorf("%d items kept, want the 2 web links", n)
	}
}

func TestDigestFailures(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  error
		want string
	}{
		{"claude missing", fmt.Errorf("claude: %w", exec.ErrNotFound), "claude was not found"},
		{"claude fails", context.Canceled, "context canceled"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			h.Claude.Err = tt.err
			if err := h.RunDigest(); err != nil {
				t.Fatal(err)
			}
			eventually(t, "the failure", func() bool { d := lastDigest(h); return !d.Running && d.Error != "" })
			if d := lastDigest(h); !strings.Contains(d.Error, tt.want) || d.LastRunAt != 0 {
				t.Errorf("digest = %+v", d)
			}
		})
	}
}

func TestDigestAutomaticRuns(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{DigestFirst: 50 * time.Millisecond, DigestTick: 50 * time.Millisecond})
	eventually(t, "the automatic run", func() bool { return len(h.Claude.Calls()) == 1 && !h.Digest().Running })
	time.Sleep(300 * time.Millisecond)
	if n := len(h.Claude.Calls()); n != 1 {
		t.Errorf("ran %d times; a run in the last week should stop the next", n)
	}
	if h.Digest().NextRunAt == 0 {
		t.Error("no next run time")
	}

	off := apptest.NewWith(t, apptest.Options{ProjectExtra: "    digest: off\n", DigestFirst: 50 * time.Millisecond, DigestTick: 50 * time.Millisecond})
	time.Sleep(400 * time.Millisecond)
	if n := len(off.Claude.Calls()); n != 0 || off.Digest().NextRunAt != 0 {
		t.Errorf("a project with the digest off ran %d times, next run %d", n, off.Digest().NextRunAt)
	}
}

func TestDigestCommand(t *testing.T) {
	h := newHarness(t)
	if resp := h.Ask(t, ctl.Request{Cmd: "digest"}); !resp.OK || resp.Out != "ok" || h.Rec.LastUI().Name != "digest" {
		t.Errorf("digest = %+v, ui %+v", resp, h.Rec.LastUI())
	}
	resp := h.Ask(t, ctl.Request{Cmd: "digest", Opts: map[string]string{"run": "1"}})
	if !resp.OK || resp.Out != "digest started" {
		t.Fatalf("digest --run = %+v", resp)
	}
	eventually(t, "the run to call claude", func() bool { return len(h.Claude.Calls()) > 0 })
}
