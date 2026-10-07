package browser_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/sessions"
)

const demoPage = `<!doctype html><html><head><title>Demo</title></head><body>
<h1>Demo page</h1>
<a href="/second" target="_blank">Second page</a>
<div style="height:3000px"></div>
</body></html>`

func demoServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, demoPage)
	})
	mux.HandleFunc("/second", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<!doctype html><title>Second</title><h1>Second page header</h1><p>You made it.</p>`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func browserHarness(t *testing.T) (*apptest.Harness, sessions.Session) {
	t.Helper()
	h := newHarness(t)
	if !h.App.Browsers().Available() {
		t.Skip("no Chromium based browser installed")
	}
	s, err := h.NewSession("web", "")
	if err != nil {
		t.Fatal(err)
	}
	return h, s
}

func TestBrowserUserScreenshot(t *testing.T) {
	h, s := browserHarness(t)
	srv := demoServer(t)
	state, err := h.BrowserOpen(s.ID, srv.URL)
	if err != nil || !state.Open {
		t.Fatalf("BrowserOpen = %+v, %v", state, err)
	}
	eventually(t, "title", func() bool { return h.BrowserState(s.ID).Title == "Demo" })

	before := h.Rec.Count("attention")
	item, err := h.BrowserScreenshot(s.ID, "mine")
	if err != nil || item.Source != "user" || item.Caption != "mine" {
		t.Fatalf("BrowserScreenshot = %+v, %v", item, err)
	}
	if h.Rec.Count("attention") != before {
		t.Error("the user's own screenshot raised attention")
	}
	if err := h.DeleteEvidence(s.ID, item.ID); err != nil || len(h.Evidence(s.ID)) != 0 {
		t.Errorf("DeleteEvidence: %v", err)
	}
}
