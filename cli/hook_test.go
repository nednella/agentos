package cli

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/session"
)

// hookSetup listens on a socket in a short temp dir and points the hook at it.
func hookSetup(t *testing.T) (dir string, received func() []session.Record) {
	t.Helper()
	dir, err := os.MkdirTemp("", "aoshook")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	var mu sync.Mutex
	var got []session.Record
	ln, err := bus.Listen(bus.SocketPath(dir), func(rec session.Record) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, rec)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(ln.Close)
	t.Setenv("AGENTOS_SESSION", "demo/1")
	t.Setenv("AGENTOS_SOCKET", bus.SocketPath(dir))
	return dir, func() []session.Record {
		mu.Lock()
		defer mu.Unlock()
		return append([]session.Record(nil), got...)
	}
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestHookSavesAndReportsTheNewState(t *testing.T) {
	dir, received := hookSetup(t)

	reportHook("PreToolUse", strings.NewReader(`{"tool_name":"Bash","tool_input":{"command":"ls"}}`))
	rec, err := bus.ReadState(dir, "demo/1")
	if err != nil || rec.State != session.Working || rec.Detail != "Bash ls" {
		t.Fatalf("saved state = %+v, %v", rec, err)
	}
	waitFor(t, "the app to hear of it", func() bool { return len(received()) == 1 })

	reportHook("Stop", strings.NewReader(`{}`))
	if rec, _ = bus.ReadState(dir, "demo/1"); rec.State != session.Idle {
		t.Errorf("state after Stop = %s", rec.State)
	}
	waitFor(t, "the second report", func() bool { return len(received()) == 2 })
}

func TestHookSaysNothingWhenTheStateHasNotChanged(t *testing.T) {
	_, received := hookSetup(t)
	reportHook("SessionStart", strings.NewReader(`{}`))
	waitFor(t, "the first report", func() bool { return len(received()) == 1 })
	// An idle reminder after a start changes nothing, so it is not sent again.
	reportHook("Notification", strings.NewReader(`{"notification_type":"idle_prompt"}`))
	time.Sleep(150 * time.Millisecond)
	if n := len(received()); n != 1 {
		t.Errorf("%d reports, want 1", n)
	}
}

func TestHookIgnoresWhatIsNotASession(t *testing.T) {
	dir, received := hookSetup(t)
	t.Setenv("AGENTOS_SESSION", "not a session")
	reportHook("Stop", strings.NewReader(`{}`))
	t.Setenv("AGENTOS_SESSION", "demo/1")
	t.Setenv("AGENTOS_SOCKET", "")
	reportHook("Stop", strings.NewReader(`{}`))
	time.Sleep(100 * time.Millisecond)
	if len(bus.ReadAll(dir)) != 0 || len(received()) != 0 {
		t.Error("the hook reported for something that is not a session")
	}
}

func TestHookWithoutAnAppStillSavesTheState(t *testing.T) {
	dir, err := os.MkdirTemp("", "aoshook")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	t.Setenv("AGENTOS_SESSION", "demo/1")
	t.Setenv("AGENTOS_SOCKET", bus.SocketPath(dir))
	reportHook("UserPromptSubmit", strings.NewReader(`{"prompt":"go"}`))
	if rec, _ := bus.ReadState(dir, "demo/1"); rec.State != session.Working {
		t.Errorf("state = %+v", rec)
	}
}

func TestHookCommandIsQuiet(t *testing.T) {
	hookSetup(t)
	var out bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetIn(strings.NewReader(`{}`))
	root.SetArgs([]string{"hook", "Stop"})
	if err := root.Execute(); err != nil || out.Len() != 0 {
		t.Errorf("hook printed %q, %v", out.String(), err)
	}
}

func TestLateHookDoesNotUndoALaterEvent(t *testing.T) {
	dir, _ := hookSetup(t)
	t0 := time.Now()
	reportHookAt("Stop", strings.NewReader(`{}`), t0.Add(time.Second))
	reportHookAt("PostToolUse", strings.NewReader(`{"tool_name":"Bash"}`), t0)
	if rec, _ := bus.ReadState(dir, "demo/1"); rec.State != session.Idle || rec.Event != "Stop" {
		t.Errorf("a late PostToolUse overwrote Stop: %+v", rec)
	}
}

func TestConcurrentHooksKeepTheLatestEvent(t *testing.T) {
	dir, _ := hookSetup(t)
	for i := range 40 {
		t0 := time.Now().Add(time.Duration(i) * time.Minute)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			reportHookAt("PostToolUse", strings.NewReader(`{"tool_name":"Bash"}`), t0)
		}()
		go func() {
			defer wg.Done()
			reportHookAt("Stop", strings.NewReader(`{}`), t0.Add(time.Second))
		}()
		wg.Wait()
		if rec, _ := bus.ReadState(dir, "demo/1"); rec.State != session.Idle || rec.Event != "Stop" {
			t.Fatalf("round %d: state = %+v, want the Stop", i, rec)
		}
	}
}
