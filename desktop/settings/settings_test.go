package settings_test

import (
	"os"
	"strings"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/internal/project"
)

func TestMain(m *testing.M) { os.Exit(apptest.Main(m)) }

func TestTheme(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{TopExtra: "# mine\n"})
	if got := h.Settings().Theme; got != "system" {
		t.Fatalf("default theme = %q", got)
	}

	for _, theme := range []string{"dark", "light"} {
		if got, err := h.SetTheme(theme); err != nil || got.Theme != theme {
			t.Fatalf("SetTheme(%s) = %+v, %v", theme, got, err)
		}
		if cfg, err := project.Load(h.Conf); err != nil || cfg.Theme != theme {
			t.Errorf("config after %s = %+v, %v", theme, cfg, err)
		}
	}
	if data, _ := os.ReadFile(h.Conf); !strings.Contains(string(data), "# mine") || !strings.Contains(string(data), "theme: light") {
		t.Errorf("config lost its comment or the theme:\n%s", data)
	}

	if got, err := h.SetTheme("system"); err != nil || got.Theme != "system" {
		t.Fatalf("SetTheme(system) = %+v, %v", got, err)
	}
	if data, _ := os.ReadFile(h.Conf); strings.Contains(string(data), "theme") {
		t.Errorf("system left the key in the file:\n%s", data)
	}

	if _, err := h.SetTheme("sepia"); err == nil {
		t.Error("an unknown theme was accepted")
	}
	if got := h.Settings().Theme; got != "system" {
		t.Errorf("a rejected theme changed the setting to %q", got)
	}
}

func TestThemeSurvivesRestart(t *testing.T) {
	h := apptest.New(t)
	if _, err := h.SetTheme("dark"); err != nil {
		t.Fatal(err)
	}
	if got := h.Restart(t).Settings().Theme; got != "dark" {
		t.Errorf("theme after restart = %q", got)
	}
}

func TestCleanup(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    # keep me\n    url: http://localhost\n"})
	if got := h.Settings().Cleanup; got != (project.Cleanup{Merge: "auto", Close: "manual"}) {
		t.Fatalf("default cleanup = %+v", got)
	}
	if got, err := h.SetCleanup("close", "auto"); err != nil || got.Cleanup != (project.Cleanup{Merge: "auto", Close: "auto"}) {
		t.Fatalf("SetCleanup(close, auto) = %+v, %v", got, err)
	}
	if got, err := h.SetCleanup("merge", "manual"); err != nil || got.Cleanup != (project.Cleanup{Merge: "manual", Close: "auto"}) {
		t.Fatalf("SetCleanup(merge, manual) = %+v, %v", got, err)
	}
	cfg, err := project.Load(h.Conf)
	if err != nil || cfg.Projects[0].Cleanup != (project.Cleanup{Merge: "manual", Close: "auto"}) {
		t.Errorf("config = %+v, %v", cfg.Projects, err)
	}
	if data, _ := os.ReadFile(h.Conf); !strings.Contains(string(data), "# keep me") {
		t.Errorf("config lost the project's comment:\n%s", data)
	}
	for _, bad := range [][2]string{{"merge", "sometimes"}, {"open", "auto"}} {
		if _, err := h.SetCleanup(bad[0], bad[1]); err == nil {
			t.Errorf("SetCleanup(%s, %s) was accepted", bad[0], bad[1])
		}
	}
}
