package bus

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/session"
)

func tempDir(t *testing.T) string {
	t.Helper()
	// Unix socket paths are short, so not t.TempDir().
	dir, err := os.MkdirTemp("", "aosbus")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestStateFiles(t *testing.T) {
	dir := tempDir(t)
	rec := session.Record{Session: "p/1", State: session.Waiting, Event: "Notification", At: time.Now().UTC().Truncate(time.Second), Detail: "needs you"}
	if got, err := ReadState(dir, "p/1"); err != nil || got != (session.Record{}) {
		t.Fatalf("ReadState of nothing = %+v, %v", got, err)
	}
	if err := WriteState(dir, rec); err != nil {
		t.Fatal(err)
	}
	if got, err := ReadState(dir, "p/1"); err != nil || got != rec {
		t.Errorf("ReadState = %+v, %v", got, err)
	}

	other := session.Record{Session: "q/2", State: session.Working}
	if err := WriteState(dir, other); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sessions", "junk.json"), []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	all := ReadAll(dir)
	if len(all) != 2 || all["p/1"] != rec || all["q/2"] != other {
		t.Errorf("ReadAll = %+v", all)
	}

	if err := RemoveState(dir, "p/1"); err != nil {
		t.Fatal(err)
	}
	if err := RemoveState(dir, "p/1"); err != nil {
		t.Errorf("removing a missing file failed: %v", err)
	}
	if got := ReadAll(dir); len(got) != 1 {
		t.Errorf("after remove: %+v", got)
	}
}

func TestDefaultDirHonoursTheOverride(t *testing.T) {
	t.Setenv("AGENTOS_STATE_DIR", "/somewhere")
	if dir, err := DefaultDir(); err != nil || dir != "/somewhere" {
		t.Errorf("DefaultDir = %q, %v", dir, err)
	}
	t.Setenv("AGENTOS_STATE_DIR", "")
	if dir, err := DefaultDir(); err != nil || !strings.HasSuffix(dir, filepath.Join(".local", "state", "agentos")) {
		t.Errorf("DefaultDir = %q, %v", dir, err)
	}
	if got := DirOf(SocketPath("/a/b")); got != "/a/b" {
		t.Errorf("DirOf = %q", got)
	}
}

func TestSendReachesTheListener(t *testing.T) {
	socket := SocketPath(tempDir(t))
	var mu sync.Mutex
	got := map[string]session.Record{}
	ln, err := Listen(socket, func(rec session.Record) {
		mu.Lock()
		defer mu.Unlock()
		got[rec.Session] = rec
	})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	ctx := context.Background()
	want := session.Record{Session: "p/1", State: session.Idle, Event: "Stop"}
	if err := Send(ctx, socket, want); err != nil {
		t.Fatal(err)
	}
	if err := Send(ctx, socket, session.Record{Session: "p/2", State: session.Working}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		n := len(got)
		mu.Unlock()
		if n == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("listener got %d records, want 2", n)
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if got["p/1"] != want {
		t.Errorf("record = %+v, want %+v", got["p/1"], want)
	}
}

func TestListenRefusesATakenSocket(t *testing.T) {
	socket := SocketPath(tempDir(t))
	ln, err := Listen(socket, func(session.Record) {})
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if _, err := Listen(socket, func(session.Record) {}); !errors.Is(err, ErrInUse) {
		t.Errorf("a second Listen = %v, want ErrInUse", err)
	}
}

func TestListenRefusesALongPath(t *testing.T) {
	long := filepath.Join(tempDir(t), strings.Repeat("x", 120), "agentos.sock")
	if _, err := Listen(long, func(session.Record) {}); err == nil {
		t.Error("a path over the limit was accepted")
	}
}

func TestSendWithoutAListenerFails(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := Send(ctx, SocketPath(tempDir(t)), session.Record{Session: "p/1"}); err == nil {
		t.Error("Send to nobody succeeded")
	}
}
