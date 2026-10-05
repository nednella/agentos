package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/bus"
)

const appBundle = "agentos.app"

// opener opens the app bundle at path.
type opener func(ctx context.Context, path string) error

func openWithMac(ctx context.Context, path string) error {
	if out, err := exec.CommandContext(ctx, "open", "-a", path).CombinedOutput(); err != nil {
		return fmt.Errorf("opening %s: %w: %s", path, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// openApp is what bare "agentos" does: open the desktop app.
func openApp(open opener, appPlaces func() []string) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		places := appPlaces()
		for _, place := range places {
			if info, err := os.Stat(place); err == nil && info.IsDir() {
				return open(cmd.Context(), place)
			}
		}
		return fmt.Errorf("%s not found; looked in:\n  %s", appBundle, strings.Join(places, "\n  "))
	}
}

// appPlaces are where the app may be, in the order to look: next to this command (or the bundle it
// sits in), the user's and the system's Applications folders, then where the app last said it ran from.
func appPlaces() []string {
	var places []string
	if exe, err := os.Executable(); err == nil {
		if real, err := filepath.EvalSymlinks(exe); err == nil {
			exe = real
		}
		places = append(places, filepath.Join(besideBundle(exe), appBundle), filepath.Join(filepath.Dir(exe), appBundle))
	}
	if home, err := os.UserHomeDir(); err == nil {
		places = append(places, filepath.Join(home, "Applications", appBundle))
	}
	places = append(places, filepath.Join("/Applications", appBundle))
	if dir, err := bus.DefaultDir(); err == nil {
		if data, err := os.ReadFile(bus.AppPath(dir)); err == nil {
			if recorded := strings.TrimSpace(string(data)); recorded != "" {
				places = append(places, recorded)
			}
		}
	}
	return places
}

// besideBundle is the folder that holds the .app bundle exe sits in, or exe's own folder when it sits in none.
func besideBundle(exe string) string {
	for dir := filepath.Dir(exe); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if strings.HasSuffix(dir, ".app") {
			return filepath.Dir(dir)
		}
	}
	return filepath.Dir(exe)
}
