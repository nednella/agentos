package cli

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/control"
)

// fakeApp answers commands on a socket in a short temp dir, and points the command at it.
func fakeApp(t *testing.T, handle control.Handler) {
	t.Helper()
	dir, err := os.MkdirTemp("", "aosapp")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	srv, err := control.Listen(control.SocketPath(dir), handle)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Close)
	t.Setenv("AGENTOS_SOCKET", bus.SocketPath(dir))
}

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return strings.TrimSpace(out.String()), err
}

func TestNoteCommandSendsTheText(t *testing.T) {
	var got control.Request
	fakeApp(t, func(_ context.Context, req control.Request) control.Response {
		got = req
		return control.Response{OK: true, Out: "note saved: check"}
	})
	t.Setenv("AGENTOS_SESSION", "demo/1")
	t.Setenv("AGENTOS_PROJECT", "")
	t.Setenv("AGENTOS_DIGEST_PROJECT", "")
	out, err := run(t, "note", "check the", "login")
	if err != nil || out != "note saved: check" {
		t.Fatalf("note = %q, %v", out, err)
	}
	if got.Cmd != "note" || got.Session != "demo/1" || strings.Join(got.Args, "|") != "check the login" {
		t.Errorf("request = %+v", got)
	}
}

func TestNoteReadsStdinWithoutText(t *testing.T) {
	var got control.Request
	fakeApp(t, func(_ context.Context, req control.Request) control.Response {
		got = req
		return control.Response{OK: true, Out: "ok"}
	})
	var out bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader("from a pipe"))
	root.SetOut(&out)
	root.SetArgs([]string{"note"})
	if err := root.Execute(); err != nil || len(got.Args) != 1 || got.Args[0] != "from a pipe" {
		t.Errorf("request = %+v, %v", got, err)
	}
}

func TestNewSendsThePrompt(t *testing.T) {
	var got control.Request
	fakeApp(t, func(_ context.Context, req control.Request) control.Response {
		got = req
		return control.Response{OK: true, Out: "ok"}
	})
	tests := []struct {
		name  string
		args  []string
		stdin string
		want  string
	}{
		{"flag", []string{"new", "side", "topic", "--prompt", "look at auth"}, "", "look at auth"},
		{"stdin", []string{"new", "side", "topic", "--prompt", "-"}, "line one\nline two\n", "line one\nline two"},
		{"none", []string{"new", "side", "topic"}, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := newRootCmd()
			root.SetIn(strings.NewReader(tt.stdin))
			root.SetOut(&bytes.Buffer{})
			root.SetArgs(tt.args)
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if got.Cmd != "new" || strings.Join(got.Args, " ") != "side topic" || got.Opts["prompt"] != tt.want {
				t.Errorf("request = %+v", got)
			}
		})
	}
}

func TestStatsCommandSendsItsFlags(t *testing.T) {
	var got control.Request
	fakeApp(t, func(_ context.Context, req control.Request) control.Response {
		got = req
		return control.Response{OK: true, Out: "tally"}
	})
	if out, err := run(t, "stats", "--days", "30", "--json"); err != nil || out != "tally" {
		t.Fatalf("stats = %q, %v", out, err)
	}
	if got.Cmd != "stats" || got.Opts["days"] != "30" || got.Opts["json"] != "1" {
		t.Errorf("request = %+v", got)
	}
}

func TestAnErrorFromTheAppFailsTheCommand(t *testing.T) {
	fakeApp(t, func(context.Context, control.Request) control.Response {
		return control.Response{Error: "no such thing"}
	})
	if _, err := run(t, "stats"); err == nil || err.Error() != "no such thing" {
		t.Errorf("error = %v", err)
	}
}

func TestWithoutTheAppTheCommandSaysSo(t *testing.T) {
	dir, _ := os.MkdirTemp("", "aosnone")
	defer os.RemoveAll(dir)
	t.Setenv("AGENTOS_SOCKET", bus.SocketPath(dir))
	if _, err := run(t, "stats"); err == nil || !strings.Contains(err.Error(), "agentos is not running") {
		t.Errorf("error = %v", err)
	}
}

func TestParseFlags(t *testing.T) {
	pos, opts, err := parseFlags([]string{"a", "--caption", "x y", "--full", "b", "--n=3", "--", "--raw"}, []string{"caption", "n"}, []string{"full"})
	if err != nil || strings.Join(pos, "|") != "a|b|--raw" || opts["caption"] != "x y" || opts["full"] != "1" || opts["n"] != "3" {
		t.Errorf("parseFlags = %v %v %v", pos, opts, err)
	}
	if _, _, err := parseFlags([]string{"--nope"}, nil, nil); err == nil {
		t.Error("an unknown flag was accepted")
	}
	if _, _, err := parseFlags([]string{"--caption"}, []string{"caption"}, nil); err == nil {
		t.Error("a flag without its value was accepted")
	}
}

func TestShowSendsTheAbsolutePathAndCaption(t *testing.T) {
	var got control.Request
	fakeApp(t, func(_ context.Context, req control.Request) control.Response {
		got = req
		return control.Response{OK: true, Out: "filed as evidence"}
	})
	dir := t.TempDir()
	t.Chdir(dir)
	if out, err := run(t, "show", "shot.png", "--caption", "the page"); err != nil || out != "filed as evidence" {
		t.Fatalf("show = %q, %v", out, err)
	}
	if got.Cmd != "show" || len(got.Args) != 1 || !strings.HasSuffix(got.Args[0], "/shot.png") || !strings.HasPrefix(got.Args[0], "/") || got.Opts["caption"] != "the page" {
		t.Errorf("request = %+v", got)
	}
	if _, err := run(t, "show", "--text", "ran 40 tests"); err != nil || got.Opts["text"] != "ran 40 tests" || len(got.Args) != 0 {
		t.Errorf("text card request = %+v, %v", got, err)
	}
	for _, bad := range [][]string{{"show"}, {"show", "a.png", "--text", "x"}} {
		if _, err := run(t, bad...); err == nil {
			t.Errorf("agentos %v succeeded", bad)
		}
	}
}

func TestBrowserHelpWorksWithoutTheApp(t *testing.T) {
	t.Setenv("AGENTOS_SOCKET", "/nonexistent/agentos.sock")
	for _, args := range [][]string{{"browser", "help"}, {"browser", "--help"}} {
		out, err := run(t, args...)
		if err != nil || !strings.Contains(out, "open <url>") || !strings.Contains(out, "tab <url>") || !strings.Contains(out, "screenshot") {
			t.Errorf("agentos %v = %q, %v", args, out, err)
		}
	}
}

func TestSetupHelpWorksWithoutTheApp(t *testing.T) {
	t.Setenv("AGENTOS_SOCKET", "/nonexistent/agentos.sock")
	out, err := run(t, "setup", "help")
	if err != nil || !strings.Contains(out, ".github/ISSUE_TEMPLATE/issue.md") || !strings.Contains(out, ".claude/commands/work.md") {
		t.Errorf("agentos setup help = %q, %v", out, err)
	}
	for _, bad := range [][]string{{"setup"}, {"setup", "now"}} {
		if _, err := run(t, bad...); err == nil {
			t.Errorf("agentos %v succeeded", bad)
		}
	}
}

func TestBrowserCommandSendsItsWords(t *testing.T) {
	var got control.Request
	fakeApp(t, func(_ context.Context, req control.Request) control.Response {
		got = req
		return control.Response{OK: true, Out: "done"}
	})
	tests := []struct {
		args    []string
		cmd     []string
		opts    map[string]string
		timeout int
	}{
		{[]string{"browser", "open", "https://example.com"}, []string{"open", "https://example.com"}, map[string]string{}, 75_000},
		{[]string{"browser", "screenshot", "--caption", "the page", "--full"}, []string{"screenshot"}, map[string]string{"caption": "the page", "full": "1"}, 75_000},
		{[]string{"browser", "tab", "https://example.com"}, []string{"tab", "https://example.com"}, map[string]string{}, 75_000},
		{[]string{"browser", "open", "https://example.com", "--front"}, []string{"open", "https://example.com"}, map[string]string{"front": "1"}, 75_000},
	}
	for _, tt := range tests {
		if out, err := run(t, tt.args...); err != nil || out != "done" {
			t.Fatalf("agentos %v = %q, %v", tt.args, out, err)
		}
		if got.Cmd != "browser" || strings.Join(got.Args, "|") != strings.Join(tt.cmd, "|") || got.TimeoutMs != tt.timeout {
			t.Errorf("agentos %v: request = %+v", tt.args, got)
		}
		for k, v := range tt.opts {
			if got.Opts[k] != v {
				t.Errorf("agentos %v: option %s = %q, want %q", tt.args, k, got.Opts[k], v)
			}
		}
	}
	for _, flag := range []string{"--bogus", "--append", "--timeout"} {
		if _, err := run(t, "browser", "open", flag); err == nil {
			t.Errorf("the flag %s was accepted", flag)
		}
	}
}

func TestDigestAddSendsItsFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want map[string]string
	}{
		{"all flags", []string{"digest", "add", "--title", "T", "--why", "W", "--url", "https://x", "--source", "S"},
			map[string]string{"title": "T", "why": "W", "url": "https://x", "source": "S"}},
		{"only required", []string{"digest", "add", "--title", "T", "--url", "https://x"},
			map[string]string{"title": "T", "why": "", "url": "https://x", "source": ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got control.Request
			fakeApp(t, func(_ context.Context, req control.Request) control.Response {
				got = req
				return control.Response{OK: true, Out: "added"}
			})
			if out, err := run(t, tt.args...); err != nil || out != "added" {
				t.Fatalf("digest add = %q, %v", out, err)
			}
			if got.Cmd != "digest-add" {
				t.Errorf("cmd = %q", got.Cmd)
			}
			for k, v := range tt.want {
				if got.Opts[k] != v {
					t.Errorf("opt %s = %q, want %q", k, got.Opts[k], v)
				}
			}
		})
	}
}
