package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/internal/bus"
)

func TestBareAgentosOpensTheFirstAppItFinds(t *testing.T) {
	root := t.TempDir()
	mkApp := func(dir string) string {
		t.Helper()
		path := filepath.Join(root, dir, appBundle)
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
		return path
	}
	first, second := mkApp("one"), mkApp("two")
	tests := []struct {
		name   string
		places []string
		want   string
		errHas string
	}{
		{"the first that exists", []string{filepath.Join(root, "none", appBundle), first, second}, first, ""},
		{"a file is not an app", []string{filepath.Join(root, "file"), second}, second, ""},
		{"none exists", []string{filepath.Join(root, "none", appBundle), filepath.Join(root, "other", appBundle)}, "", "agentos.app not found; looked in:\n  " + filepath.Join(root, "none", appBundle) + "\n  " + filepath.Join(root, "other", appBundle)},
	}
	if err := os.WriteFile(filepath.Join(root, "file"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opened []string
			cmd := newRootCmdWith(func(_ context.Context, path string) error { opened = append(opened, path); return nil },
				func() []string { return tt.places })
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(nil)
			err := cmd.Execute()
			switch {
			case tt.errHas != "":
				if err == nil || err.Error() != tt.errHas || len(opened) != 0 {
					t.Errorf("err = %v, opened %v", err, opened)
				}
			case err != nil || !slices.Equal(opened, []string{tt.want}):
				t.Errorf("err = %v, opened %v, want %s", err, opened, tt.want)
			}
		})
	}
}

func TestAppPlacesIncludeTheRecordedPathLast(t *testing.T) {
	state := t.TempDir()
	t.Setenv("AGENTOS_STATE_DIR", state)
	if err := os.WriteFile(bus.AppPath(state), []byte("/opt/dev/agentos.app\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	places := appPlaces()
	if len(places) < 3 || places[len(places)-1] != "/opt/dev/agentos.app" || places[len(places)-2] != "/Applications/agentos.app" {
		t.Errorf("places = %q", places)
	}
	if !slices.ContainsFunc(places, func(p string) bool {
		return strings.HasSuffix(p, "/Applications/agentos.app") && p != "/Applications/agentos.app"
	}) {
		t.Errorf("no ~/Applications in %q", places)
	}

	if err := os.Remove(bus.AppPath(state)); err != nil {
		t.Fatal(err)
	}
	if places = appPlaces(); places[len(places)-1] != "/Applications/agentos.app" {
		t.Errorf("without a record, places = %q", places)
	}
}

func TestBesideBundle(t *testing.T) {
	for exe, want := range map[string]string{
		"/opt/x/agentos.app/Contents/MacOS/agentos": "/opt/x",
		"/usr/local/bin/agentos":                    "/usr/local/bin",
		"/agentos":                                  "/",
	} {
		if got := besideBundle(exe); got != want {
			t.Errorf("besideBundle(%q) = %q, want %q", exe, got, want)
		}
	}
}
