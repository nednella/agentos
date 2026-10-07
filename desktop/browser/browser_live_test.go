package browser_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/browser"
	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/sessions"
)

const demoPage = `<!doctype html><html><head><title>Demo</title><style>
body{margin:0;font:16px sans-serif} #pad{position:absolute;left:300px;top:400px;width:200px;height:100px;background:#ccd}
#spacer{height:3000px}</style></head><body>
<h1>Demo page</h1>
<input id="q" aria-label="Search" onkeydown="if(event.key==='Enter')document.getElementById('entered').textContent='enter:'+this.value">
<p id="entered"></p>
<a href="/second" target="_blank">Second page</a>
<div id="pad" onmousedown="window.__down=[event.clientX,event.clientY,event.button]"></div>
<div id="spacer"></div>
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

func frames(h *apptest.Harness, id string) int { return h.Rec.CountFrames(id) }

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

func TestBrowserLiveView(t *testing.T) {
	h, s := browserHarness(t)
	srv := demoServer(t)
	state, err := h.BrowserOpen(s.ID, srv.URL)
	if err != nil || !state.Open {
		t.Fatalf("BrowserOpen = %+v, %v", state, err)
	}
	eventually(t, "title", func() bool { return h.BrowserState(s.ID).Title == "Demo" })
	eval := func(js string) string {
		out, err := h.App.Browsers().Eval(context.Background(), s.ID, js)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	flip := func(i int) { eval(fmt.Sprintf("document.body.style.background='#%06x'", (i*997331)%0xffffff)) }

	if n := frames(h, s.ID); n != 0 {
		t.Fatalf("%d frames before the view was on", n)
	}
	flip(1)
	time.Sleep(300 * time.Millisecond)
	if n := frames(h, s.ID); n != 0 {
		t.Errorf("%d frames while hidden", n)
	}

	if err := h.BrowserView(s.ID, true); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the first frame", func() bool { return frames(h, s.ID) >= 1 })
	for i := 2; i < 6; i++ {
		flip(i)
		time.Sleep(60 * time.Millisecond)
	}
	eventually(t, "more frames from changes", func() bool { return frames(h, s.ID) >= 3 })
	last := h.Rec.LastFrame()
	if data, _ := last["data"].(string); len(data) < 100 || last["width"] != 1280 || last["height"] != 800 {
		t.Errorf("frame = width %v height %v, %d bytes", last["width"], last["height"], len(data))
	}

	t.Run("resize", func(t *testing.T) {
		if err := h.BrowserResize(s.ID, 800, 500); err != nil {
			t.Fatal(err)
		}
		if got := eval("innerWidth + 'x' + innerHeight"); got != `"800x500"` {
			t.Errorf("viewport = %s", got)
		}
		eventually(t, "a frame of the new size", func() bool { return h.Rec.HasFrame(800, 500) })
	})

	if err := h.BrowserView(s.ID, false); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	stopped := frames(h, s.ID)
	for i := 10; i < 15; i++ {
		flip(i)
		time.Sleep(60 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	if frames(h, s.ID) != stopped {
		t.Errorf("frames kept coming while hidden: %d -> %d", stopped, frames(h, s.ID))
	}

	t.Run("input", func(t *testing.T) {
		click := func(x, y float64) {
			for _, action := range []string{"move", "down", "up"} {
				if err := h.BrowserInput(s.ID, browser.BrowserInput{Type: "mouse", Action: action, X: x, Y: y, Button: "left", ClickCount: 1}); err != nil {
					t.Fatal(err)
				}
			}
		}
		click(350, 450)
		if got := eval("JSON.stringify(window.__down)"); got != `"[350,450,0]"` {
			t.Errorf("mouse down seen by the page = %s", got)
		}

		click(150, 700) // an empty spot
		eval("document.getElementById('q').focus()")
		for _, k := range []browser.BrowserInput{
			{Key: "h", Code: "KeyH", Text: "h"},
			{Key: "i", Code: "KeyI", Text: "i"},
			{Key: "!", Code: "Digit1", Text: "!", Modifiers: 8},
			{Key: "Backspace", Code: "Backspace"},
		} {
			for _, action := range []string{"down", "up"} {
				k.Type, k.Action = "key", action
				if err := h.BrowserInput(s.ID, k); err != nil {
					t.Fatal(err)
				}
			}
		}
		if got := eval("document.getElementById('q').value"); got != `"hi"` {
			t.Errorf("typed value = %s", got)
		}
		if err := h.BrowserInput(s.ID, browser.BrowserInput{Type: "paste", Text: " pasted"}); err != nil {
			t.Fatal(err)
		}
		for _, action := range []string{"down", "up"} {
			if err := h.BrowserInput(s.ID, browser.BrowserInput{Type: "key", Action: action, Key: "Enter", Code: "Enter"}); err != nil {
				t.Fatal(err)
			}
		}
		eventually(t, "Enter reaching the page", func() bool { return eval("document.getElementById('entered').textContent") == `"enter:hi pasted"` })

		if err := h.BrowserInput(s.ID, browser.BrowserInput{Type: "wheel", X: 400, Y: 300, DeltaY: 700}); err != nil {
			t.Fatal(err)
		}
		eventually(t, "the wheel scrolling the page", func() bool { return eval("Math.round(scrollY)") == "700" })
		if err := h.BrowserInput(s.ID, browser.BrowserInput{Type: "nope"}); err == nil {
			t.Error("an unknown input type was accepted")
		}
	})

	t.Run("user screenshot", func(t *testing.T) {
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
	})
}
