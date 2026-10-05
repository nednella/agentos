package browser_test

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
	ctl "github.com/nednella/agentos/internal/control"
)

// browserCmd runs an agentos browser command the way the cli sends it: flags become options.
func browserCmd(t *testing.T, h *apptest.Harness, session string, args ...string) (string, error) {
	t.Helper()
	if len(args) > 0 && args[0] == "browser" {
		args = args[1:]
	}
	var pos []string
	opts := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--caption", "--timeout":
			i++
			opts[a[2:]] = args[i]
		case "--full", "--append":
			opts[a[2:]] = "1"
		default:
			pos = append(pos, a)
		}
	}
	resp := h.Ask(t, ctl.Request{Cmd: "browser", Session: session, Args: pos, Opts: opts, TimeoutMs: 60_000})
	if !resp.OK {
		return "", errors.New(resp.Error)
	}
	return strings.TrimSpace(resp.Out), nil
}

func mustBrowser(t *testing.T, h *apptest.Harness, session string, args ...string) string {
	t.Helper()
	out, err := browserCmd(t, h, session, args...)
	if err != nil {
		t.Fatalf("agentos browser %s: %v", strings.Join(args, " "), err)
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

func profileRunning(profile string) bool {
	return exec.Command("pgrep", "-f", profile).Run() == nil
}

func decodePNG(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return png.Decode(bytes.NewReader(data))
}

func TestBrowserAgentCLI(t *testing.T) {
	h, s := browserHarness(t)
	srv := demoServer(t)
	profile := filepath.Join(h.State, "data", "main", "browser")

	out := mustBrowser(t, h, s.ID, "open", srv.URL)
	if !strings.Contains(out, "Demo") || !strings.Contains(out, srv.URL) {
		t.Errorf("open printed %q", out)
	}
	if got, _ := h.Session(s.ID); !got.Browser {
		t.Error("the session does not show its browser")
	}

	snap := mustBrowser(t, h, s.ID, "snapshot")
	for _, want := range []string{"# Demo page", `textbox "Email"`, `placeholder="you@example.com"`, `select "Color"`, `value="Red"`, "options=Red | Green | Blue",
		`checkbox "Subscribe"`, `button "Save"`, `button "Nope" disabled`, `textbox "Search"`, `link "Second page" -> /second`, "not saved"} {
		if !strings.Contains(snap, want) {
			t.Errorf("snapshot lacks %q:\n%s", want, snap)
		}
	}

	email, color, save, sub := refOf(t, snap, "textbox", "Email"), refOf(t, snap, "select", "Color"), refOf(t, snap, "button", "Save"), refOf(t, snap, "checkbox", "Subscribe")
	mustBrowser(t, h, s.ID, "type", email, "ned@example.com")
	mustBrowser(t, h, s.ID, "select", color, "Green")
	mustBrowser(t, h, s.ID, "click", sub)
	mustBrowser(t, h, s.ID, "click", save)
	mustBrowser(t, h, s.ID, "wait-for", "Saved: ned@example.com / Green")
	mustBrowser(t, h, s.ID, "type", email, "new", "--append")
	if v := mustBrowser(t, h, s.ID, "eval", "document.getElementById('email').value + '|' + document.getElementById('sub').checked"); v != `"ned@example.comnew|true"` {
		t.Errorf("form state = %s", v)
	}
	mustBrowser(t, h, s.ID, "type", email, "fresh")
	if v := mustBrowser(t, h, s.ID, "eval", "document.getElementById('email').value"); v != `"fresh"` {
		t.Errorf("replaced value = %s", v)
	}

	q := refOf(t, snap, "textbox", "Search")
	mustBrowser(t, h, s.ID, "type", q, "hello")
	mustBrowser(t, h, s.ID, "press", "Enter")
	mustBrowser(t, h, s.ID, "wait-for", "enter:hello")

	if text := mustBrowser(t, h, s.ID, "text", save); text != "Save" {
		t.Errorf("text of the button = %q", text)
	}
	if out := mustBrowser(t, h, s.ID, "scroll", "down", "1000"); !strings.Contains(out, "scrolled to 1000") {
		t.Errorf("scroll = %q", out)
	}
	mustBrowser(t, h, s.ID, "scroll", "up", "5000")
	mustBrowser(t, h, s.ID, "hover", save)

	t.Run("console", func(t *testing.T) {
		mustBrowser(t, h, s.ID, "click", refOf(t, snap, "button", "Boom"))
		mustBrowser(t, h, s.ID, "click", refOf(t, snap, "button", "Fetch"))
		var out string
		eventually(t, "both console lines", func() bool {
			out += "\n" + mustBrowser(t, h, s.ID, "console")
			return strings.Contains(out, "console.error: boom happened") && strings.Contains(out, "/nope -> 404")
		})
		if again := mustBrowser(t, h, s.ID, "console"); !strings.Contains(again, "no console errors") {
			t.Errorf("console after reading = %q", again)
		}
	})

	t.Run("target blank link stays in the tab", func(t *testing.T) {
		mustBrowser(t, h, s.ID, "click", refOf(t, snap, "link", "Second page"))
		mustBrowser(t, h, s.ID, "wait-for", "Second page header")
		if got := mustBrowser(t, h, s.ID, "url"); got != srv.URL+"/second\nSecond" {
			t.Errorf("url = %q", got)
		}
		eventually(t, "back flag", func() bool { return h.BrowserState(s.ID).CanGoBack })
		if got := mustBrowser(t, h, s.ID, "back"); !strings.Contains(got, srv.URL+"/") {
			t.Errorf("back = %q", got)
		}
		if got := mustBrowser(t, h, s.ID, "reload"); !strings.Contains(got, "Demo") {
			t.Errorf("reload = %q", got)
		}
	})

	t.Run("screenshot becomes evidence", func(t *testing.T) {
		before := h.Rec.Count("attention")
		path := mustBrowser(t, h, s.ID, "screenshot", "--caption", "the demo page")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("the printed path %q: %v", path, err)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil || img.Bounds().Dx() != 1280 || img.Bounds().Dy() != 800 {
			t.Errorf("screenshot = %v, %v", img, err)
		}
		items := h.Evidence(s.ID)
		if len(items) != 1 || items[0].Caption != "the demo page" || items[0].Source != "agent" || items[0].Kind != "image" ||
			!strings.HasPrefix(items[0].URL, "/media/main/evidence/") {
			t.Fatalf("evidence = %+v", items)
		}
		srvMedia := httptest.NewServer(h.App.Media())
		defer srvMedia.Close()
		if resp, err := http.Get(srvMedia.URL + items[0].URL); err != nil || resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "image/png" {
			t.Errorf("serving the screenshot: %v %v", resp, err)
		}
		if got, _ := h.Session(s.ID); got.Evidence != 1 {
			t.Errorf("session evidence count = %d", got.Evidence)
		}
		if h.Rec.Count("attention") != before+1 || h.Rec.Count("evidence") == 0 {
			t.Error("no attention or evidence event for the agent's screenshot")
		}

		full := mustBrowser(t, h, s.ID, "screenshot", "--full")
		if img, err := decodePNG(full); err != nil || img.Bounds().Dy() < 3000 {
			t.Errorf("full page screenshot: %v %v", img, err)
		}
		save = refOf(t, mustBrowser(t, h, s.ID, "snapshot"), "button", "Save")
		element := mustBrowser(t, h, s.ID, "screenshot", save)
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
			if out, err := browserCmd(t, h, s.ID, args...); err == nil {
				t.Errorf("agentos %s succeeded: %q", strings.Join(args, " "), out)
			}
		}
		if _, err := browserCmd(t, h, s.ID, "browser", "click", "e999"); err == nil || !strings.Contains(err.Error(), "run snapshot again") {
			t.Errorf("unknown ref error = %v", err)
		}
		if out, err := browserCmd(t, h, "", "browser", "url"); err == nil || !strings.Contains(err.Error(), "AGENTOS_SESSION") {
			t.Errorf("without a session: %q, %v", out, err)
		}
		if out := mustBrowser(t, h, "", "help"); !strings.Contains(out, "snapshot") {
			t.Errorf("help = %q", out)
		}
	})

	t.Run("closing the tab ends the browser", func(t *testing.T) {
		h.App.Browsers().SetGrace(300 * time.Millisecond)
		h.BrowserClose(s.ID)
		if got, _ := h.Session(s.ID); got.Browser {
			t.Error("the session still shows a browser")
		}
		if st := h.BrowserState(s.ID); st.Open {
			t.Errorf("state = %+v", st)
		}
		eventually(t, "the browser to stop", func() bool { return !profileRunning(profile) })
	})
}
