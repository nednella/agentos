package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

const demoPage = `<!doctype html><html><head><title>Demo</title><style>
body{margin:0;font:16px sans-serif} #pad{position:absolute;left:300px;top:400px;width:200px;height:100px;background:#ccd}
#spacer{height:3000px}</style></head><body>
<h1>Demo page</h1>
<form onsubmit="return false">
  <label for="email">Email</label><input id="email" type="email" placeholder="you@example.com">
  <label for="color">Color</label><select id="color"><option value="r">Red</option><option value="g">Green</option><option value="b">Blue</option></select>
  <label><input type="checkbox" id="sub"> Subscribe</label>
  <button id="save" onclick="document.getElementById('status').textContent='Saved: '+document.getElementById('email').value+' / '+document.getElementById('color').selectedOptions[0].text">Save</button>
  <button disabled>Nope</button>
</form>
<p id="status">not saved</p>
<input id="q" aria-label="Search" onkeydown="if(event.key==='Enter')document.getElementById('entered').textContent='enter:'+this.value">
<p id="entered"></p>
<a href="/second" target="_blank">Second page</a>
<button onclick="console.error('boom happened')">Boom</button>
<button onclick="fetch('/nope')">Fetch</button>
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

// cli runs the agentos command line as a session would, against the app under test.
func (h *harness) cli(t *testing.T, session string, args ...string) (string, error) {
	t.Helper()
	return h.cliWith(t, append(os.Environ(), "AGENTOS_SESSION="+session), args...)
}

func (h *harness) cliWith(t *testing.T, env []string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(cliPath(t), args...)
	cmd.Env = append(slices.Clone(env), "AGENTOS_SOCKET="+filepath.Join(h.state, "agentos.sock"))
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	if err != nil {
		return strings.TrimSpace(out.String()), fmt.Errorf("%s", strings.TrimSpace(errOut.String()))
	}
	return strings.TrimSpace(out.String()), nil
}

func (h *harness) mustCLI(t *testing.T, session string, args ...string) string {
	t.Helper()
	out, err := h.cli(t, session, args...)
	if err != nil {
		t.Fatalf("agentos %s: %v", strings.Join(args, " "), err)
	}
	return out
}

func refOf(t *testing.T, snapshot, role, name string) string {
	t.Helper()
	m := regexp.MustCompile(`\[(e\d+)\] ` + regexp.QuoteMeta(role) + ` "` + regexp.QuoteMeta(name) + `"`).FindStringSubmatch(snapshot)
	if m == nil {
		t.Fatalf("no %s %q in the snapshot:\n%s", role, name, snapshot)
	}
	return m[1]
}

func (h *harness) frames(id string) int {
	h.rec.mu.Lock()
	defer h.rec.mu.Unlock()
	n := 0
	for _, e := range h.rec.events {
		if m, ok := e.payload.(map[string]any); ok && e.name == "browser:frame" && m["id"] == id {
			n++
		}
	}
	return n
}

func browserHarness(t *testing.T) (*harness, Session) {
	t.Helper()
	h := newHarness(t)
	if !h.app.browsers.Available() {
		t.Skip("no Chromium based browser installed")
	}
	s, err := h.app.NewSession("web", "")
	if err != nil {
		t.Fatal(err)
	}
	return h, s
}

func profileRunning(profile string) bool {
	return exec.Command("pgrep", "-f", profile).Run() == nil
}

func TestBrowserAgentCLI(t *testing.T) {
	h, s := browserHarness(t)
	srv := demoServer(t)
	profile := filepath.Join(h.state, "data", "main", "browser")

	out := h.mustCLI(t, s.ID, "browser", "open", srv.URL)
	if !strings.Contains(out, "Demo") || !strings.Contains(out, srv.URL) {
		t.Errorf("open printed %q", out)
	}
	if got, _ := h.session(s.ID); !got.Browser {
		t.Error("the session does not show its browser")
	}

	snap := h.mustCLI(t, s.ID, "browser", "snapshot")
	for _, want := range []string{"# Demo page", `textbox "Email"`, `placeholder="you@example.com"`, `select "Color"`, `value="Red"`, "options=Red | Green | Blue",
		`checkbox "Subscribe"`, `button "Save"`, `button "Nope" disabled`, `textbox "Search"`, `link "Second page" -> /second`, "not saved"} {
		if !strings.Contains(snap, want) {
			t.Errorf("snapshot lacks %q:\n%s", want, snap)
		}
	}

	email, color, save, sub := refOf(t, snap, "textbox", "Email"), refOf(t, snap, "select", "Color"), refOf(t, snap, "button", "Save"), refOf(t, snap, "checkbox", "Subscribe")
	h.mustCLI(t, s.ID, "browser", "type", email, "ned@example.com")
	h.mustCLI(t, s.ID, "browser", "select", color, "Green")
	h.mustCLI(t, s.ID, "browser", "click", sub)
	h.mustCLI(t, s.ID, "browser", "click", save)
	h.mustCLI(t, s.ID, "browser", "wait-for", "Saved: ned@example.com / Green")
	h.mustCLI(t, s.ID, "browser", "type", email, "new", "--append")
	if v := h.mustCLI(t, s.ID, "browser", "eval", "document.getElementById('email').value + '|' + document.getElementById('sub').checked"); v != `"ned@example.comnew|true"` {
		t.Errorf("form state = %s", v)
	}
	h.mustCLI(t, s.ID, "browser", "type", email, "fresh")
	if v := h.mustCLI(t, s.ID, "browser", "eval", "document.getElementById('email').value"); v != `"fresh"` {
		t.Errorf("replaced value = %s", v)
	}

	q := refOf(t, snap, "textbox", "Search")
	h.mustCLI(t, s.ID, "browser", "type", q, "hello")
	h.mustCLI(t, s.ID, "browser", "press", "Enter")
	h.mustCLI(t, s.ID, "browser", "wait-for", "enter:hello")

	if text := h.mustCLI(t, s.ID, "browser", "text", save); text != "Save" {
		t.Errorf("text of the button = %q", text)
	}
	if out := h.mustCLI(t, s.ID, "browser", "scroll", "down", "1000"); !strings.Contains(out, "scrolled to 1000") {
		t.Errorf("scroll = %q", out)
	}
	h.mustCLI(t, s.ID, "browser", "scroll", "up", "5000")
	h.mustCLI(t, s.ID, "browser", "hover", save)

	t.Run("console", func(t *testing.T) {
		h.mustCLI(t, s.ID, "browser", "click", refOf(t, snap, "button", "Boom"))
		h.mustCLI(t, s.ID, "browser", "click", refOf(t, snap, "button", "Fetch"))
		var out string
		eventually(t, "both console lines", func() bool {
			out += "\n" + h.mustCLI(t, s.ID, "browser", "console")
			return strings.Contains(out, "console.error: boom happened") && strings.Contains(out, "/nope -> 404")
		})
		if again := h.mustCLI(t, s.ID, "browser", "console"); !strings.Contains(again, "no console errors") {
			t.Errorf("console after reading = %q", again)
		}
	})

	t.Run("target blank link stays in the tab", func(t *testing.T) {
		h.mustCLI(t, s.ID, "browser", "click", refOf(t, snap, "link", "Second page"))
		h.mustCLI(t, s.ID, "browser", "wait-for", "Second page header")
		if got := h.mustCLI(t, s.ID, "browser", "url"); got != srv.URL+"/second\nSecond" {
			t.Errorf("url = %q", got)
		}
		eventually(t, "back flag", func() bool { return h.app.BrowserState(s.ID).CanGoBack })
		if got := h.mustCLI(t, s.ID, "browser", "back"); !strings.Contains(got, srv.URL+"/") {
			t.Errorf("back = %q", got)
		}
		if got := h.mustCLI(t, s.ID, "browser", "reload"); !strings.Contains(got, "Demo") {
			t.Errorf("reload = %q", got)
		}
	})

	t.Run("screenshot becomes evidence", func(t *testing.T) {
		before := h.rec.count("attention")
		path := h.mustCLI(t, s.ID, "browser", "screenshot", "--caption", "the demo page")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("the printed path %q: %v", path, err)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil || img.Bounds().Dx() != defaultViewWidth || img.Bounds().Dy() != defaultViewHeight {
			t.Errorf("screenshot = %v, %v", img, err)
		}
		items := h.app.Evidence(s.ID)
		if len(items) != 1 || items[0].Caption != "the demo page" || items[0].Source != "agent" || items[0].Kind != "image" ||
			!strings.HasPrefix(items[0].URL, "/media/main/evidence/") {
			t.Fatalf("evidence = %+v", items)
		}
		srvMedia := httptest.NewServer(h.app.mediaHandler())
		defer srvMedia.Close()
		if resp, err := http.Get(srvMedia.URL + items[0].URL); err != nil || resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/png" {
			t.Errorf("serving the screenshot: %v %v", resp, err)
		}
		if got, _ := h.session(s.ID); got.Evidence != 1 {
			t.Errorf("session evidence count = %d", got.Evidence)
		}
		if h.rec.count("attention") != before+1 || h.rec.count("evidence") == 0 {
			t.Error("no attention or evidence event for the agent's screenshot")
		}

		full := h.mustCLI(t, s.ID, "browser", "screenshot", "--full")
		if img, err := decodePNG(full); err != nil || img.Bounds().Dy() < 3000 {
			t.Errorf("full page screenshot: %v %v", img, err)
		}
		save = refOf(t, h.mustCLI(t, s.ID, "browser", "snapshot"), "button", "Save")
		element := h.mustCLI(t, s.ID, "browser", "screenshot", save)
		if img, err := decodePNG(element); err != nil || img.Bounds().Dx() > 200 || img.Bounds().Dx() < 20 {
			t.Errorf("element screenshot: %v %v", img, err)
		}
	})

	t.Run("errors", func(t *testing.T) {
		for _, args := range [][]string{
			{"browser", "click", "e999"},
			{"browser", "click", "nope"},
			{"browser", "select", color, "Purple"},
			{"browser", "wait-for", "never there", "--timeout", "300"},
			{"browser", "frobnicate"},
			{"browser", "type", "e1"},
			{"browser", "eval", "throw new Error('bad js')"},
			{"browser", "open", "javascript:alert(1)"},
			{"browser", "open", "http://127.0.0.1:1"},
		} {
			if out, err := h.cli(t, s.ID, args...); err == nil {
				t.Errorf("agentos %s succeeded: %q", strings.Join(args, " "), out)
			}
		}
		if _, err := h.cli(t, s.ID, "browser", "click", "e999"); err == nil || !strings.Contains(err.Error(), "run snapshot again") {
			t.Errorf("unknown ref error = %v", err)
		}
		if out, err := h.cli(t, "", "browser", "url"); err == nil || !strings.Contains(err.Error(), "AGENTOS_SESSION") {
			t.Errorf("without a session: %q, %v", out, err)
		}
		if out := h.mustCLI(t, "", "browser", "help"); !strings.Contains(out, "snapshot") {
			t.Errorf("help = %q", out)
		}
	})

	t.Run("closing the tab ends the browser", func(t *testing.T) {
		h.app.browsers.grace = 300 * time.Millisecond
		h.app.BrowserClose(s.ID)
		if got, _ := h.session(s.ID); got.Browser {
			t.Error("the session still shows a browser")
		}
		if st := h.app.BrowserState(s.ID); st.Open {
			t.Errorf("state = %+v", st)
		}
		eventually(t, "the browser to stop", func() bool { return !profileRunning(profile) })
	})
}

func TestBrowserLiveView(t *testing.T) {
	h, s := browserHarness(t)
	srv := demoServer(t)
	state, err := h.app.BrowserOpen(s.ID, srv.URL)
	if err != nil || !state.Open {
		t.Fatalf("BrowserOpen = %+v, %v", state, err)
	}
	eventually(t, "title", func() bool { return h.app.BrowserState(s.ID).Title == "Demo" })
	eval := func(js string) string {
		out, err := h.app.browsers.agent(context.Background(), s.ID, "eval", []string{js}, nil)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	flip := func(i int) { eval(fmt.Sprintf("document.body.style.background='#%06x'", (i*997331)%0xffffff)) }

	if n := h.frames(s.ID); n != 0 {
		t.Fatalf("%d frames before the view was on", n)
	}
	flip(1)
	time.Sleep(300 * time.Millisecond)
	if n := h.frames(s.ID); n != 0 {
		t.Errorf("%d frames while hidden", n)
	}

	if err := h.app.BrowserView(s.ID, true); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the first frame", func() bool { return h.frames(s.ID) >= 1 })
	for i := 2; i < 6; i++ {
		flip(i)
		time.Sleep(60 * time.Millisecond)
	}
	eventually(t, "more frames from changes", func() bool { return h.frames(s.ID) >= 3 })
	h.rec.mu.Lock()
	var last map[string]any
	for _, e := range h.rec.events {
		if m, ok := e.payload.(map[string]any); ok && e.name == "browser:frame" {
			last = m
		}
	}
	h.rec.mu.Unlock()
	if data, _ := last["data"].(string); len(data) < 100 || last["width"] != defaultViewWidth || last["height"] != defaultViewHeight {
		t.Errorf("frame = width %v height %v, %d bytes", last["width"], last["height"], len(data))
	}

	t.Run("resize", func(t *testing.T) {
		if err := h.app.BrowserResize(s.ID, 800, 500); err != nil {
			t.Fatal(err)
		}
		if got := eval("innerWidth + 'x' + innerHeight"); got != `"800x500"` {
			t.Errorf("viewport = %s", got)
		}
		eventually(t, "a frame of the new size", func() bool {
			h.rec.mu.Lock()
			defer h.rec.mu.Unlock()
			for _, e := range h.rec.events {
				if m, ok := e.payload.(map[string]any); ok && e.name == "browser:frame" && m["width"] == 800 && m["height"] == 500 {
					return true
				}
			}
			return false
		})
	})

	if err := h.app.BrowserView(s.ID, false); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	stopped := h.frames(s.ID)
	for i := 10; i < 15; i++ {
		flip(i)
		time.Sleep(60 * time.Millisecond)
	}
	time.Sleep(500 * time.Millisecond)
	if h.frames(s.ID) != stopped {
		t.Errorf("frames kept coming while hidden: %d -> %d", stopped, h.frames(s.ID))
	}

	t.Run("input", func(t *testing.T) {
		click := func(x, y float64) {
			for _, action := range []string{"move", "down", "up"} {
				if err := h.app.BrowserInput(s.ID, BrowserInput{Type: "mouse", Action: action, X: x, Y: y, Button: "left", ClickCount: 1}); err != nil {
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
		for _, k := range []BrowserInput{
			{Key: "h", Code: "KeyH", Text: "h"},
			{Key: "i", Code: "KeyI", Text: "i"},
			{Key: "!", Code: "Digit1", Text: "!", Modifiers: modShift},
			{Key: "Backspace", Code: "Backspace"},
		} {
			for _, action := range []string{"down", "up"} {
				k.Type, k.Action = "key", action
				if err := h.app.BrowserInput(s.ID, k); err != nil {
					t.Fatal(err)
				}
			}
		}
		if got := eval("document.getElementById('q').value"); got != `"hi"` {
			t.Errorf("typed value = %s", got)
		}
		if err := h.app.BrowserInput(s.ID, BrowserInput{Type: "paste", Text: " pasted"}); err != nil {
			t.Fatal(err)
		}
		for _, action := range []string{"down", "up"} {
			if err := h.app.BrowserInput(s.ID, BrowserInput{Type: "key", Action: action, Key: "Enter", Code: "Enter"}); err != nil {
				t.Fatal(err)
			}
		}
		eventually(t, "Enter reaching the page", func() bool { return eval("document.getElementById('entered').textContent") == `"enter:hi pasted"` })

		if err := h.app.BrowserInput(s.ID, BrowserInput{Type: "wheel", X: 400, Y: 300, DeltaY: 700}); err != nil {
			t.Fatal(err)
		}
		eventually(t, "the wheel scrolling the page", func() bool { return eval("Math.round(scrollY)") == "700" })
		if err := h.app.BrowserInput(s.ID, BrowserInput{Type: "nope"}); err == nil {
			t.Error("an unknown input type was accepted")
		}
	})

	t.Run("user screenshot", func(t *testing.T) {
		before := h.rec.count("attention")
		item, err := h.app.BrowserScreenshot(s.ID, "mine")
		if err != nil || item.Source != "user" || item.Caption != "mine" {
			t.Fatalf("BrowserScreenshot = %+v, %v", item, err)
		}
		if h.rec.count("attention") != before {
			t.Error("the user's own screenshot raised attention")
		}
		if err := h.app.DeleteEvidence(s.ID, item.ID); err != nil || len(h.app.Evidence(s.ID)) != 0 {
			t.Errorf("DeleteEvidence: %v", err)
		}
	})
}

func decodePNG(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(data))
}

func TestStrayBrowserIsReplaced(t *testing.T) {
	dir, err := os.MkdirTemp("", "aosb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	first := newBrowsers(dir, func(string, any) {})
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
	second := newBrowsers(dir, func(string, any) {})
	defer second.CloseAll()
	if _, err := second.Open(ctx, "p/1", "about:blank"); err != nil {
		t.Fatalf("the restarted app could not get a browser: %v", err)
	}
}
