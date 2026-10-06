package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nednella/agentos/internal/atomicfile"
	"github.com/nednella/agentos/internal/bus"
)

// Bundle is the .app the app runs from, or "" when it runs from none.
func Bundle() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return bundleOf(exe)
}

// RecordBundle writes the path of the .app bundle the app runs from to the state dir, where the
// agentos command looks for it. An app that runs from no bundle records nothing.
func RecordBundle(stateDir string) error {
	if _, err := os.Executable(); err != nil {
		return fmt.Errorf("finding the app: %w", err)
	}
	return recordBundleOf(Bundle(), stateDir)
}

func recordBundleOf(bundle, stateDir string) error {
	if bundle == "" {
		return nil
	}
	return atomicfile.Write(bus.AppPath(stateDir), []byte(bundle+"\n"), 0o600)
}

// bundleOf is the .app folder that holds exe, or "".
func bundleOf(exe string) string {
	for dir := filepath.Dir(exe); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if strings.HasSuffix(dir, ".app") {
			return dir
		}
	}
	return ""
}
