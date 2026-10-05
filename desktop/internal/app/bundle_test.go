package app

import (
	"os"
	"testing"

	"github.com/nednella/agentos/internal/bus"
)

func TestBundleOf(t *testing.T) {
	for exe, want := range map[string]string{
		"/Applications/agentos.app/Contents/MacOS/agentos": "/Applications/agentos.app",
		"/repo/bin/agentos.app/Contents/MacOS/agentos":     "/repo/bin/agentos.app",
		"/repo/bin/agentos-desktop":                        "",
		"/agentos":                                         "",
	} {
		if got := bundleOf(exe); got != want {
			t.Errorf("bundleOf(%q) = %q, want %q", exe, got, want)
		}
	}
}

func TestRecordBundle(t *testing.T) {
	dir := t.TempDir()
	if err := recordBundleOf("/repo/bin/agentos-desktop", dir); err != nil || exists(bus.AppPath(dir)) {
		t.Errorf("a binary outside a bundle recorded a path: %v", err)
	}
	if err := recordBundleOf("/Apps/agentos.app/Contents/MacOS/agentos", dir); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(bus.AppPath(dir)); string(got) != "/Apps/agentos.app\n" {
		t.Errorf("recorded %q", got)
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
