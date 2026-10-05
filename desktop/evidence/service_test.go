package evidence_test

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ctl "github.com/nednella/agentos/internal/control"
)

var onePixelPNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")

func TestShowFilesEvidence(t *testing.T) {
	h := newHarness(t)
	s, err := h.NewSession("show", "")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	img, note, bin := filepath.Join(dir, "shot.png"), filepath.Join(dir, "notes.txt"), filepath.Join(dir, "blob.bin")
	for path, data := range map[string][]byte{img: onePixelPNG, note: []byte("all green\nno regressions\n"), bin: {0xff, 0xfe, 0x00}} {
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	show := func(args []string, opts map[string]string) ctl.Response {
		return h.Ask(t, ctl.Request{Cmd: "show", Session: s.ID, Args: args, Opts: opts})
	}
	before := h.Rec.CountAttention("evidence")
	for _, resp := range []ctl.Response{show([]string{img}, map[string]string{"caption": "pixel"}), show([]string{note}, nil), show(nil, map[string]string{"text": "ran 40 tests", "caption": "tests"})} {
		if !resp.OK {
			t.Fatalf("response = %+v", resp)
		}
	}
	items := h.Evidence(s.ID)
	if len(items) != 3 {
		t.Fatalf("evidence = %+v", items)
	}
	if items[0].Kind != "image" || items[0].Caption != "pixel" || items[0].Source != "agent" || !strings.HasPrefix(items[0].URL, "/media/main/evidence/") {
		t.Errorf("image = %+v", items[0])
	}
	if items[1].Kind != "text" || items[1].Text != "all green\nno regressions\n" || items[1].Caption != "notes.txt" || items[1].URL != "" {
		t.Errorf("text file = %+v", items[1])
	}
	if items[2].Text != "ran 40 tests" || items[2].Caption != "tests" {
		t.Errorf("text card = %+v", items[2])
	}
	if got := h.Rec.CountAttention("evidence"); got != before+3 {
		t.Errorf("attention events = %d, want %d", got, before+3)
	}
	if got, _ := h.Session(s.ID); got.Evidence != 3 {
		t.Errorf("session evidence count = %d", got.Evidence)
	}
	for _, bad := range []ctl.Request{
		{Cmd: "show", Session: s.ID, Args: []string{bin}},
		{Cmd: "show", Session: s.ID, Args: []string{filepath.Join(dir, "missing.png")}},
		{Cmd: "show", Session: s.ID},
		{Cmd: "show", Args: []string{img}},
	} {
		if h.Ask(t, bad).OK {
			t.Errorf("%+v succeeded", bad)
		}
	}

	t.Run("the pictures are served", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h.App.Media().ServeHTTP(rec, httptest.NewRequest("GET", items[0].URL, nil))
		if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("status %d, headers %v", rec.Code, rec.Header())
		}
		for _, p := range []string{"/media/main/evidence/../notes.json", "/media/main/evidence/1/missing.png", "/media/main/evidence/"} {
			rec := httptest.NewRecorder()
			h.App.Media().ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
			if rec.Code != http.StatusNotFound {
				t.Errorf("GET %s = %d", p, rec.Code)
			}
		}
	})

	t.Run("deleting and dismissing", func(t *testing.T) {
		if err := h.DeleteEvidence(s.ID, items[0].ID); err != nil || len(h.Evidence(s.ID)) != 2 {
			t.Errorf("DeleteEvidence: %v", err)
		}
		if err := h.KillSession(s.ID); err != nil {
			t.Fatal(err)
		}
		eventually(t, "the session to show as ended", func() bool { got, _ := h.Session(s.ID); return got.State == "ended" })
		if len(h.Evidence(s.ID)) != 2 {
			t.Error("evidence went with the ended session; it should stay until dismissed")
		}
		if err := h.DismissSession(s.ID); err != nil {
			t.Fatal(err)
		}
		eventually(t, "evidence removed with the dismissed session", func() bool { return len(h.Evidence(s.ID)) == 0 })
	})
}
