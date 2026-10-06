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
	if data, _ := os.ReadFile(h.Conf); !strings.Contains(string(data), "# mine") || !strings.Contains(string(data), "app_theme: light") {
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
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    # keep me\n    browser_start_url: http://localhost\n"})
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
	if err != nil || cfg.Projects[0].CleanupMode != (project.Cleanup{Merge: "manual", Close: "auto"}) {
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

func TestTextScale(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{TopExtra: "# mine\n"})
	if got := h.Settings().TextScale; got != 1 {
		t.Fatalf("default text scale = %v", got)
	}
	if got, err := h.SetTextScale(1.2); err != nil || got.TextScale != 1.2 {
		t.Fatalf("SetTextScale(1.2) = %+v, %v", got, err)
	}
	if data, _ := os.ReadFile(h.Conf); !strings.Contains(string(data), "# mine") || !strings.Contains(string(data), "app_text_scale: 1.2") {
		t.Errorf("config lost its comment or the scale:\n%s", data)
	}
	if got := h.Restart(t).Settings().TextScale; got != 1.2 {
		t.Errorf("scale after restart = %v", got)
	}
	if got, err := h.SetTextScale(1); err != nil || got.TextScale != 1 {
		t.Fatalf("SetTextScale(1) = %+v, %v", got, err)
	}
	if data, _ := os.ReadFile(h.Conf); strings.Contains(string(data), "app_text_scale") {
		t.Errorf("the default left the key in the file:\n%s", data)
	}
	for _, bad := range []float64{0, 0.5, 2, -1} {
		if _, err := h.SetTextScale(bad); err == nil {
			t.Errorf("SetTextScale(%v) was accepted", bad)
		}
	}
}

func TestKeepAwake(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{TopExtra: "keep_mac_awake: true # mine\n"})
	if !h.Settings().KeepAwake {
		t.Fatal("keep awake is off by default")
	}
	if got, err := h.SetKeepAwake(false); err != nil || got.KeepAwake {
		t.Fatalf("SetKeepAwake(false) = %+v, %v", got, err)
	}
	if data, _ := os.ReadFile(h.Conf); !strings.Contains(string(data), "keep_mac_awake: false # mine") {
		t.Errorf("config lost its comment or the value:\n%s", data)
	}
	if cfg, err := project.Load(h.Conf); err != nil || cfg.KeepAwake == nil || *cfg.KeepAwake {
		t.Errorf("config = %+v, %v", cfg, err)
	}
	if h.Restart(t).Settings().KeepAwake {
		t.Error("keep awake came back on after a restart")
	}
}

func TestBrowserEnabled(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    # keep me\n    browser_start_url: http://localhost\n"})
	if !h.Settings().Browser {
		t.Fatal("the browser is off by default")
	}
	if got, err := h.SetBrowserEnabled(false); err != nil || got.Browser {
		t.Fatalf("SetBrowserEnabled(false) = %+v, %v", got, err)
	}
	if h.App.Sessions().Current().BrowserOn() {
		t.Error("the current project still has the browser on")
	}
	if data, _ := os.ReadFile(h.Conf); !strings.Contains(string(data), "# keep me") || !strings.Contains(string(data), "browser_enabled: false") {
		t.Errorf("config lost the comment or the value:\n%s", data)
	}
	if h.Restart(t).Settings().Browser {
		t.Error("the browser came back on after a restart")
	}
}

func TestDigestSchedule(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    # keep me\n    browser_start_url: http://localhost\n"})
	if got := h.Settings().Digest; got != "weekly" {
		t.Fatalf("default digest schedule = %q", got)
	}
	if got, err := h.SetDigestSchedule("off"); err != nil || got.Digest != "off" {
		t.Fatalf("SetDigestSchedule(off) = %+v, %v", got, err)
	}
	if h.App.Sessions().Current().DigestOn() {
		t.Error("the current project still runs its digest")
	}
	if data, _ := os.ReadFile(h.Conf); !strings.Contains(string(data), "# keep me") || !strings.Contains(string(data), "digest_schedule: off") {
		t.Errorf("config lost the comment or the value:\n%s", data)
	}
	if got, err := h.SetDigestSchedule("weekly"); err != nil || got.Digest != "weekly" {
		t.Fatalf("SetDigestSchedule(weekly) = %+v, %v", got, err)
	}
	if _, err := h.SetDigestSchedule("daily"); err == nil {
		t.Error("an unknown schedule was accepted")
	}
	if got := h.Settings().Digest; got != "weekly" {
		t.Errorf("a rejected schedule changed the setting to %q", got)
	}
}
