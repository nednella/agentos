package browser

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/session"
)

// fakeAgentBrowser makes b's agent-browser a stand-in that appends each call to the returned log, as
// "<session> <socket dir> <args>".
func fakeAgentBrowser(t *testing.T, b *Browsers) (log string) {
	t.Helper()
	dir := t.TempDir()
	log = filepath.Join(dir, "calls")
	script := "#!/bin/sh\necho \"$AGENT_BROWSER_SESSION $AGENT_BROWSER_SOCKET_DIR $*\" >> " + log + "\n"
	b.agentBrowser = filepath.Join(dir, "agent-browser")
	if err := os.WriteFile(b.agentBrowser, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return log
}

func readBinding(t *testing.T, b *Browsers, token string) agentBinding {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(b.agentDir(), token+".target"))
	if err != nil {
		t.Fatal(err)
	}
	var got agentBinding
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("binding %s: %v", data, err)
	}
	return got
}

func calls(t *testing.T, log string) []string {
	t.Helper()
	data, _ := os.ReadFile(log)
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func TestAgentEnvBindsTheSessionToNoTab(t *testing.T) {
	b := New(t.TempDir(), func(string, any) {})
	b.binary = "/chrome"
	fakeAgentBrowser(t, b)
	env, err := b.AgentEnv("p/abcd2345", "/tmp/session")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"PATH=" + filepath.Dir(b.agentBrowser) + string(os.PathListSeparator) + os.Getenv("PATH"),
		"AGENT_BROWSER_SESSION=abcd2345",
		"AGENT_BROWSER_SOCKET_DIR=" + filepath.Join(b.dataDir, "agent-browser"),
		"AGENT_BROWSER_CDP=" + strconv.Itoa(b.Port("p")),
		"AGENT_BROWSER_PIN_TAB=1",
		"AGENT_BROWSER_DEFAULT_TIMEOUT=5000",
		"AGENT_BROWSER_SCREENSHOT_DIR=/tmp/session",
		"AGENT_BROWSER_NO_VERSION_CHECK=1",
	}
	if !slices.Equal(env, want) {
		t.Errorf("env = %q\nwant %q", env, want)
	}
	if got := readBinding(t, b, "abcd2345"); got != (agentBinding{Pinned: true}) {
		t.Errorf("binding = %+v, want pinned to no tab", got)
	}
	if info, _ := os.Stat(b.agentDir()); info.Mode().Perm() != 0o700 {
		t.Errorf("the socket folder is %v", info.Mode().Perm())
	}
}

func TestAgentEnvNeedsAgentBrowser(t *testing.T) {
	b := New(t.TempDir(), func(string, any) {})
	b.binary, b.agentBrowser = "/chrome", ""
	if _, err := b.AgentEnv("p/abcd2345", "/tmp/session"); err == nil || b.AgentReady() {
		t.Errorf("without agent-browser: ready %v, AgentEnv error %v", b.AgentReady(), err)
	}
}

func TestOpenCloseAndForgetBindTheSessionsAgentBrowser(t *testing.T) {
	b, ctx := newTestBrowsers(t)
	log := fakeAgentBrowser(t, b)
	if _, err := b.AgentEnv("p/abcd2345", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	stop := "abcd2345 " + b.agentDir() + " close"

	if _, err := b.Open(ctx, "p/abcd2345", "about:blank"); err != nil {
		t.Fatal(err)
	}
	pg, err := b.tab("p/abcd2345").activePage()
	if err != nil {
		t.Fatal(err)
	}
	if got := readBinding(t, b, "abcd2345"); got != (agentBinding{TargetID: pg.targetID, Pinned: true}) {
		t.Errorf("binding after Open = %+v, want the tab %s", got, pg.targetID)
	}
	if got := calls(t, log); !slices.Equal(got, []string{stop}) {
		t.Errorf("calls after Open = %q, want the daemon stopped so it reads the binding again", got)
	}

	b.Close(ctx, "p/abcd2345")
	if got := readBinding(t, b, "abcd2345"); got != (agentBinding{Pinned: true}) {
		t.Errorf("binding after Close = %+v, want pinned to no tab", got)
	}
	if got := calls(t, log); !slices.Equal(got, []string{stop, stop}) {
		t.Errorf("calls after Close = %q", got)
	}

	// The daemon's own files go too.
	if err := os.WriteFile(filepath.Join(b.agentDir(), "abcd2345.log"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b.agentDir(), "zzzz2345.target"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	b.Forget(ctx, "p/abcd2345")
	if files, _ := os.ReadDir(b.agentDir()); len(files) != 1 || files[0].Name() != "zzzz2345.target" {
		t.Errorf("files after Forget = %v, want only the other session's", files)
	}
}

func TestASessionWithoutAgentBrowserIsLeftAlone(t *testing.T) {
	b, ctx := newTestBrowsers(t)
	log := fakeAgentBrowser(t, b)
	if _, err := b.Open(ctx, "p/abcd2345", "about:blank"); err != nil {
		t.Fatal(err)
	}
	b.Forget(ctx, "p/abcd2345")
	if _, err := os.Stat(log); err == nil {
		t.Errorf("agent-browser ran for a session that never got it: %q", calls(t, log))
	}
	if _, err := os.Stat(b.agentDir()); err == nil {
		t.Error("Close made the agent-browser folder")
	}
}

// TestAgentBrowserDrivesOnlyTheSessionsTab runs the agent-browser that AGENTOS_AGENT_BROWSER names, as make
// agent-browser fetches it: the binding file is agent-browser's own, so a new version must pass this before it ships.
func TestAgentBrowserDrivesOnlyTheSessionsTab(t *testing.T) {
	real := os.Getenv("AGENTOS_AGENT_BROWSER")
	if out, err := exec.Command(real, "--version").Output(); real == "" || err != nil || !strings.HasPrefix(string(out), "agent-browser ") {
		t.Skip("AGENTOS_AGENT_BROWSER names no agent-browser: run make agent-browser and set it to bin/agent-browser")
	}
	// The socket path may not pass 103 bytes, and the test's own temp folder is too deep for it.
	data, err := os.MkdirTemp("/tmp", "agentos-ab-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(data) })
	b := New(data, func(string, any) {})
	if !b.Available() {
		t.Skip("no Chrome for Testing installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(func() { cancel(); b.CloseAll() })

	ids := []string{"p/aaaa2345", "p/bbbb2345"}
	envs := map[string][]string{}
	for _, id := range ids {
		env, err := b.AgentEnv(id, t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		envs[id] = env
		t.Cleanup(func() { b.Forget(context.Background(), id) })
	}
	run := func(id string, args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, real, args...)
		cmd.Env = append(os.Environ(), envs[id]...)
		out, err := cmd.CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}

	// With no window the project's browser is not running, and agent-browser fails rather than start one of its own.
	if out, err := run(ids[0], "get", "url"); err == nil {
		t.Errorf("before any window: %q", out)
	}
	open := func(i int) {
		t.Helper()
		if _, err := b.Open(ctx, ids[i], "data:text/html,<title>page "+strconv.Itoa(i)+"</title>"); err != nil {
			t.Fatal(err)
		}
	}
	open(1)
	if out, err := run(ids[0], "get", "url"); err == nil || !strings.Contains(out, "tab_gone") {
		t.Errorf("before its window, beside another session's: %q, %v; want tab_gone", out, err)
	}
	open(0)
	for i, id := range ids {
		if out, err := run(id, "get", "title"); err != nil || !strings.Contains(out, "page "+strconv.Itoa(i)) {
			t.Errorf("%s get title = %q, %v; want its own page", id, out, err)
		}
	}
	if out, err := run(ids[0], "open", "data:text/html,<title>moved</title>"); err != nil {
		t.Fatalf("open = %q, %v", out, err)
	}
	if out, _ := run(ids[1], "get", "title"); !strings.Contains(out, "page 1") {
		t.Errorf("the other session's tab = %q, want it untouched", out)
	}
	if st := b.State(ids[0]); st.Title != "moved" {
		t.Errorf("the session's state = %+v, want the page agent-browser opened", st)
	}

	b.Close(ctx, ids[0])
	if out, err := run(ids[0], "get", "url"); err == nil || !strings.Contains(out, "tab_gone") {
		t.Errorf("after its window closed: %q, %v; want tab_gone", out, err)
	}
	name, _ := session.ParseName(ids[1])
	if _, err := os.Stat(filepath.Join(b.agentDir(), name.Token+".sock")); err != nil {
		t.Errorf("the other session's daemon has no socket: %v", err)
	}
}
