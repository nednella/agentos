package browser

import (
	"context"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStrayBrowserIsReplaced(t *testing.T) {
	dir, err := os.MkdirTemp("", "aosb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	first := New(dir, func(string, any) {})
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
	second := New(dir, func(string, any) {})
	defer second.CloseAll()
	if _, err := second.Open(ctx, "p/1", "about:blank"); err != nil {
		t.Fatalf("the restarted app could not get a browser: %v", err)
	}
}

func TestNoBrowserFound(t *testing.T) {
	b := New(t.TempDir(), func(string, any) {})
	b.binary = ""
	if b.Available() {
		t.Error("available without a browser")
	}
	st, err := b.Open(context.Background(), "p/1", "")
	if err == nil || !strings.Contains(st.Error, "no Brave") {
		t.Errorf("Open = %+v, %v", st, err)
	}
	if s := b.State("p/1"); s.Open || s.ID != "p/1" {
		t.Errorf("state = %+v", s)
	}
}

func TestLaunchArgsPerMode(t *testing.T) {
	b := New(t.TempDir(), func(string, any) {})
	b.headless = true
	if args := b.launchArgs("/p"); !slices.Contains(args, "--headless=new") || !slices.Contains(args, "--hide-scrollbars") {
		t.Errorf("headless args = %v", args)
	}
	b.headless = false
	args := b.launchArgs("/p")
	for _, want := range []string{"--user-data-dir=/p", "--disable-backgrounding-occluded-windows", "--disable-renderer-backgrounding", "--no-startup-window"} {
		if !slices.Contains(args, want) {
			t.Errorf("headed args lack %s: %v", want, args)
		}
	}
	if slices.Contains(args, "--headless=new") || slices.Contains(args, "--hide-scrollbars") {
		t.Errorf("headed args = %v", args)
	}
}

func TestWebAddress(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"example.com", "https://example.com", true},
		{"example.com/a?b=1", "https://example.com/a?b=1", true},
		{"https://example.com", "https://example.com", true},
		{"localhost:3000", "http://localhost:3000", true},
		{"127.0.0.1:8080/x", "http://127.0.0.1:8080/x", true},
		{"about:blank", "about:blank", true},
		{"  spaced.dev  ", "https://spaced.dev", true},
		{"javascript:alert(1)", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		got, err := webAddress(tt.in)
		if (err == nil) != tt.ok || got != tt.want {
			t.Errorf("webAddress(%q) = %q, %v", tt.in, got, err)
		}
	}
}

func newTestBrowsers(t *testing.T) (*Browsers, context.Context) {
	t.Helper()
	b := New(t.TempDir(), func(string, any) {})
	if !b.Available() {
		t.Skip("no Chromium based browser installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	t.Cleanup(func() { cancel(); b.CloseAll() })
	return b, ctx
}

func TestConcurrentOpensShareOneTab(t *testing.T) {
	b, ctx := newTestBrowsers(t)
	var wg sync.WaitGroup
	states := make([]BrowserState, 2)
	for i := range states {
		wg.Go(func() {
			st, err := b.Open(ctx, "p/1", "about:blank")
			if err != nil {
				t.Errorf("Open %d: %v", i, err)
			}
			states[i] = st
		})
	}
	wg.Wait()
	if !states[0].Open || !states[1].Open || states[0].ID != states[1].ID {
		t.Errorf("the two opens returned %+v and %+v", states[0], states[1])
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.tabs) != 1 || b.procs["p"] == nil || b.procs["p"].tabs != 1 {
		t.Errorf("%d tabs, browser counts %d", len(b.tabs), b.procs["p"].tabs)
	}
}

func TestOpenReportsAFirstPageThatFails(t *testing.T) {
	b, ctx := newTestBrowsers(t)
	st, err := b.Open(ctx, "p/1", "http://127.0.0.1:1/")
	if err == nil || !strings.Contains(err.Error(), "could not open") {
		t.Errorf("Open of an address nobody serves = %v", err)
	}
	if !st.Open || !b.Has("p/1") {
		t.Errorf("the tab did not stay open: %+v", st)
	}
}
