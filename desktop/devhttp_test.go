package main

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCallRunsABoundMethod(t *testing.T) {
	h := newHarness(t)
	got, err := call(h.app, "NewSession", []byte(`["via call",""]`))
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := got.(Session); !ok || s.Title != "via call" {
		t.Errorf("result = %+v", got)
	}
	if got, err = call(h.app, "Snapshot", []byte(`[]`)); err != nil || len(got.(Snapshot).Sessions) != 1 {
		t.Errorf("Snapshot = %+v, %v", got, err)
	}
	if got, err = call(h.app, "RenameSession", []byte(`["main/1","renamed"]`)); err != nil || got != nil {
		t.Errorf("a method with no result gave %v, %v", got, err)
	}
}

func TestCallReportsProblems(t *testing.T) {
	h := newHarness(t)
	for _, tt := range []struct{ method, body string }{
		{"NoSuchMethod", `[]`},
		{"NewSession", `["only one argument"]`},
		{"NewSession", `[1,2]`},
		{"Snapshot", `not json`},
		{"KillSession", `["main/99"]`}, // the method itself fails
	} {
		if got, err := call(h.app, tt.method, []byte(tt.body)); err == nil {
			t.Errorf("%s %s succeeded: %v", tt.method, tt.body, got)
		}
	}
}

func TestEventHubReachesEveryTab(t *testing.T) {
	hub := &eventHub{subs: map[chan []byte]struct{}{}}
	srv := httptest.NewServer(http.HandlerFunc(hub.serve))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	lines := make(chan string, 4)
	for range 2 {
		req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		go func() {
			sc := bufio.NewScanner(resp.Body)
			for sc.Scan() {
				if strings.HasPrefix(sc.Text(), "data: ") {
					lines <- sc.Text()
				}
			}
		}()
	}
	// Wait until both tabs are subscribed, then emit.
	deadline := time.Now().Add(3 * time.Second)
	for {
		hub.mu.Lock()
		n := len(hub.subs)
		hub.mu.Unlock()
		if n == 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("tabs did not subscribe")
		}
		time.Sleep(10 * time.Millisecond)
	}
	hub.emit("sessions", []string{"a"})
	for range 2 {
		select {
		case line := <-lines:
			var m struct {
				Event   string
				Payload []string
			}
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &m); err != nil || m.Event != "sessions" || m.Payload[0] != "a" {
				t.Errorf("event line %q: %v", line, err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("a tab did not get the event")
		}
	}
}

func TestBrowserModeServesTheAppAndTheShim(t *testing.T) {
	h := newHarness(t)
	hub := &eventHub{subs: map[chan []byte]struct{}{}}
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	app := newApp(cfg, host{emit: hub.emit, clipboard: func(string) {}})
	assets, err := fs.Sub(assetsFS, assetsRoot)
	if err != nil {
		t.Fatal(err)
	}
	addr := "127.0.0.1:" + freePort(t)
	go func() { _ = serveHTTP(app, hub, assets, addr) }()
	base := "http://" + addr
	eventually(t, "the server", func() bool {
		resp, err := http.Get(base + "/__shim.js")
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == 200
	})

	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(page), `<script src="/__shim.js"></script>`) {
		t.Errorf("the page does not load the shim:\n%s", page)
	}

	resp, err = http.Post(base+"/__call/Snapshot", "application/json", strings.NewReader(`[]`))
	if err != nil {
		t.Fatal(err)
	}
	var snap Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil || snap.Project.Dir != h.dir {
		t.Errorf("Snapshot over HTTP = %+v, %v", snap, err)
	}
	resp.Body.Close()

	resp, err = http.Post(base+"/__call/Nope", "application/json", strings.NewReader(`[]`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("an unknown method gave %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	return port
}
