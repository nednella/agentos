package devhttp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/nednella/agentos/desktop/internal/app"
	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/projects"
	"github.com/nednella/agentos/desktop/sessions"
)

func TestCallRunsABoundMethod(t *testing.T) {
	h := apptest.New(t)
	got, err := call(h.App.Services(), "sessions.Service", "NewSession", []byte(`["via call",""]`))
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := got.(sessions.Session); !ok || s.Title != "via call" {
		t.Errorf("result = %+v", got)
	}
	if got, err = call(h.App.Services(), "projects.Service", "Snapshot", []byte(`[]`)); err != nil || len(got.(projects.Snapshot).Sessions) != 1 {
		t.Errorf("Snapshot = %+v, %v", got, err)
	}
	if got, err = call(h.App.Services(), "sessions.Service", "RenameSession", []byte(`["main/1","renamed"]`)); err != nil || got != nil {
		t.Errorf("a method with no result gave %v, %v", got, err)
	}
}

func TestCallReportsProblems(t *testing.T) {
	h := apptest.New(t)
	for _, tt := range []struct{ method, body string }{
		{"sessions.Service/NoSuchMethod", `[]`},
		{"sessions.Service/NewSession", `["only one argument"]`},
		{"sessions.Service/NewSession", `[1,2]`},
		{"projects.Service/Snapshot", `not json`},
		{"sessions.Service/KillSession", `["main/99"]`}, // the method itself fails
		{"nothing.Service/Snapshot", `[]`},
	} {
		svc, method, _ := strings.Cut(tt.method, "/")
		if got, err := call(h.App.Services(), svc, method, []byte(tt.body)); err == nil {
			t.Errorf("%s %s succeeded: %v", tt.method, tt.body, got)
		}
	}
}

func TestEventHubReachesEveryTab(t *testing.T) {
	hub := NewHub()
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
	hub.Emit("sessions", []string{"a"})
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
	h := apptest.New(t)
	hub := NewHub()
	cfg, err := app.Load()
	if err != nil {
		t.Fatal(err)
	}
	a := app.New(cfg, app.Host{Emit: hub.Emit, Clipboard: func(string) {}}, apptest.NoGH, apptest.NoStream, apptest.NoClaude)
	assets := fstest.MapFS{"index.html": {Data: []byte("<!doctype html><html><head></head><body>stub</body></html>")}}
	addr := "127.0.0.1:" + freePort(t)
	go func() {
		_ = Serve(Options{Services: a.Services(), Start: a.Start, Stop: a.Stop, Hub: hub, Assets: assets, Media: a.Media(), Addr: addr})
	}()
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

	resp, err = post(base+"/__call/projects.Service/Snapshot", nil)
	if err != nil {
		t.Fatal(err)
	}
	var snap projects.Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil || snap.Project.Dir != h.Dir {
		t.Errorf("Snapshot over HTTP = %+v, %v", snap, err)
	}
	resp.Body.Close()

	resp, err = post(base+"/__call/projects.Service/Nope", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("an unknown method gave %d", resp.StatusCode)
	}
	resp.Body.Close()

	for _, tt := range []struct {
		name    string
		headers map[string]string
		host    string
		want    int
	}{
		{"the shim's call", nil, "", http.StatusOK},
		{"same origin", map[string]string{"Origin": base}, "", http.StatusOK},
		{"a foreign origin", map[string]string{"Origin": "https://evil.example"}, "", http.StatusForbidden},
		{"a null origin", map[string]string{"Origin": "null"}, "", http.StatusForbidden},
		{"a foreign host name", nil, "evil.example", http.StatusForbidden},
		{"no marker header", map[string]string{callHeader: ""}, "", http.StatusForbidden},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := post(base+"/__call/projects.Service/Snapshot", tt.headers, withHost(tt.host))
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != tt.want {
				t.Errorf("status %d, want %d", resp.StatusCode, tt.want)
			}
		})
	}
	resp, err = http.Get(base + "/__shim.js")
	if err != nil {
		t.Fatal(err)
	}
	shim, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(shim), "'"+callHeader+"': '1'") {
		t.Error("the shim does not send the marker header")
	}
}

func withHost(host string) func(*http.Request) {
	return func(r *http.Request) {
		if host != "" {
			r.Host = host
		}
	}
}

// post sends an empty-argument call as the shim does, with headers changed by the given ones.
func post(url string, headers map[string]string, edits ...func(*http.Request)) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(`[]`))
	if err != nil {
		return nil, err
	}
	req.Header.Set(callHeader, "1")
	for k, v := range headers {
		if v == "" {
			req.Header.Del(k)
		} else {
			req.Header.Set(k, v)
		}
	}
	for _, edit := range edits {
		edit(req)
	}
	return http.DefaultClient.Do(req)
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

func TestMain(m *testing.M) { os.Exit(apptest.Main(m)) }

func eventually(t *testing.T, what string, cond func() bool) { apptest.Eventually(t, what, cond) }
