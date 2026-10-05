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
