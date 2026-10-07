package browser_test

import (
	"bytes"
	"errors"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
	ctl "github.com/nednella/agentos/internal/control"
)

// browserCmd runs an agentos browser command the way the cli sends it: flags become options.
func browserCmd(t *testing.T, h *apptest.Harness, session string, args ...string) (string, error) {
	t.Helper()
	var pos []string
	opts := map[string]string{}
	for i := 0; i < len(args); i++ {
		switch a := args[i]; a {
		case "--caption":
			i++
			opts[a[2:]] = args[i]
		case "--full", "--front":
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

func profileRunning(profile string) bool {
	return exec.Command("pgrep", "-f", profile).Run() == nil
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
	if out := mustBrowser(t, h, s.ID, "open", srv.URL+"/second", "--front"); !strings.Contains(out, "Second") {
		t.Errorf("open --front printed %q", out)
	}

	t.Run("screenshot becomes evidence", func(t *testing.T) {
		mustBrowser(t, h, s.ID, "open", srv.URL)
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
		data, err = os.ReadFile(full)
		if err != nil {
			t.Fatal(err)
		}
		if img, err := png.Decode(bytes.NewReader(data)); err != nil || img.Bounds().Dy() < 3000 {
			t.Errorf("full page screenshot: %v %v", img, err)
		}
	})

	t.Run("errors", func(t *testing.T) {
		for _, args := range [][]string{
			{"open"},
			{"open", "javascript:alert(1)"},
			{"open", "http://127.0.0.1:1"},
			{"frobnicate"},
			{"click", "e1"},
			{"snapshot"},
			{"eval", "1"},
		} {
			if out, err := browserCmd(t, h, s.ID, args...); err == nil {
				t.Errorf("agentos browser %s succeeded: %q", strings.Join(args, " "), out)
			}
		}
		if _, err := browserCmd(t, h, s.ID, "click", "e1"); err == nil || !strings.Contains(err.Error(), "agentos browser help") {
			t.Errorf("a removed command's error = %v", err)
		}
		if out, err := browserCmd(t, h, "", "open", srv.URL); err == nil || !strings.Contains(err.Error(), "AGENTOS_SESSION") {
			t.Errorf("without a session: %q, %v", out, err)
		}
		if out := mustBrowser(t, h, "", "help"); !strings.Contains(out, "list_pages") || strings.Contains(out, "snapshot") {
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

func TestBrowserWithoutArgumentsShowsTheView(t *testing.T) {
	h := newHarness(t)
	resp := h.Ask(t, ctl.Request{Cmd: "browser"})
	if got := h.Rec.LastUI(); !resp.OK || resp.Out != "ok" || got.Name != "browser" || len(got.Args) != 0 {
		t.Errorf("browser = %+v, ui %+v", resp, got)
	}
	if help := h.Ask(t, ctl.Request{Cmd: "browser", Args: []string{"help"}}); !help.OK || !strings.Contains(help.Out, "your session's browser window") {
		t.Errorf("browser help = %+v", help)
	}
}
