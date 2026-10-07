package browser

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStrayBrowserIsReplaced(t *testing.T) {
	dir, err := os.MkdirTemp("", "aosb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	first := New(dir, func(string, any) {})
	if !first.Available() {
		t.Skip("no Chromium based browser installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	defer first.CloseAll()
	if _, err := first.Open(ctx, "p/1", "about:blank"); err != nil {
		t.Fatal(err)
	}
	// The app was killed: a new one starts on the same profile while the old browser still runs.
	second := New(dir, func(string, any) {})
	defer second.CloseAll()
	if _, err := second.Open(ctx, "p/1", "about:blank"); err != nil {
		t.Fatalf("the restarted app could not get a browser: %v", err)
	}
}

func TestNoBrowserFound(t *testing.T) {
	b := New(t.TempDir(), func(string, any) {})
	b.binary = ""
	if b.Available() {
		t.Error("available without a browser")
	}
	st, err := b.Open(context.Background(), "p/1", "")
	if err == nil || !strings.Contains(st.Error, "no Brave") {
		t.Errorf("Open = %+v, %v", st, err)
	}
	if s := b.State("p/1"); s.Open || s.ID != "p/1" {
		t.Errorf("state = %+v", s)
	}
}

func TestLaunchArgsPerMode(t *testing.T) {
	b := New(t.TempDir(), func(string, any) {})
	b.headless = true
	if args := b.launchArgs("/p", 24680); !slices.Contains(args, "--headless=new") || !slices.Contains(args, "--hide-scrollbars") {
		t.Errorf("headless args = %v", args)
	}
	b.headless = false
	args := b.launchArgs("/p", 24680)
	for _, want := range []string{"--remote-debugging-port=24680", "--user-data-dir=/p", "--disable-backgrounding-occluded-windows", "--disable-renderer-backgrounding", "--no-startup-window"} {
		if !slices.Contains(args, want) {
			t.Errorf("headed args lack %s: %v", want, args)
		}
	}
	if slices.Contains(args, "--headless=new") || slices.Contains(args, "--hide-scrollbars") {
		t.Errorf("headed args = %v", args)
	}
}

func TestWebAddress(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"example.com", "https://example.com", true},
		{"example.com/a?b=1", "https://example.com/a?b=1", true},
		{"https://example.com", "https://example.com", true},
		{"localhost:3000", "http://localhost:3000", true},
		{"127.0.0.1:8080/x", "http://127.0.0.1:8080/x", true},
		{"about:blank", "about:blank", true},
		{"  spaced.dev  ", "https://spaced.dev", true},
		{"javascript:alert(1)", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, err := webAddress(tt.in)
		if (err == nil) != tt.ok || got != tt.want {
			t.Errorf("webAddress(%q) = %q, %v", tt.in, got, err)
		}
	}
}

func newTestBrowsers(t *testing.T) (*Browsers, context.Context) {
	t.Helper()
	b := New(t.TempDir(), func(string, any) {})
	if !b.Available() {
		t.Skip("no Chromium based browser installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	t.Cleanup(func() { cancel(); b.CloseAll() })
	return b, ctx
}

func TestConcurrentOpensShareOneTab(t *testing.T) {
	b, ctx := newTestBrowsers(t)
	var wg sync.WaitGroup
	states := make([]BrowserState, 2)
	for i := range states {
		wg.Go(func() {
			st, err := b.Open(ctx, "p/1", "about:blank")
			if err != nil {
				t.Errorf("Open %d: %v", i, err)
			}
			states[i] = st
		})
	}
	wg.Wait()
	if !states[0].Open || !states[1].Open || states[0].ID != states[1].ID {
		t.Errorf("the two opens returned %+v and %+v", states[0], states[1])
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.tabs) != 1 || b.procs["p"] == nil || b.procs["p"].tabs != 1 {
		t.Errorf("%d tabs, browser counts %d", len(b.tabs), b.procs["p"].tabs)
	}
}

func TestOpenReportsAFirstPageThatFails(t *testing.T) {
	b, ctx := newTestBrowsers(t)
	st, err := b.Open(ctx, "p/1", "http://127.0.0.1:1/")
	if err == nil || !strings.Contains(err.Error(), "could not open") {
		t.Errorf("Open of an address nobody serves = %v", err)
	}
	if !st.Open || !b.Has("p/1") {
		t.Errorf("the tab did not stay open: %+v", st)
	}
}

func TestPageTitleDropsTheLabel(t *testing.T) {
	tests := []struct{ prefix, title, want string }{
		{"[demo · 2 fix]", "[demo · 2 fix] Example", "Example"},
		{"[demo · 2 fix]", "[demo · 2 fix]", ""},
		{"[demo · 2 fix]", "https://example.com/", "https://example.com/"},
		{"[demo · 2 fix]", "[other] Example", "[other] Example"},
		{"", "Example", "Example"},
	}
	for _, tt := range tests {
		if got := (&tab{prefix: tt.prefix}).pageTitle(tt.title); got != tt.want {
			t.Errorf("pageTitle(%q) with prefix %q = %q, want %q", tt.title, tt.prefix, got, tt.want)
		}
	}
}

func TestTitleScriptQuotesTheLabel(t *testing.T) {
	if script := titleScript(`[a "b" \ c]`); !strings.Contains(script, `"[a \"b\" \\ c]"`) {
		t.Errorf("label not quoted in %s", script)
	}
}

func TestWindowLabel(t *testing.T) {
	tests := []struct{ in, want string }{
		{"demo · 1 Fix", "[demo · 1 Fix]"},
		{"demo · 1 Two  spaces", "[demo · 1 Two spaces]"},
		{"demo · 1 new\nline\t!", "[demo · 1 new line !]"},
		{"  padded  ", "[padded]"},
		{"", "[]"},
	}
	for _, tt := range tests {
		if got := windowLabel(tt.in); got != tt.want {
			t.Errorf("windowLabel(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestConsoleShownKeepsTheLastLinesAfterTheAgentDrains(t *testing.T) {
	var mu sync.Mutex
	events := 0
	b := New(t.TempDir(), func(event string, _ any) {
		mu.Lock()
		defer mu.Unlock()
		if event == "browser:state" {
			events++
		}
	})
	tb := &tab{b: b, id: "demo/1"}
	if got := tb.snapshotState().Console; got == nil || len(got) != 0 {
		t.Fatalf("Console = %#v, want an empty list", got)
	}
	for i := range 25 {
		tb.logConsole(strconv.Itoa(i))
	}
	if drained := tb.drainConsole(); len(drained) != 25 {
		t.Fatalf("drained %d lines, want 25", len(drained))
	}
	got := tb.snapshotState().Console
	if len(got) != consoleShown || got[0] != "5" || got[consoleShown-1] != "24" {
		t.Fatalf("Console = %v, want lines 5 to 24", got)
	}
	time.Sleep(consolePublishGap * 3)
	mu.Lock()
	defer mu.Unlock()
	if events != 1 {
		t.Errorf("published %d state events for 25 lines, want 1", events)
	}
}

func TestPortIsChosenOnceAndKept(t *testing.T) {
	dir := t.TempDir()
	first := New(dir, func(string, any) {}).Port("demo")
	if first < portMin || first > portMax {
		t.Fatalf("port = %d, want %d to %d", first, portMin, portMax)
	}
	if again := New(dir, func(string, any) {}).Port("demo"); again != first {
		t.Errorf("a new app got port %d, want the saved %d", again, first)
	}
	if other := New(dir, func(string, any) {}).Port("other"); other == 0 {
		t.Error("a second project got no port")
	}
	if _, err := os.Stat(filepath.Join(dir, "demo", "browser-port")); err != nil {
		t.Errorf("the port is not saved beside the profile: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "demo", "browser")); err == nil {
		t.Error("asking for the port made the browser profile")
	}
}

func TestLaunchPortReplacesAPortSomethingElseHolds(t *testing.T) {
	b := New(t.TempDir(), func(string, any) {})
	saved := b.Port("demo")
	if got, err := b.launchPort("demo"); err != nil || got != saved {
		t.Fatalf("launchPort with the saved port free = %d, %v; want %d", got, err, saved)
	}
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(saved)))
	if err != nil {
		t.Skipf("port %d taken before the test could take it: %v", saved, err)
	}
	defer l.Close()
	got, err := b.launchPort("demo")
	if err != nil || got == saved {
		t.Fatalf("launchPort with the saved port taken = %d, %v", got, err)
	}
	if b.Port("demo") != got {
		t.Error("the new port was not saved")
	}
}

func TestBrowserListensOnItsSavedPort(t *testing.T) {
	b, ctx := newTestBrowsers(t)
	port := b.Port("p")
	if _, err := b.Open(ctx, "p/1", "about:blank"); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/json/version")
	if err != nil {
		t.Fatalf("nothing listens on the saved port %d: %v", port, err)
	}
	resp.Body.Close()
	if b.Port("p") != port {
		t.Errorf("the port changed from %d to %d once the browser ran", port, b.Port("p"))
	}
}

func dialogLines(b *Browsers, id string) string {
	return strings.Join(b.State(id).Console, "\n")
}

func TestDialogLeftOpenIsDismissed(t *testing.T) {
	const page = `<!doctype html><title>Dialogs</title>`
	tests := []struct{ name, js, result, line string }{
		{"confirm", `confirm("Delete?")`, "[false,true]", `dialog dismissed after 100ms: confirm "Delete?"`},
		{"prompt", `prompt("Name?", "x")`, "[null,true]", `dialog dismissed after 100ms: prompt "Name?"`},
		{"alert", `alert("Saved")`, "[null,true]", `dialog dismissed after 100ms: alert "Saved"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, ctx := newTestBrowsers(t)
			b.SetDialogWait(100 * time.Millisecond)
			if _, err := b.Open(ctx, "p/1", "data:text/html,"+page); err != nil {
				t.Fatal(err)
			}
			// The dialog opens from a timer, so the call that starts it is not blocked by it.
			if _, err := b.Eval(ctx, "p/1", `setTimeout(() => { window.answer = `+tt.js+`; window.after = true }, 0)`); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(10 * time.Second)
			for !strings.Contains(dialogLines(b, "p/1"), tt.line) {
				if time.Now().After(deadline) {
					t.Fatalf("no line %q in the console: %q", tt.line, dialogLines(b, "p/1"))
				}
				time.Sleep(20 * time.Millisecond)
			}
			if got, err := b.Eval(ctx, "p/1", `JSON.stringify([window.answer, window.after])`); err != nil || got != strconv.Quote(tt.result) {
				t.Errorf("page state after the dismissal = %s, %v; want %s", got, err, tt.result)
			}
		})
	}
}

func TestDialogAnsweredInTimeIsLeftAlone(t *testing.T) {
	b, ctx := newTestBrowsers(t)
	b.SetDialogWait(500 * time.Millisecond)
	if _, err := b.Open(ctx, "p/1", "about:blank"); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Eval(ctx, "p/1", `setTimeout(() => { window.answer = confirm("Keep?") }, 0)`); err != nil {
		t.Fatal(err)
	}
	tb := b.tab("p/1")
	for range 100 {
		tb.mu.Lock()
		open := tb.dialog != nil
		tb.mu.Unlock()
		if open {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := tb.call(ctx, "Page.handleJavaScriptDialog", map[string]any{"accept": true}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(900 * time.Millisecond)
	if got, err := b.Eval(ctx, "p/1", `window.answer`); err != nil || got != "true" {
		t.Errorf("answer = %s, %v; want true", got, err)
	}
	if lines := dialogLines(b, "p/1"); strings.Contains(lines, "dismissed") {
		t.Errorf("a dialog answered in time was reported dismissed: %q", lines)
	}
}
